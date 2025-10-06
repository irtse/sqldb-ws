package favorite_service

import (
	ds "sqldb-ws/domain/schema/database_resources"
	servutils "sqldb-ws/domain/specialized_service/utils"
	"sqldb-ws/domain/utils"
)

// DONE - ~ 200 LINES - PARTIALLY TESTED
type HistoryService struct {
	servutils.AbstractSpecializedService
}

func NewHistoryService() utils.SpecializedServiceITF {
	return &HistoryService{}
}

func (s *HistoryService) Entity() utils.SpecializedServiceInfo { return ds.DBDataAccess }

func (s *HistoryService) TransformToGenericView(results utils.Results, tableName string, dest_id ...string) (res utils.Results) {
	for _, d := range results {
		if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
			utils.SpecialIDParam: d[ds.UserDBField],
		}, false); err == nil && len(res) > 0 {
			d["user"] = utils.GetString(res[0], "name")
		}
	}
	return results
}
