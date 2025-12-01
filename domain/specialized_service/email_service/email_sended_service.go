package email_service

import (
	ds "sqldb-ws/domain/schema/database_resources"
	servutils "sqldb-ws/domain/specialized_service/utils"
	"sqldb-ws/domain/utils"
	connector "sqldb-ws/infrastructure/connector/db"
	"strings"

	"github.com/google/uuid"
)

// DONE - ~ 200 LINES - PARTIALLY TESTED
type EmailSendedService struct {
	servutils.AbstractSpecializedService
	To []string
}

func NewEmailSendedService() utils.SpecializedServiceITF {
	return &EmailSendedService{}
}

func (s *EmailSendedService) Entity() utils.SpecializedServiceInfo { return ds.DBEmailSended }

func (s *EmailSendedService) SpecializedCreateRow(record map[string]interface{}, tableName string) {
	for _, to := range s.To {
		if strings.Contains(to, "@") {
			if res, err := s.Domain.CreateSuperCall(utils.AllParams(ds.DBEmailSendedUser.Name).RootRaw(), map[string]interface{}{
				"name":                to,
				ds.EmailSendedDBField: record[utils.SpecialIDParam],
			}); err == nil && len(res) > 0 {
				record["to_email"] = res[0][utils.SpecialIDParam]
				s.Domain.GetDb().ClearQueryFilter().UpdateQuery(ds.DBEmailSended.Name, record, map[string]interface{}{
					utils.SpecialIDParam: record[utils.SpecialIDParam],
				}, false)
			}
		} else if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
			"name": connector.Quote(to),
		}, false); err == nil && len(res) > 0 {
			for _, r := range res {
				if res, err := s.Domain.CreateSuperCall(utils.AllParams(ds.DBEmailSendedUser.Name).RootRaw(), map[string]interface{}{
					"name":                to,
					ds.UserDBField:        r[utils.SpecialIDParam],
					ds.EmailSendedDBField: record[utils.SpecialIDParam],
				}); err == nil && len(res) > 0 {
					record["to_email"] = res[0][utils.SpecialIDParam]
					s.Domain.GetDb().ClearQueryFilter().UpdateQuery(ds.DBEmailSended.Name, record, map[string]interface{}{
						utils.SpecialIDParam: record[utils.SpecialIDParam],
					}, false)
				}
			}
		}
	}
	if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBEmailTemplate.Name, map[string]interface{}{
		utils.SpecialIDParam:  record[ds.EmailTemplateDBField],
		"is_response_valid":   false,
		"is_response_refused": false,
	}, false); err == nil && len(res) > 0 {
		if utils.GetBool(res[0], "generate_task") {
			i := int64(-1)
			if t, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
				"is_meta":           false,
				"is_close":          false,
				ds.DestTableDBField: record["mapped_with"+ds.DestTableDBField],
				ds.SchemaDBField:    record["mapped_with"+ds.SchemaDBField],
			}, false); err == nil && len(t) > 0 {
				if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
					"name":              connector.Quote("waiting mails responses"),
					"current_index":     utils.GetFloat(t[0], "current_index"),
					"is_meta":           true,
					"is_close":          false,
					ds.DestTableDBField: record["mapped_with"+ds.DestTableDBField],
					ds.SchemaDBField:    record["mapped_with"+ds.SchemaDBField],
				}, false); err == nil && len(res) > 0 {
					i = utils.GetInt(res[0], utils.SpecialIDParam)
				} else {
					if id, err := s.Domain.GetDb().ClearQueryFilter().CreateQuery(ds.DBRequest.Name, map[string]interface{}{
						"name":              "waiting mails responses",
						"current_index":     1,
						"is_meta":           true,
						ds.DestTableDBField: record["mapped_with"+ds.DestTableDBField],
						ds.SchemaDBField:    record["mapped_with"+ds.SchemaDBField],
					}, func(s string) (string, bool) { return "", true }); err == nil {
						i = id
					} else {
						return
					}
				}
				if i >= 0 {
					for _, r := range t {
						if tt, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
							ds.RequestDBField:           r[utils.SpecialIDParam],
							"meta_" + ds.RequestDBField: i,
							"name":                      connector.Quote("waiting mails responses"),
						}, false); err != nil || len(tt) == 0 {
							s.Domain.GetDb().CreateQuery(ds.DBTask.Name, map[string]interface{}{
								ds.DestTableDBField:         r[ds.DestTableDBField],
								"name":                      "waiting mails responses",
								ds.SchemaDBField:            r[ds.SchemaDBField],
								ds.RequestDBField:           r[utils.SpecialIDParam],
								"meta_" + ds.RequestDBField: i,
							}, func(v string) (string, bool) { return "", true })
						}
					}
				}
				s.Domain.GetDb().CreateQuery(ds.DBTask.Name, map[string]interface{}{
					ds.DestTableDBField: record["mapped_with"+ds.DestTableDBField],
					ds.SchemaDBField:    record["mapped_with"+ds.SchemaDBField],
					ds.RequestDBField:   i,
					"name":              utils.GetString(record, "code"),
				}, func(s string) (string, bool) { return "", true })
			}
		}
	}

	s.AbstractSpecializedService.SpecializedCreateRow(record, tableName)
}

func (s *EmailSendedService) VerifyDataIntegrity(record map[string]interface{}, tablename string) (map[string]interface{}, error, bool) {
	if record["to_email"] != nil {
		for _, e := range utils.ToList(record["to_email"]) {
			s.To = append(s.To, utils.ToString(utils.ToMap(e)["name"]))
		}
	}
	delete(record, "to_email")
	tos := []string{}
	for i, to := range s.To {
		record["code"] = uuid.New()
		if i < len(s.To)-1 {
			s.Domain.CreateSuperCall(utils.AllParams(ds.DBEmailSended.Name), record)
		} else {
			tos = append(tos, to)
		}
	}
	s.To = tos
	return s.AbstractSpecializedService.VerifyDataIntegrity(record, tablename)
}
