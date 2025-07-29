package email_service

import (
	"errors"
	"fmt"
	ds "sqldb-ws/domain/schema/database_resources"
	servutils "sqldb-ws/domain/specialized_service/utils"
	"sqldb-ws/domain/utils"
	conn "sqldb-ws/infrastructure/connector"
	db "sqldb-ws/infrastructure/connector/db"
)

// DONE - ~ 200 LINES - PARTIALLY TESTED
type EmailSendedUserService struct {
	servutils.AbstractSpecializedService
}

func NewEmailSendedUserService() utils.SpecializedServiceITF {
	return &EmailSendedUserService{}
}

func (s *EmailSendedUserService) Entity() utils.SpecializedServiceInfo { return ds.DBEmailSendedUser }

func (s *EmailSendedUserService) SpecializedCreateRow(record map[string]interface{}, tableName string) {
	isValid := false
	emailTo := ""
	if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBEmailTemplate.Name, map[string]interface{}{
		"is_response_valid":   false,
		"is_response_refused": false,
		utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(
			ds.DBEmailSended.Name, map[string]interface{}{
				utils.SpecialIDParam: record[ds.EmailSendedDBField],
			}, false, ds.EmailTemplateDBField,
		),
	}, false); err == nil && len(res) > 0 {
		if utils.GetBool(res[0], "waiting_response") {
			// should enrich with a binary response yes or no.
			isValid = true
		}
	}
	if record[ds.UserDBField] != nil {
		if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
			utils.SpecialIDParam: record[ds.UserDBField],
		}, false); err == nil && len(res) > 0 {
			emailTo = utils.GetString(res[0], "email")
		}
	} else if record["name"] != nil {
		emailTo = utils.GetString(record, "name")
	}
	if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
		utils.SpecialIDParam: s.Domain.GetUserID(),
	}, false); err == nil && len(res) > 0 && emailTo != "" {
		if rr, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBEmailSended.Name, map[string]interface{}{
			utils.SpecialIDParam: record[ds.EmailSendedDBField],
		}, false); err == nil && len(rr) > 0 {
			go conn.SendMail(utils.GetString(res[0], "email"), emailTo, rr[0], isValid)
		}
	} else {
		fmt.Println("can't email because of a missing <send to> user")
	}
}

func (s *EmailSendedUserService) VerifyDataIntegrity(record map[string]interface{}, tablename string) (map[string]interface{}, error, bool) {
	fmt.Println("SEND TO 1", record)
	if utils.GetString(record, "name") == "" && utils.GetString(record, ds.UserDBField) == "" {
		return record, errors.New("no email to send to"), false
	}
	if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBEmailSendedUser.Name, map[string]interface{}{
		ds.EmailSendedDBField: record[ds.EmailSendedDBField],
		utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBEmailSendedUser.Name, map[string]interface{}{
			ds.UserDBField: record[ds.UserDBField],
			"name":         db.Quote(utils.GetString(record, "name")),
		}, true, utils.SpecialIDParam),
	}, false); err == nil && len(res) > 0 {
		return record, errors.New("already send"), false
	}
	fmt.Println("SEND TO 2", record)
	return s.AbstractSpecializedService.VerifyDataIntegrity(record, tablename)
}
