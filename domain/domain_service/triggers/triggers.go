package triggers

import (
	"errors"
	"fmt"
	"slices"
	"sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	"sqldb-ws/domain/schema/models"
	sm "sqldb-ws/domain/schema/models"
	"sqldb-ws/domain/utils"
	conn "sqldb-ws/infrastructure/connector/db"
	"strings"
)

type TriggerService struct {
	Domain utils.DomainITF
}

func NewTrigger(domain utils.DomainITF) *TriggerService {
	return &TriggerService{
		Domain: domain,
	}
}

func (t *TriggerService) GetViewTriggers(record utils.Record, method utils.Method, fromSchema *sm.SchemaModel, toSchemaID, destID int64) []sm.ManualTriggerModel {
	if _, ok := t.Domain.GetParams().Get(utils.SpecialIDParam); method == utils.DELETE || (!ok && method == utils.SELECT) {
		return []sm.ManualTriggerModel{}
	}
	if utils.UPDATE == method && t.Domain.GetIsDraftToPublished() {
		method = utils.CREATE
		restr := []interface{}{
			ds.SchemaDBField + "=" + fromSchema.ID,
			ds.DestTableDBField + "=" + utils.GetString(record, utils.SpecialIDParam),
			"current_index > 1",
		}
		if res, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBRequest.Name, restr, false); err == nil && len(res) > 0 {
			return []sm.ManualTriggerModel{}
		}
	}
	mt := []sm.ManualTriggerModel{}
	if res, err := t.GetTriggers("manual", method, fromSchema.ID, utils.GetString(record, utils.SpecialIDParam)); err == nil {
		for _, r := range res {
			typ := utils.GetString(r, "type")
			switch typ {
			case "mail":
				if t, err := t.GetViewMailTriggers(record, fromSchema, utils.GetString(r, "description"), utils.GetString(r, "name"),
					utils.GetInt(r, utils.SpecialIDParam), toSchemaID, destID); err == nil {
					mt = append(mt, t...)
				}
			}
		}
	}
	return mt
}

func (t *TriggerService) GetTriggers(mode string, method utils.Method, fromSchemaID string, recordID string) ([]map[string]interface{}, error) {
	if method == utils.SELECT {
		method = utils.CREATE
		restr := []interface{}{
			ds.SchemaDBField + "=" + fromSchemaID,
			ds.DestTableDBField + "=" + recordID,
			"current_index > 1",
		}
		if recordID != "" {
			if res, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBRequest.Name, restr, false); err == nil && len(res) > 0 {
				return []map[string]interface{}{}, errors.New("can't select a trigger create on a upper after first task of request's workflow")
			}
		}

	}

	// TODO if it's the first task
	return t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBTrigger.Name, map[string]interface{}{
		"on_" + method.String(): true,
		"mode":                  conn.Quote(mode),
		ds.SchemaDBField:        fromSchemaID,
	}, false)
}

func (t *TriggerService) Trigger(fromSchema *sm.SchemaModel, record utils.Record, method utils.Method) {
	if t.Domain.GetAutoload() {
		return
	}
	if res, err := t.GetTriggers("auto", method, fromSchema.ID, utils.GetString(record, utils.SpecialIDParam)); err == nil {
		for _, r := range res {
			fmt.Println("TRIGGERS ID ", r[utils.SpecialIDParam])
			if !ShouldExecLater(r) {
				t.ExecTrigger(fromSchema, record, r)
				ShouldExecJob(r)
			}
		}
	}
}
func (t *TriggerService) ExecTrigger(fromSchema *sm.SchemaModel, record utils.Record, r map[string]interface{}) {
	typ := utils.GetString(r, "type")
	switch typ {
	case "mail":
		t.triggerMail(record, fromSchema,
			utils.GetInt(r, utils.SpecialIDParam),
			utils.GetInt(record, ds.SchemaDBField),
			utils.GetInt(record, ds.DestTableDBField))
	case "sms":
		break
	case "teams notification":
		break
	case "data":
		t.triggerData(record, fromSchema,
			utils.GetInt(r, utils.SpecialIDParam),
			utils.GetInt(record, ds.SchemaDBField),
			utils.GetInt(record, ds.DestTableDBField))
	}
}

func (t *TriggerService) ParseMails(toSplit string) []map[string]interface{} {
	splitted := ""
	if len(strings.Split(toSplit, ";")) > 0 {
		splitted = strings.ReplaceAll(strings.Join(strings.Split(toSplit, ";"), ","), " ", "")
	} else if len(strings.Split(toSplit, ",")) > 0 {
		splitted = strings.ReplaceAll(toSplit, " ", "")
	} else if len(strings.Split(toSplit, " ")) > 0 {
		splitted = strings.ReplaceAll(strings.Join(strings.Split(toSplit, ","), ","), " ", "")
	}
	if len(splitted) > 0 {
		s := []string{}
		for _, ss := range strings.Split(splitted, ",") {
			s = append(s, conn.Quote(ss))
		}
		if res, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
			"email": s,
		}, false); err == nil {
			return res
		}
	}
	return []map[string]interface{}{}
}

func (t *TriggerService) handleOverrideEmailTo(record, dest map[string]interface{}, destSchema models.SchemaModel, triggerID int64) []map[string]interface{} {
	userIDS := []string{}
	if res, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBTriggerDestination.Name, map[string]interface{}{
		ds.TriggerDBField: triggerID,
	}, false); err == nil {
		for _, userDest := range res {
			if utils.GetBool(userDest, "is_own") && userDest["from_"+ds.SchemaDBField] == nil {
				// wait no... should send to creator
				if res, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBDataAccess.Name, map[string]interface{}{
					ds.DestTableDBField: dest[utils.SpecialIDParam],
					ds.SchemaDBField:    destSchema.ID,
					"write":             true,
				}, false); err == nil {
					for _, r := range res {
						if !slices.Contains(userIDS, utils.GetString(r, ds.UserDBField)) {
							userIDS = append(userIDS, utils.GetString(r, ds.UserDBField))
						}
					}
				}
			} else if userDest["from_"+ds.SchemaDBField] != nil {
				sch, err := schema.GetSchemaByID(utils.GetInt(userDest, "from_"+ds.SchemaDBField))
				if err != nil {
					continue
				}
				key := "id"
				var v string
				if utils.GetBool(userDest, "is_own") {
					key = ds.UserDBField
					v = t.Domain.GetUserID()
				} else if userDest["from_"+ds.SchemaFieldDBField] == nil {
					continue
				} else {
					f, err := sch.GetFieldByID(utils.GetInt(userDest, "from_"+ds.SchemaFieldDBField))
					if err != nil {
						continue
					}
					key = f.Name
					if userDest["value"] != nil {
						v = fmt.Sprintf("%v", userDest["value"])
						if strings.Contains(f.Type, "char") {
							v = conn.Quote(v)
						}
					} else if f.GetLink() > 0 {
						if f.GetLink() == destSchema.GetID() {
							v = utils.GetString(dest, utils.SpecialIDParam)
						} else {
							for _, ff := range destSchema.Fields {
								if f.GetLink() == ff.GetLink() {
									v = utils.GetString(dest, ff.Name)
									break
								}
							}
						}
					}
				}
				if v == "" {
					continue
				}
				if usr, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(sch.Name, map[string]interface{}{
					key: v,
				}, false); err == nil {
					for _, u := range usr {
						if userDest["from_compare_"+ds.SchemaFieldDBField] != nil {
							ff, err := sch.GetFieldByID(utils.GetInt(userDest, "from_compare_"+ds.SchemaFieldDBField))
							if err != nil {
								continue
							}
							if !slices.Contains(userIDS, utils.GetString(u, ff.Name)) {
								userIDS = append(userIDS, utils.GetString(u, ff.Name))
							}
						} else {
							if !slices.Contains(userIDS, utils.GetString(u, ds.UserDBField)) {
								userIDS = append(userIDS, utils.GetString(u, ds.UserDBField))
							}
						}
					}
				}
			}
		}
	}
	if len(userIDS) > 0 {
		if usto, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
			utils.SpecialIDParam: userIDS,
		}, false); err == nil {
			return usto
		}
	}
	return []map[string]interface{}{}
}

func (t *TriggerService) triggerMail(record utils.Record, fromSchema *sm.SchemaModel, triggerID, toSchemaID, destID int64) {
	for _, mail := range t.TriggerManualMail("auto", record, fromSchema, triggerID, toSchemaID, destID) {
		delete(mail, "force_file_attached")
		t.Domain.CreateSuperCall(utils.AllParams(ds.DBEmailSended.Name).RootRaw(), mail)
	}
}

func (t *TriggerService) triggerData(record utils.Record, fromSchema *sm.SchemaModel, triggerID, toSchemaID, destID int64) {
	if toSchemaID < 0 || destID < 0 {
		toSchemaID = utils.ToInt64(fromSchema.ID)
		destID = utils.GetInt(record, utils.SpecialIDParam)
	}
	// PROBLEM WE CAN'T DECOLERATE and action on not a sub data of it. (not a problem for now)

	rules := t.GetTriggerRules(triggerID, fromSchema, toSchemaID, record)
	for _, r := range rules {
		if toSchemaID != utils.GetInt(r, "to_"+ds.SchemaDBField) {
			continue
		}

		toSchema, err := schema.GetSchemaByID(toSchemaID)
		if err != nil {
			fmt.Println("ERR", err)
			continue
		}

		field, err := toSchema.GetFieldByID(utils.GetInt(r, "to_"+ds.SchemaFieldDBField))
		if err != nil {
			fmt.Println("ERR2", err)
			continue
		}

		value := utils.GetString(r, "value")
		if value == "" {
			value = utils.GetString(record, field.Name)
		}
		t.Domain.GetDb().ClearQueryFilter().UpdateQuery(toSchema.Name, map[string]interface{}{
			field.Name: value,
		}, map[string]interface{}{
			utils.SpecialIDParam: destID,
		}, false)
		s := t.Domain.GetSpecialized(toSchema.Name)
		s.SpecializedUpdateRow([]map[string]interface{}{
			map[string]interface{}{
				field.Name:           value,
				utils.SpecialIDParam: destID,
			},
		},
			map[string]interface{}{
				field.Name: value,
			},
		)
	}
}

func (t *TriggerService) GetTriggerRules(triggerID int64, fromSchema *sm.SchemaModel, toSchemaID int64, record utils.Record) []map[string]interface{} {
	if res, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBTriggerCondition.Name, map[string]interface{}{
		ds.TriggerDBField: triggerID,
	}, false); err == nil && len(res) > 0 {
		for _, cond := range res {
			if cond[ds.SchemaFieldDBField] == nil && utils.GetString(record, utils.SpecialIDParam) != utils.GetString(cond, "value") {
				fmt.Println("!Value", utils.GetString(record, utils.SpecialIDParam), utils.GetString(cond, "value"))
				return []map[string]interface{}{}
			}
			f, err := fromSchema.GetFieldByID(utils.GetInt(cond, ds.SchemaFieldDBField))
			if err != nil || (record[f.Name] == nil && utils.GetBool(cond, "not_null")) || utils.GetString(record, f.Name) != utils.GetString(cond, "value") {
				fmt.Println("!Value null", utils.GetString(record, utils.SpecialIDParam), utils.GetString(cond, "value"))
				return []map[string]interface{}{}
			}
		}
	}
	rules, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBTriggerRule.Name, map[string]interface{}{
		ds.TriggerDBField:        triggerID,
		"to_" + ds.SchemaDBField: toSchemaID,
	}, false)
	if err != nil {
		return []map[string]interface{}{}
	}
	return rules
}
