package user_service

import (
	"sqldb-ws/domain/domain_service/filter"
	ds "sqldb-ws/domain/schema/database_resources"
	servutils "sqldb-ws/domain/specialized_service/utils"
	"sqldb-ws/domain/utils"
	connector "sqldb-ws/infrastructure/connector/db"
	"strings"
	"time"
)

type UserService struct {
	servutils.AbstractSpecializedService
}

func NewUserService() utils.SpecializedServiceITF {
	return &UserService{}
}

func (s *UserService) VerifyDataIntegrity(record map[string]interface{}, tablename string) (map[string]interface{}, error, bool) {
	record["name"] = strings.ToLower(utils.GetString(record, "name"))
	record["email"] = strings.ToLower(utils.GetString(record, "email"))
	return record, nil, true
}
func (s *UserService) SpecializedCreateRow(record map[string]interface{}, tableName string) {
	s.AbstractSpecializedService.SpecializedCreateRow(record, tableName)
}
func (s *UserService) Entity() utils.SpecializedServiceInfo { return ds.DBUser }

func (s *UserService) GenerateQueryFilter(tableName string, innerestr ...string) (string, string, string, string) {
	if scope, ok := s.Domain.GetParams().Get(utils.RootScope); ok && strings.Contains(scope, "enable_share") && s.Domain.GetUserID() != "" {
		splitted := strings.Split(strings.ReplaceAll(scope, "enable_share_", ""), "_")
		if len(splitted) > 1 {
			arr := []string{
				connector.FormatSQLRestrictionWhereByMap("", map[string]interface{}{
					ds.DestTableDBField: utils.ToInt64(splitted[1]),
					ds.SchemaDBField:    utils.ToInt64(splitted[0]),
					ds.UserDBField:      s.Domain.GetUserID(),
				}, false),
			}
			currentTime := time.Now()
			arr = append(arr, "('"+currentTime.Format("2000-01-01")+"' >= start_date AND '"+currentTime.Format("2000-01-01")+"' < end_date)")
			innerestr = append(innerestr, connector.FormatSQLRestrictionWhereByMap("", map[string]interface{}{
				"!" + utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBShare.Name, arr, false, "shared_"+ds.UserDBField),
			}, true))
		}
	} else if scope, ok := s.Domain.GetParams().Get(utils.RootScope); ok && strings.Contains(scope, "disable_share") && s.Domain.GetUserID() != "" {
		splitted := strings.Split(strings.ReplaceAll(scope, "disable_share", ""), "_")
		if len(splitted) > 1 {
			arr := []string{
				connector.FormatSQLRestrictionWhereByMap("", map[string]interface{}{
					ds.UserDBField:      s.Domain.GetUserID(),
					ds.DestTableDBField: utils.ToInt64(splitted[1]),
					ds.SchemaDBField:    utils.ToInt64(splitted[0]),
				}, false),
			}
			currentTime := time.Now()
			arr = append(arr, "('"+currentTime.Format("2000-01-01")+"' >= start_date AND '"+currentTime.Format("2000-01-01")+"' < end_date)")
			innerestr = append(innerestr, connector.FormatSQLRestrictionWhereByMap("", map[string]interface{}{
				utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBShare.Name, arr, false, "shared_"+ds.UserDBField),
			}, true))
		}
	}
	if scope, ok := s.Domain.GetParams().Get(utils.RootScope); ok && scope == "enable_delegate" && s.Domain.GetUserID() != "" {
		innerestr = append(innerestr, connector.FormatSQLRestrictionWhereByMap("", map[string]interface{}{
			"!" + utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBDelegation.Name, map[string]interface{}{
				ds.UserDBField: s.Domain.GetUserID(),
			}, true, "delegated_"+ds.UserDBField),
		}, true))
	} else if scope, ok := s.Domain.GetParams().Get(utils.RootScope); ok && scope == "disable_delegate" && s.Domain.GetUserID() != "" {
		innerestr = append(innerestr, connector.FormatSQLRestrictionWhereByMap("", map[string]interface{}{
			utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBDelegation.Name, map[string]interface{}{
				ds.UserDBField: s.Domain.GetUserID(),
			}, true, "delegated_"+ds.UserDBField),
		}, true))
	}
	return filter.NewFilterService(s.Domain).GetQueryFilter(tableName, s.Domain.GetParams().Copy(), innerestr...)
}
