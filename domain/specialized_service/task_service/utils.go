package task_service

import (
	schserv "sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	sm "sqldb-ws/domain/schema/models"
	"sqldb-ws/domain/utils"
	connector "sqldb-ws/infrastructure/connector/db"
	"strings"
	"time"
)

var SchemaDBField = ds.RootID(ds.DBSchema.Name)
var RequestDBField = ds.RootID(ds.DBRequest.Name)
var WorkflowSchemaDBField = ds.RootID(ds.DBWorkflowSchema.Name)
var UserDBField = ds.RootID(ds.DBUser.Name)
var EntityDBField = ds.RootID(ds.DBEntity.Name)
var DestTableDBField = ds.RootID("dest_table")
var FilterDBField = ds.RootID(ds.DBFilter.Name)

func ConstructNotificationTask(scheme utils.Record, request utils.Record, domain utils.DomainITF) map[string]interface{} {
	task := map[string]interface{}{
		sm.NAMEKEY:               scheme.GetString(sm.NAMEKEY),
		"description":            scheme.GetString(sm.NAMEKEY),
		"urgency":                scheme["urgency"],
		"priority":               scheme["priority"],
		ds.WorkflowSchemaDBField: scheme[utils.SpecialIDParam],
		ds.UserDBField:           scheme[ds.UserDBField],
		ds.EntityDBField:         scheme[ds.EntityDBField],
		ds.SchemaDBField:         scheme[ds.SchemaDBField],
		ds.DestTableDBField:      scheme[ds.DestTableDBField],
		ds.RequestDBField:        request[utils.SpecialIDParam],
		"opening_date":           time.Now().Format(time.RFC3339),

		"override_state_completed": scheme["override_state_completed"],
		"override_state_dismiss":   scheme["override_state_dismiss"],
		"override_state_refused":   scheme["override_state_refused"],
	}
	if utils.GetBool(scheme, "assign_to_creator") {
		task[ds.UserDBField] = domain.GetUserID()
	}
	return task
}

func CheckStateIsEnded(state interface{}) bool {
	return state == "completed" || state == "dismiss" || state == "refused" || state == "canceled"
}

func SetClosureStatus(res map[string]interface{}) map[string]interface{} {
	if state, ok := res["state"]; ok && CheckStateIsEnded(utils.ToString(state)) {
		res["is_close"] = true
		res["closing_date"] = time.Now().Format(time.RFC3339)
	} else {
		res["state"] = "progressing"
		res["is_close"] = false
		res["closing_date"] = nil
	}
	return res
}

func CreateNewDataFromTask(schema sm.SchemaModel, newTask utils.Record, record utils.Record, domain utils.DomainITF) utils.Record {
	r := utils.Record{"is_draft": true}
	if schema.HasField("name") {
		if schema, err := schserv.GetSchemaByID(utils.GetInt(record, ds.SchemaDBField)); err == nil {
			if res, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(schema.Name, map[string]interface{}{
				utils.SpecialIDParam: record[ds.DestTableDBField],
			}, false); err == nil && len(res) > 0 {
				r[sm.NAMEKEY] = utils.GetString(res[0], "name")
			}
		} else {
			r["name"] = utils.GetString(newTask, "name")
		}
	}
	if schema.HasField(ds.DestTableDBField) && schema.HasField(ds.SchemaDBField) {
		// get workflow source schema + dest ID
		r[ds.DestTableDBField] = record[ds.DestTableDBField]
		r[ds.SchemaDBField] = record[ds.SchemaDBField]
	}
	if schema.HasField(ds.UserDBField) {
		r[ds.UserDBField] = record[ds.UserDBField]
	}
	if schema.HasField(ds.EntityDBField) {
		r[ds.EntityDBField] = record[ds.EntityDBField]
	}
	for _, f := range schema.Fields {
		if f.GetLink() == record[ds.SchemaDBField] {
			r[f.Name] = record[ds.DestTableDBField]
		}
	}

	if i, err := domain.GetDb().ClearQueryFilter().CreateQuery(schema.Name, r, func(s string) (string, bool) { return "", true }); err == nil {
		r[utils.SpecialIDParam] = i

		newTask[ds.DestTableDBField] = i
		domain.GetDb().CreateQuery(ds.DBDataAccess.Name, map[string]interface{}{
			ds.SchemaDBField:    schema.ID,
			ds.DestTableDBField: i,
			ds.UserDBField:      domain.GetUserID(),
			"write":             true,
			"update":            false,
		}, func(s string) (string, bool) {
			return "", true
		})
	}
	return newTask
}

func PrepareAndCreateTask(scheme utils.Record, request map[string]interface{}, record map[string]interface{}, domain utils.DomainITF, fromTask bool) map[string]interface{} {
	newTask := ConstructNotificationTask(scheme, request, domain)
	delete(newTask, utils.SpecialIDParam)
	if utils.GetString(newTask, ds.SchemaDBField) == utils.GetString(request, ds.SchemaDBField) {
		newTask[ds.SchemaDBField] = request[ds.SchemaDBField]
		newTask[ds.DestTableDBField] = request[ds.DestTableDBField]
	} else if schema, err := schserv.GetSchemaByID(utils.GetInt(newTask, ds.SchemaDBField)); err == nil {
		newTask = CreateNewDataFromTask(schema, newTask, record, domain)
	}
	isMeta := strings.Contains(utils.GetString(record, "nexts"), utils.GetString(scheme, "wrapped_"+ds.WorkflowDBField)) && utils.GetString(scheme, "wrapped_"+ds.WorkflowDBField) != "" || !fromTask
	if id, ok := scheme["wrapped_"+ds.WorkflowDBField]; ok && id != nil && isMeta {
		createMetaRequest(newTask, id, domain)
	}
	shouldCreate := utils.GetString(record, "nexts") == utils.ReservedParam || utils.GetString(record, "nexts") == "" || isMeta
	if shouldCreate {
		createTaskAndNotify(newTask, request, domain, fromTask)
	}
	return newTask
}

func createTaskAndNotify(task map[string]interface{}, request map[string]interface{}, domain utils.DomainITF, isTask bool) {
	i, err := domain.GetDb().CreateQuery(ds.DBTask.Name, task, func(s string) (string, bool) {
		return "", true
	})
	if err != nil {
		return
	}
	CreateDelegated(task, request, i, domain)
	notify(task, i, domain)
}

func notify(task utils.Record, i int64, domain utils.DomainITF) {
	if schema, err := schserv.GetSchema(ds.DBTask.Name); err == nil {
		name := utils.GetString(task, "name")
		if res, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(schema.Name, map[string]interface{}{
			utils.SpecialIDParam: i,
		}, false); err == nil && len(res) > 0 {
			name += " <" + utils.GetString(res[0], "name") + ">"
		}
		notif := utils.Record{
			"name":              utils.GetString(task, "name"),
			"description":       utils.GetString(task, "name"),
			ds.UserDBField:      task[ds.UserDBField],
			ds.EntityDBField:    task[ds.EntityDBField],
			ds.DestTableDBField: i,
		}
		notif["link_id"] = schema.ID
		if schema, err := schserv.GetSchemaByID(utils.GetInt(task, ds.SchemaDBField)); err == nil {
			if res, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(schema.Name, map[string]interface{}{
				utils.SpecialIDParam: task[ds.DestTableDBField],
			}, false); err == nil && len(res) > 0 {
				notif[sm.NAMEKEY] = utils.GetString(res[0], "name")
			}
		}
		domain.GetDb().ClearQueryFilter().CreateQuery(ds.DBNotification.Name, notif, func(s string) (string, bool) {
			return "", true
		})
	}
}

func createMetaRequest(task map[string]interface{}, id interface{}, domain utils.DomainITF) {
	domain.CreateSuperCall(utils.AllParams(ds.DBRequest.Name).RootRaw(), utils.Record{
		ds.WorkflowDBField:  id,
		sm.NAMEKEY:          "Meta request for " + utils.GetString(task, sm.NAMEKEY) + " task.",
		"current_index":     1,
		"is_meta":           true,
		ds.SchemaDBField:    task[ds.SchemaDBField],
		ds.DestTableDBField: task[ds.DestTableDBField],
		ds.UserDBField:      utils.GetInt(task, ds.UserDBField),
	})
}

func CreateDelegated(record utils.Record, request utils.Record, id int64, domain utils.DomainITF) {
	currentTime := time.Now()
	sqlFilter := []string{
		"('" + currentTime.Format("2006-01-02") + "' >= start_date AND '" + currentTime.Format("2006-01-02") + "' < end_date)",
	}
	sqlFilter = append(sqlFilter, connector.FormatSQLRestrictionWhereByMap("", map[string]interface{}{
		"all_tasks": true,
		utils.SpecialIDParam: domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBDelegation.Name, map[string]interface{}{
			ds.UserDBField:                      record[ds.UserDBField],
			"!" + "delegated_" + ds.UserDBField: domain.GetUserID(),
		}, false, utils.SpecialIDParam),
	}, false))
	if dels, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(
		ds.DBDelegation.Name, utils.ToListAnonymized(sqlFilter), false); err == nil && len(dels) > 0 {
		for _, delegated := range dels {
			newRec := record.Copy()
			newRec["binded_dbtask"] = id
			k1 := "delegated_" + ds.UserDBField
			k2 := ds.UserDBField

			ks1 := "shared_" + ds.UserDBField
			ks2 := ds.UserDBField
			newRec[ds.UserDBField] = delegated["delegated_"+ds.UserDBField]
			delete(newRec, utils.SpecialIDParam)
			if res, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
				utils.SpecialIDParam: newRec[ds.RequestDBField],
			}, false); err == nil && len(res) > 0 {
				CreateDelegated(newRec, res[0], utils.GetInt(newRec, utils.SpecialIDParam), domain)
			}
			share := map[string]interface{}{
				ks1:                  delegated[k1],
				ks2:                  delegated[k2],
				ds.SchemaDBField:     record[ds.SchemaDBField],
				ds.DestTableDBField:  record[ds.DestTableDBField],
				ds.DelegationDBField: delegated[utils.SpecialIDParam],
				"delete_access":      delegated["delete_access"],
			}

			arr := []interface{}{
				connector.FormatSQLRestrictionWhereByMap("", share, false),
			}
			arr = append(arr, "(('"+utils.GetString(delegated, "end_date")+"' > end_date AND '"+utils.GetString(delegated, "start_date")+"' <= end_date)  OR end_date IS NULL)")
			if res, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBShare.Name, arr, false); err == nil && len(res) == 0 {
				share["start_date"] = delegated["start_date"]
				share["end_date"] = delegated["end_date"]
				domain.GetDb().ClearQueryFilter().CreateQuery(ds.DBShare.Name, share, func(s string) (string, bool) { return "", true })
			}
			if request[ds.DestTableDBField] != share[ds.DestTableDBField] && request[ds.SchemaDBField] != share[ds.SchemaDBField] {
				delete(share, "start_date")
				delete(share, "end_date")
				share[ds.SchemaDBField] = request[ds.SchemaDBField]
				share[ds.DestTableDBField] = request[ds.DestTableDBField]
				arr := []interface{}{
					connector.FormatSQLRestrictionWhereByMap("", share, false),
				}
				arr = append(arr, "(()'"+utils.GetString(delegated, "end_date")+"' > end_date AND '"+utils.GetString(delegated, "start_date")+"' <= end_date) OR end_date IS NULL)")
				if res, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBShare.Name, arr, false); err == nil && len(res) == 0 {
					share["start_date"] = delegated["start_date"]
					share["end_date"] = delegated["end_date"]
					domain.GetDb().ClearQueryFilter().CreateQuery(ds.DBShare.Name, share, func(s string) (string, bool) { return "", true })
				}
			}
		}
	}
}

func UpdateDelegated(task utils.Record, request utils.Record, domain utils.DomainITF) {
	m := map[string]interface{}{
		"state": task["state"],
	}
	if task["closing_by"] != nil && task["closing_by"] != "" {
		m["closing_by"] = task["closing_by"]
	}
	if task["closing_comment"] != nil && task["closing_comment"] != "" {
		m["closing_comment"] = task["closing_comment"]
	}
	if task["closing_date"] != nil && task["closing_date"] != "" {
		m["closing_date"] = task["closing_date"]
	}
	if task["nexts"] != nil && task["nexts"] != "" {
		m["nexts"] = task["nexts"]
	}
	if task["is_close"] != nil && task["is_close"] != "" {
		m["is_close"] = task["is_close"]
	}
	id := task[utils.SpecialIDParam]
	if task["binded_dbtask"] != nil {
		id := task["binded_dbtask"]
		domain.GetDb().ClearQueryFilter().UpdateQuery(ds.DBTask.Name, m, map[string]interface{}{
			utils.SpecialIDParam: id,
		}, true)
	}
	domain.GetDb().ClearQueryFilter().UpdateQuery(ds.DBTask.Name, m, map[string]interface{}{
		"binded_dbtask": id,
	}, false)
}

func HandleHierarchicalVerification(domain utils.DomainITF, request utils.Record, record map[string]interface{}) map[string]interface{} {
	if utils.GetBool(request, "is_close") {
		return record
	}
	if hierarchy, err := GetHierarchical(domain); err == nil {
		for _, hierarch := range hierarchy {
			CreateHierarchicalTask(domain, request, record, hierarch)
		}
	}
	return record
}

func CreateHierarchicalTask(domain utils.DomainITF, request utils.Record, record, hierarch map[string]interface{}) {
	newTask := utils.Record{
		ds.SchemaDBField:    record[ds.SchemaDBField],
		ds.DestTableDBField: record[ds.DestTableDBField],
		ds.RequestDBField:   request[utils.SpecialIDParam],
		ds.UserDBField:      hierarch["parent_"+ds.UserDBField],
		"description":       "hierarchical verification expected by the system.",
		"urgency":           "normal",
		"priority":          "normal",
		sm.NAMEKEY:          "hierarchical verification",
	}
	if i, err := domain.GetDb().CreateQuery(ds.DBTask.Name, newTask, func(s string) (string, bool) {
		return "", true
	}); err == nil {
		CreateDelegated(newTask, request, i, domain)
		notify(newTask, i, domain)
	}
}
