package triggers

import (
	"fmt"
	"net/url"
	"sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	sm "sqldb-ws/domain/schema/models"
	"sqldb-ws/domain/utils"
	"sqldb-ws/infrastructure/connector"
	db "sqldb-ws/infrastructure/connector/db"
	"strconv"
	"strings"
)

func (t *TriggerService) GetViewMailTriggers(record utils.Record, fromSchema *sm.SchemaModel, triggerDesc string, triggerName string, triggerID, toSchemaID, destID int64) ([]sm.ManualTriggerModel, error) {
	if sch, err := schema.GetSchema(ds.DBEmailSended.Name); err != nil {
		return nil, err
	} else {
		mails := t.TriggerManualMail("manual", record, fromSchema, triggerID, toSchemaID, destID)
		bodies := []sm.ManualTriggerModel{}
		s := sch.ToMapRecord()
		for _, f := range sch.Fields {
			if f.GetLink() > 0 {
				if sch2, err := schema.GetSchemaByID(f.GetLink()); err == nil {
					s[f.Name].(map[string]interface{})["action_path"] = utils.BuildPath(sch2.Name, utils.ReservedParam, utils.RootShallow+"=enable")
					for _, f2 := range sch2.Fields {
						if f2.GetLink() > 0 && strings.Contains(f2.Name, "_id") && !strings.Contains(f2.Name, sch2.Name) {
							if sch3, err := schema.GetSchemaByID(f2.GetLink()); err == nil {
								s[f.Name].(map[string]interface{})["data_schema"] = sch2.ToMapRecord()
								s[f.Name].(map[string]interface{})["values_path"] = utils.BuildPath(sch3.Name, utils.ReservedParam, utils.RootShallow+"=enable")
							}
						}
					}
				}
			}
			if strings.Contains(f.Type, "upload") {
				s[f.Name].(map[string]interface{})["action_path"] = fmt.Sprintf("/%s/%s/import?rows=all&columns=%s", utils.MAIN_PREFIX, sch.Name, f.Name)
				s[f.Name].(map[string]interface{})["values_path"] = fmt.Sprintf("/%s/%s/import?rows=all&columns=%s", utils.MAIN_PREFIX, sch.Name, f.Name)
			}
		}
		for _, m := range mails {
			if utils.GetBool(m, "force_file_attached") {
				utils.ToMap(s["file_attached"])["required"] = true
			}
			delete(m, "force_file_attached")
			bodies = append(bodies, sm.ManualTriggerModel{
				Name:        triggerName,
				Description: triggerDesc,
				Type:        "mail",
				Schema:      s,
				Body:        m,
				ActionPath:  utils.BuildPath(sch.Name, utils.ReservedParam),
			})
		}
		return bodies, nil
	}

}

func (t *TriggerService) TriggerManualMail(mode string, record utils.Record, fromSchema *sm.SchemaModel, triggerID, toSchemaID, destID int64) []utils.Record {
	mailings := []utils.Record{}
	var err error
	var toSchema sm.SchemaModel
	dest := []map[string]interface{}{}
	if toSchemaID < 0 || destID < 0 {
		toSchema = *fromSchema
		dest = []map[string]interface{}{record}
		toSchemaID = utils.ToInt64(fromSchema.ID)
		destID = utils.ToInt64(record[utils.SpecialIDParam])
	} else {
		toSchema, err = schema.GetSchemaByID(toSchemaID)
		if err == nil {
			if d, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(toSchema.Name, map[string]interface{}{
				utils.SpecialIDParam: destID,
			}, false); err == nil {
				dest = d
				if len(dest) > 0 {
					dest[0]["closing_by"] = t.Domain.GetUser()
					dest[0]["closing_comment"] = utils.GetString(record, "closing_comment")
					if len(utils.GetString(dest[0], "closing_comment")) > 0 {
						dest[0]["closing_comment"] = "\"" + utils.GetString(record, "closing_comment") + "\""
					}
				}
			}
		}
	}
	var toUsers []map[string]interface{}
	if len(dest) > 0 {
		if toUsers = t.handleOverrideEmailTo(record, dest[0], toSchema, triggerID); len(toUsers) == 0 {
			if mode == "auto" {
				return mailings
			}
		}
	} else if toUsers = t.handleOverrideEmailTo(record, map[string]interface{}{}, toSchema, triggerID); len(toUsers) == 0 {
		if mode == "auto" {
			return mailings
		}
	}
	mailSchema, err := schema.GetSchema(ds.DBEmailTemplate.Name)
	if err != nil {
		return mailings
	}
	rules := t.GetTriggerRules(triggerID, fromSchema, mailSchema.GetID(), record)
	for _, r := range rules {
		mailID := r["value"]
		if mailID == nil {
			continue
		}
		mails, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBEmailTemplate.Name, map[string]interface{}{
			utils.SpecialIDParam: mailID,
		}, false)
		if err != nil || len(mails) == 0 {
			continue
		}
		mail := mails[0]
		if tmplPath, ok := mail["redirect_on"]; ok { // with redirection only such as outlook
			if len(dest) > 0 {
				d := dest[0]
				path := utils.ToString(tmplPath)
				for k, v := range d {
					if strings.Contains(path, k) {
						path = strings.ReplaceAll(path, k, utils.ToString(v))
					}
				}
			}
			values, err := url.ParseQuery(utils.GetString(mail, "redirect_on"))
			if err == nil {
				SetRedirection(t.Domain.GetDomainID(), values.Encode())
			}
			return mailings
		}

		usfrom, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
			utils.SpecialIDParam: t.Domain.GetUserID(),
		}, false)
		if err != nil || len(usfrom) == 0 {
			continue
		}
		destOnResponse := int64(-1)
		if fromSchema.ID == utils.GetString(mail, ds.SchemaDBField+"_on_response") {
			destOnResponse = utils.GetInt(record, utils.SpecialIDParam)
		}
		signature := utils.GetString(mail, "signature")
		if len(toUsers) == 0 {
			if len(dest) > 0 {
				if m, err := connector.ForgeMail(
					usfrom[0],
					utils.Record{}, // always keep a copy
					utils.GetString(mail, "subject"),
					utils.GetString(mail, "template"),
					t.getLinkLabel(toSchema, dest[0]),
					t.Domain,
					utils.GetInt(mail, utils.SpecialIDParam),
					toSchemaID,
					destID,
					destOnResponse,
					t.getFileAttached(toSchema, record),
					signature,
				); err == nil {
					m["force_file_attached"] = mail["force_file_attached"]
					mailings = append(mailings, m)
				}
			} else {
				if m, err := connector.ForgeMail(
					usfrom[0],
					utils.Record{}, // always keep a copy
					utils.GetString(mail, "subject"),
					utils.GetString(mail, "template"),
					utils.Record{},
					t.Domain,
					utils.GetInt(mail, utils.SpecialIDParam),
					toSchemaID,
					destID,
					destOnResponse,
					"",
					signature,
				); err == nil {
					m["force_file_attached"] = mail["force_file_attached"]
					mailings = append(mailings, m)
				}
			}
		}
		for _, to := range toUsers {
			fmt.Println("TO USER", to)
			if len(dest) > 0 {
				if fmt.Sprintf("%v", toSchemaID) == utils.GetString(mail, ds.SchemaDBField+"_on_response") {
					destOnResponse = utils.GetInt(dest[0], utils.SpecialIDParam)
				}
				if m, err := connector.ForgeMail(
					usfrom[0],
					to, // always keep a copy
					utils.GetString(mail, "subject"),
					utils.GetString(mail, "template"),
					t.getLinkLabel(toSchema, dest[0]),
					t.Domain,
					utils.GetInt(mail, utils.SpecialIDParam),
					toSchemaID,
					destID,
					destOnResponse,
					t.getFileAttached(toSchema, record),
					signature,
				); err == nil {
					m["force_file_attached"] = mail["force_file_attached"]
					mailings = append(mailings, m)
				}
			} else {
				if m, err := connector.ForgeMail(
					usfrom[0],
					to, // always keep a copy
					utils.GetString(mail, "subject"),
					utils.GetString(mail, "template"),
					map[string]interface{}{},
					t.Domain,
					utils.GetInt(mail, utils.SpecialIDParam),
					-1,
					-1,
					destOnResponse,
					"",
					signature,
				); err == nil {
					m["force_file_attached"] = mail["force_file_attached"]
					mailings = append(mailings, m)
				}
			}
		}
	}
	return mailings
}

func (t *TriggerService) getFileAttached(toSchema sm.SchemaModel, record utils.Record) string {
	attached := ""
	for k, v := range record {
		if f, err := toSchema.GetField(k); err == nil && strings.Contains(strings.ToLower(f.Type), "upload") {
			if len(attached) == 0 {
				attached = utils.ToString(v)
			}
		}
	}
	return attached
}

func (t *TriggerService) getLinkLabel(toSchema sm.SchemaModel, record utils.Record) utils.Record {
	for _, field := range toSchema.Fields {
		if linkScheme, err := sm.GetSchemaByID(field.GetLink()); err == nil {
			key := utils.SpecialIDParam
			v := record[field.Name]
			if i, err := strconv.Atoi(utils.GetString(record, field.Name)); i == 0 || err == nil {
				key = "name"
				v = db.Quote(utils.ToString(v))
			}
			if v == "" || v == "''" {
				continue
			}
			fmt.Println(linkScheme.Name, key, v)
			// there is a link... soooo do something
			if res, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(linkScheme.Name, map[string]interface{}{
				key: v,
			}, false); err == nil && len(res) > 0 {
				item := res[0]
				if utils.GetString(item, "label") != "" {
					record[field.Name] = utils.GetString(item, "label")
				}
				if utils.GetString(item, "name") != "" {
					record[field.Name] = utils.GetString(item, "name")
				}
			}
		}
	}
	return record
}
