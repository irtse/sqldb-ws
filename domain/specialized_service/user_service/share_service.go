package user_service

import (
	"errors"
	"sqldb-ws/domain/domain_service/filter"
	"sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	sm "sqldb-ws/domain/schema/models"
	servutils "sqldb-ws/domain/specialized_service/utils"
	"sqldb-ws/domain/utils"

	connector "sqldb-ws/infrastructure/connector/db"
)

type ShareService struct {
	servutils.AbstractSpecializedService
}

func NewShareService() utils.SpecializedServiceITF {
	return &ShareService{}
}

func (s *ShareService) SpecializedCreateRow(record map[string]interface{}, tableName string) {
	if sch, err := schema.GetSchemaByID(utils.GetInt(record, ds.SchemaDBField)); err == nil && sch.HasField(ds.DestTableDBField) {
		if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(sch.Name, map[string]interface{}{
			utils.SpecialIDParam: record[ds.DestTableDBField],
		}, false); err == nil {
			for _, r := range res {
				share := map[string]interface{}{
					"shared_" + ds.UserDBField: record["shared_"+ds.UserDBField],
					ds.UserDBField:             record[ds.UserDBField],
					"start_date":               record["start_date"],
					"end_date":                 record["end_date"],
					ds.SchemaDBField:           r[ds.SchemaDBField],
					ds.DestTableDBField:        r[ds.DestTableDBField],
					"update_access":            record["update_access"],
					"delete_access":            record["delete_access"],
				}
				s.Domain.CreateSuperCall(utils.AllParams(ds.DBShare.Name).RootRaw(), share)
			}
		}
	}
	s.AbstractSpecializedService.SpecializedCreateRow(record, tableName)
}
func (s *ShareService) Entity() utils.SpecializedServiceInfo { return ds.DBShare }

func (s *ShareService) VerifyDataIntegrity(record map[string]interface{}, tablename string) (map[string]interface{}, error, bool) {
	record[ds.UserDBField] = s.Domain.GetUserID() // affected create_by
	if utils.GetString(record, "shared_"+ds.UserDBField) == s.Domain.GetUserID() {
		return map[string]interface{}{}, errors.New("can't add a shared to yourself"), false
	}
	sch, err := schema.GetSchema(tablename)
	if err != nil {
		return record, errors.New("not schema found"), false
	}
	if !s.Domain.VerifyAuth(sch.Name, "", sm.LEVELNORMAL, utils.UPDATE) {
		record["update_access"] = false
	}
	if !s.Domain.VerifyAuth(sch.Name, "", sm.LEVELNORMAL, utils.CREATE) {
		record["create_access"] = false
	}
	if !s.Domain.VerifyAuth(sch.Name, "", sm.LEVELNORMAL, utils.DELETE) {
		record["delete_access"] = false
	}
	if _, err, ok := servutils.CheckAutoLoad(tablename, record, s.Domain); ok {
		return s.AbstractSpecializedService.VerifyDataIntegrity(record, tablename)
	} else {
		return record, err, false
	}
}

func (s *ShareService) GenerateQueryFilter(tableName string, innerestr ...string) (string, string, string, string) {
	if s.Domain.IsSuperCall() {
		innerestr = append(innerestr, connector.FormatSQLRestrictionWhereByMap("", map[string]interface{}{
			ds.UserDBField: s.Domain.GetUserID(),
		}, true))
	}
	return filter.NewFilterService(s.Domain).GetQueryFilter(tableName, s.Domain.GetParams().Copy(), innerestr...)
}
