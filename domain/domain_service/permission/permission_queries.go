package permission

import (
	"sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	sm "sqldb-ws/domain/schema/models"
	"sqldb-ws/domain/utils"
	connector "sqldb-ws/infrastructure/connector/db"
	"time"
)

func (p *PermDomainService) IsShared(schema sm.SchemaModel, destID string, key string, val bool) bool {
	if destID == "" {
		return false
	}
	arr := []interface{}{
		connector.FormatSQLRestrictionWhereByMap("", map[string]interface{}{
			ds.UserDBField: p.db.BuildSelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
				"name":  connector.Quote(p.User),
				"email": connector.Quote(p.User),
			}, true, "id"),
			ds.SchemaDBField:    schema.ID,
			ds.DestTableDBField: destID,
			key:                 val,
		}, false),
	}
	currentTime := time.Now()
	arr = append(arr, "('"+currentTime.Format("2006-01-02 15:04:05")+"' >= start_date AND ('"+currentTime.Format("2006-01-02 15:04:05")+"' < end_date OR end_date IS NULL))")
	res, err := p.db.SelectQueryWithRestriction(ds.DBShare.Name, arr, false)
	return err == nil && len(res) > 0
}

func (p *PermDomainService) checkUpdateCreatePermissions(tableName, destID string, domain utils.DomainITF) bool {
	if p.Empty || destID == "" {
		return true
	}
	sch, e := schema.GetSchema(tableName)
	if e != nil {
		return false
	}
	test := p.db.BuildSelectQueryWithRestriction(
		ds.DBEntityUser.Name,
		map[string]interface{}{
			ds.UserDBField: domain.GetUserID(),
		}, true, ds.EntityDBField,
	)
	if res, err := p.db.ClearQueryFilter().SimpleMathQuery("COUNT", ds.DBDataAccess.Name, map[string]interface{}{
		ds.SchemaDBField:           sch.ID,
		utils.RootDestTableIDParam: destID,
		ds.UserDBField:             domain.GetUserID(),
		"write":                    true,
	}, true); err == nil && len(res) > 0 && res[0]["result"] != nil && utils.ToInt64(res[0]["result"]) > 0 {
		if res, err := p.db.ClearQueryFilter().SimpleMathQuery("COUNT", ds.DBRequest.Name, map[string]interface{}{
			ds.SchemaDBField:           sch.ID,
			utils.RootDestTableIDParam: destID,
			"is_close":                 false,
		}, true); err == nil && len(res) > 0 && res[0]["result"] != nil && utils.ToInt64(res[0]["result"]) > 0 {
			return true
		}
	}

	res, err := p.db.SimpleMathQuery("COUNT", ds.DBTask.Name, map[string]interface{}{
		utils.SpecialIDParam: p.db.BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
			ds.UserDBField:   domain.GetUserID(),
			ds.EntityDBField: test,
		}, true, utils.SpecialIDParam),
		ds.SchemaDBField:           sch.ID,
		utils.RootDestTableIDParam: destID,
		"is_close":                 false,
	}, false)
	return err == nil && len(res) > 0 && res[0]["result"] != nil && utils.ToInt64(res[0]["result"]) > 0
}
