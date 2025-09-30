package user_service

import (
	"errors"
	"fmt"
	ds "sqldb-ws/domain/schema/database_resources"
	task "sqldb-ws/domain/specialized_service/task_service"
	servutils "sqldb-ws/domain/specialized_service/utils"
	"sqldb-ws/domain/utils"
	connector "sqldb-ws/infrastructure/connector/db"
	"time"
)

type DelegationService struct {
	servutils.AbstractSpecializedService
	SchemaID string
	DestID   string
}

func NewDelegationService() utils.SpecializedServiceITF {
	return &DelegationService{}
}

func (s *DelegationService) Entity() utils.SpecializedServiceInfo { return ds.DBDelegation }

func (s *DelegationService) VerifyDataIntegrity(record map[string]interface{}, tablename string) (map[string]interface{}, error, bool) {
	s.SchemaID = utils.GetString(record, ds.SchemaDBField)
	s.DestID = utils.GetString(record, ds.DestTableDBField)

	delete(record, ds.SchemaDBField)
	delete(record, ds.DestTableDBField)

	record[ds.UserDBField] = s.Domain.GetUserID() // affected create_by
	if utils.GetString(record, "delegated_"+ds.UserDBField) == s.Domain.GetUserID() {
		return map[string]interface{}{}, errors.New("can't add a delegation to yourself"), false
	}
	if _, err, ok := servutils.CheckAutoLoad(tablename, record, s.Domain); ok {
		return s.AbstractSpecializedService.VerifyDataIntegrity(record, tablename)
	} else {
		return record, err, false
	}
}

func (s *DelegationService) SpecializedCreateRow(record map[string]interface{}, tableName string) {
	// Define the layout for parsing
	layout := "2006-01-02" // Go's reference time format
	// Parse the date string into a time.Time
	endTime, _ := time.Parse(layout, utils.GetString(record, "end_date"))
	startTime, _ := time.Parse(layout, utils.GetString(record, "start_date"))
	fmt.Println(endTime, startTime, time.Now(), (endTime.After(time.Now()) || endTime.IsZero()), startTime.Before(time.Now()))
	now := time.Now()
	fmt.Println("THERE")
	if (endTime.After(now) || endTime.IsZero()) && (startTime.Before(now)) {
		s.Trigger(record)
	}
	s.AbstractSpecializedService.SpecializedCreateRow(record, tableName)
}

func (s *DelegationService) Trigger(rr map[string]interface{}) {
	fmt.Println("TRIGGER", rr)
	if utils.GetBool(rr, "all_tasks") {
		if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
			"is_close": false,
			utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
				ds.EntityDBField: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
					ds.UserDBField: s.Domain.GetUserID(),
				}, false, ds.EntityDBField),
				ds.UserDBField: s.Domain.GetUserID(),
			}, true, "id"),
		}, false); err == nil && len(res) > 0 {
			for _, r := range res {
				go func() {
					newTask := map[string]interface{}{}
					for k, v := range r {
						newTask[k] = v
					}
					newTask[ds.UserDBField] = rr["delegated_"+ds.UserDBField]
					newTask[ds.EntityDBField] = nil
					newTask["binded_"+ds.TaskDBField] = r[utils.SpecialIDParam]
					delete(newTask, utils.SpecialIDParam)
					share := map[string]interface{}{
						"shared_" + ds.UserDBField: rr["delegated_"+ds.UserDBField],
						ds.UserDBField:             rr[ds.UserDBField],
						"start_date":               connector.Quote(utils.GetString(rr, "start_date")),
						"end_date":                 connector.Quote(utils.GetString(rr, "end_date")),
						ds.SchemaDBField:           r[ds.SchemaDBField],
						ds.DelegationDBField:       rr[utils.SpecialIDParam],
						ds.DestTableDBField:        r[ds.DestTableDBField],
						"update_access":            false,
						"delete_access":            false,
					}
					if res, err := s.Domain.GetDb().SelectQueryWithRestriction(ds.DBShare.Name, share, false); err == nil && len(res) == 0 {
						share["start_date"] = rr["start_date"]
						share["end_date"] = rr["end_datey"]
						s.Domain.GetDb().ClearQueryFilter().CreateQuery(ds.DBShare.Name, share, func(s string) (string, bool) { return "", true })
					}
					if res, err := s.Domain.GetDb().SelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
						"binded_" + ds.TaskDBField: newTask["binded_"+ds.TaskDBField],
						ds.UserDBField:             newTask[ds.UserDBField],
					}, false); err == nil && len(res) == 0 {
						s.Domain.GetDb().ClearQueryFilter().CreateQuery(ds.DBTask.Name, newTask, func(s string) (string, bool) { return s, true })
					}
				}()
			}
		}
	} else if taskID := utils.GetInt(rr, ds.TaskDBField); taskID >= 0 {
		if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
			"is_close":           false,
			utils.SpecialIDParam: taskID,
		}, false); err == nil && len(res) > 0 {
			for _, r := range res {
				newTask := utils.Record{}
				for k, v := range r {
					newTask[k] = v
				}
				newTask[ds.UserDBField] = rr["delegated_"+ds.UserDBField]
				newTask[ds.EntityDBField] = nil
				newTask["binded_"+ds.TaskDBField] = r[utils.SpecialIDParam]
				delete(newTask, utils.SpecialIDParam)
				share := map[string]interface{}{
					"shared_" + ds.UserDBField: rr["delegated_"+ds.UserDBField],
					ds.UserDBField:             rr[ds.UserDBField],
					"start_date":               connector.Quote(utils.GetString(rr, "start_date")),
					"end_date":                 connector.Quote(utils.GetString(rr, "end_date")),
					ds.SchemaDBField:           r[ds.SchemaDBField],
					ds.DelegationDBField:       rr[utils.SpecialIDParam],
					ds.DestTableDBField:        r[ds.DestTableDBField],
					"update_access":            false,
					"delete_access":            false,
				}
				if res, err := s.Domain.GetDb().SelectQueryWithRestriction(ds.DBShare.Name, share, false); err == nil && len(res) == 0 {
					share["start_date"] = rr["start_date"]
					share["end_date"] = rr["end_datey"]
					s.Domain.GetDb().ClearQueryFilter().CreateQuery(ds.DBShare.Name, share, func(s string) (string, bool) { return "", true })
				}
				if res, err := s.Domain.GetDb().SelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
					"binded_" + ds.TaskDBField: newTask["binded_"+ds.TaskDBField],
					ds.UserDBField:             newTask[ds.UserDBField],
				}, false); err == nil && len(res) == 0 {
					s.Domain.GetDb().ClearQueryFilter().CreateQuery(ds.DBTask.Name, newTask, func(s string) (string, bool) { return s, true })
				}
			}
		}
	}
}

func (s *DelegationService) SpecializedDeleteRow(results []map[string]interface{}, tableName string) {
	for i, res := range results {
		share := map[string]interface{}{
			"binded_to_delegation": res[utils.SpecialIDParam],
		}
		s.Domain.GetDb().DeleteQueryWithRestriction(ds.DBShare.Name, share, false)
		res["state"] = "completed"
		s.Domain.GetDb().DeleteQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
			"binded_" + ds.TaskDBField: s.Domain.GetDb().BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
				ds.UserDBField: res[ds.UserDBField],
			}, false, utils.SpecialIDParam),
		}, false)
		results[i] = task.SetClosureStatus(res)
	}
}

func (s *DelegationService) SpecializedUpdateRow(results []map[string]interface{}, record map[string]interface{}) {
	for _, record := range results {
		// Define the layout for parsing
		layout := "2006-01-02" // Go's reference time format

		// Parse the date string into a time.Time
		endTime, _ := time.Parse(layout, utils.GetString(record, "end_date"))
		startTime, _ := time.Parse(layout, utils.GetString(record, "start_date"))
		now := time.Now()
		if (endTime.After(now) || endTime.IsZero()) && (startTime.Before(now)) {
			s.Trigger(record)
		}
	}
	s.AbstractSpecializedService.SpecializedUpdateRow(results, record)
}
