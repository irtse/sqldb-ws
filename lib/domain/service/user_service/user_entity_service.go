package user_service

import (
	"fmt"
	"time"
	tool "sqldb-ws/lib"
	"sqldb-ws/lib/infrastructure/entities"
)

type UserEntityService struct { tool.AbstractSpecializedService }

func (s *UserEntityService) Entity() tool.SpecializedServiceInfo { return entities.DBEntityUser }
func (s *UserEntityService) VerifyRowAutomation(record tool.Record, create bool) (tool.Record, bool) { return record, true }
func (s *UserEntityService) DeleteRowAutomation(results tool.Results) { }
func (s *UserEntityService) UpdateRowAutomation(results tool.Results, record tool.Record) {}
func (s *UserEntityService) WriteRowAutomation(record tool.Record) { }
func (s *UserEntityService) PostTreatment(results tool.Results) tool.Results {
	res := tool.Results{}
	for _, record := range results {
		found := true
		if date, ok := record["start_date"]; ok && date != nil && date != "" {
			today := time.Now() 
			start, err := time.Parse("2000-01-01", date.(string))
			if err == nil && start.Before(today) { found=false }
			if date2, ok := record["end_date"]; ok && date2 != nil && date2 != "" && found {
				end, err := time.Parse("2000-01-01", date2.(string))
				if err == nil && today.Before(end) { found=false}
			}
		}
		if !found {
			params := tool.Params{ tool.RootTableParam : entities.DBEntityUser.Name, 
				                   tool.RootRowsParam : tool.ReservedParam, }
			if entityId, ok2 := record[entities.RootID(entities.DBEntity.Name)]; ok2 {
				params[entities.RootID(entities.DBEntity.Name)]= fmt.Sprintf("%d", entityId.(int64))
			}
			if userId, ok3 := record[entities.RootID(entities.DBUser.Name)]; ok3 {
				params[entities.RootID(entities.DBUser.Name)]= fmt.Sprintf("%d", userId.(int64))
			}
			s.Domain.SafeCall(true, "", params, tool.Record{}, tool.DELETE, "Delete", )	
		} else { res = append(res, record) }
	}
	return res
}