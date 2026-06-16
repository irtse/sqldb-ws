package task_service

import (
	"fmt"
	"sqldb-ws/domain/schema"
	schserv "sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	"sqldb-ws/domain/schema/models"
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
	if task[ds.UserDBField] == "" {
		task[ds.UserDBField] = request[ds.UserDBField]
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
		res["state"] = "running"
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

func PrepareAndCreateTask(scheme utils.Record, request map[string]interface{}, record map[string]interface{}, domain utils.DomainITF, fromTask bool) {
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
		CreateTaskAndNotify(newTask, request, record, domain, fromTask)
	}
}

func CreateTaskAndNotify(task map[string]interface{}, request map[string]interface{}, initialRec map[string]interface{}, domain utils.DomainITF, isTask bool) int64 {
	if res, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
		ds.DestTableDBField: task[ds.DestTableDBField],
		ds.SchemaDBField:    task[ds.SchemaDBField],
		ds.RequestDBField:   task[ds.RequestDBField],
		"name":              connector.Quote(utils.GetString(task, "name")),
		"!state":            connector.Quote("dismiss"),
		ds.UserDBField:      task[ds.UserDBField],
	}, false); err == nil && len(res) == 0 {
		domain.GetDb().DeleteQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
			ds.DestTableDBField: task[ds.DestTableDBField],
			ds.SchemaDBField:    task[ds.SchemaDBField],
			ds.RequestDBField:   task[ds.RequestDBField],
			"name":              connector.Quote(utils.GetString(task, "name")),
			"state":             connector.Quote("dismiss"),
			ds.UserDBField:      task[ds.UserDBField],
		}, false)
		i, err := domain.GetDb().CreateQuery(ds.DBTask.Name, task, func(s string) (string, bool) {
			return "", true
		})
		if err != nil {
			return -1
		}
		CreateDelegated(task, request, i, initialRec, domain)
		Notify(task, i, domain)
		return i
	}
	return -1
}

func Notify(task utils.Record, i int64, domain utils.DomainITF) {
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
		if res, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBNotification.Name, map[string]interface{}{
			"name":              connector.Quote(utils.GetString(task, "name")),
			ds.DestTableDBField: notif[ds.DestTableDBField],
			"link_id":           schema.ID,
			ds.UserDBField:      task[ds.UserDBField],
			ds.EntityDBField:    task[ds.EntityDBField],
		}, false); err == nil && len(res) > 0 {
			return
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
	}, true)
}

func CreateDelegated(record utils.Record, request utils.Record, id int64, initialRec map[string]interface{}, domain utils.DomainITF) {
	if domain.GetUserID() == "" {
		return
	}
	currentTime := time.Now()
	bd := utils.GetString(initialRec, "binded_dbtask")
	if bd != "" {
		if res, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
			utils.SpecialIDParam: bd,
		}, false); err == nil && len(res) > 0 {
			newRec := record.Copy()
			newRec["binded_dbtask"] = nil
			newRec[ds.UserDBField] = res[0][ds.UserDBField]
			delete(newRec, utils.SpecialIDParam)
			if i := CreateTaskAndNotify(newRec, request, initialRec, domain, true); i >= 0 {
				domain.GetDb().ClearQueryFilter().UpdateQuery(ds.DBTask.Name, map[string]interface{}{
					"binded_dbtask": i,
				}, map[string]interface{}{
					utils.SpecialIDParam: id,
				}, false)
			}
		}
	}
	sqlFilter := []string{
		"('" + currentTime.Format("2006-01-02 15:04:05") + "' >= start_date AND ('" + currentTime.Format("2006-01-02 15:04:05") + "' < end_date OR end_date IS NULL))",
	}
	sqlFilter = append(sqlFilter, connector.FormatSQLRestrictionWhereByMap("", map[string]interface{}{
		"all_tasks": true,
		utils.SpecialIDParam: domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBDelegation.Name, map[string]interface{}{
			ds.UserDBField:                      record[ds.UserDBField],
			"!" + "delegated_" + ds.UserDBField: domain.GetUserID(),
		}, false, utils.SpecialIDParam),
	}, false))
	additionnalDels := GetWorkflowToDelegate(domain, record)
	if dels, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(
		ds.DBDelegation.Name, utils.ToListAnonymized(sqlFilter), false); err == nil && len(dels) > 0 {
		for _, d := range dels {
			if additionnalDels[utils.ToString(d["delegated_"+ds.UserDBField])] == nil {
				additionnalDels[utils.ToString(d["delegated_"+ds.UserDBField])] = d
			}
		}
	}
	for _, delegated := range additionnalDels {
		if delegated["delegated_"+ds.UserDBField] == record[ds.UserDBField] {
			continue
		}
		newRec := record.Copy()
		newRec["binded_dbtask"] = id
		k1 := "delegated_" + ds.UserDBField
		k2 := ds.UserDBField
		ks1 := "shared_" + ds.UserDBField
		ks2 := ds.UserDBField
		newRec[ds.UserDBField] = delegated["delegated_"+ds.UserDBField]
		delete(newRec, utils.SpecialIDParam)

		CreateTaskAndNotify(newRec, request, initialRec, domain, true)
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
		fmt.Println("DELEGATE", utils.GetString(delegated, "start_date"), utils.GetString(delegated, "end_date"))
		if utils.GetString(delegated, "end_date") == "" {
			arr = append(arr, "(('"+utils.GetString(delegated, "start_date")[0:10]+"' <= end_date)  OR end_date IS NULL)")
		} else {
			arr = append(arr, "(('"+utils.GetString(delegated, "end_date")[0:10]+"' > end_date AND '"+utils.GetString(delegated, "start_date")[0:10]+"' <= end_date)  OR end_date IS NULL)")
		}
		fmt.Println("SHARE", share)
		if res, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBShare.Name, arr, false); err == nil && len(res) == 0 {
			share["start_date"] = delegated["start_date"]
			share["end_date"] = delegated["end_date"]
			s, err := domain.GetDb().ClearQueryFilter().CreateQuery(ds.DBShare.Name, share, func(s string) (string, bool) { return "", true })
			fmt.Println("sharing pb", share, s, err)
		} else {
			fmt.Println("SHERE pb", err)
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
		if res, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
			utils.SpecialIDParam: id,
			"is_close":           false,
		}, false); err == nil && len(res) > 0 {
			m["binded_dbtask"] = id
			for _, r := range res {
				domain.GetDb().ClearQueryFilter().UpdateQuery(ds.DBTask.Name, m, map[string]interface{}{
					utils.SpecialIDParam: r[utils.SpecialIDParam],
				}, false)
				go UpdateDelegated(r, request, domain)
			}
		}
	}
	if res, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
		"binded_dbtask": id,
		"is_close":      false,
	}, false); err == nil && len(res) > 0 {
		m["binded_dbtask"] = id
		for _, r := range res {
			if sch, err := schserv.GetSchema(ds.DBTask.Name); err == nil {
				domain.GetDb().ClearQueryFilter().DeleteQueryWithRestriction(ds.DBNotification.Name, map[string]interface{}{
					ds.DestTableDBField: r[utils.SpecialIDParam],
					ds.SchemaDBField:    sch.ID,
				}, false)
			}
			domain.GetDb().ClearQueryFilter().UpdateQuery(ds.DBTask.Name, m, map[string]interface{}{
				utils.SpecialIDParam: r[utils.SpecialIDParam],
			}, false)
			go UpdateDelegated(r, request, domain)
		}
	}
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

func CreateHierarchicalTask(domain utils.DomainITF, request utils.Record, record map[string]interface{}, hierarch map[string]interface{}) {
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
		CreateDelegated(newTask, request, i, record, domain)
		Notify(newTask, i, domain)
	}
}

func GetWorkflowToDelegate(domain utils.DomainITF, task utils.Record) map[string]map[string]interface{} {
	users := map[string]map[string]interface{}{}
	wf, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBWorkflow.Name, map[string]interface{}{
		utils.SpecialIDParam: domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBWorkflowSchema.Name, map[string]interface{}{
			utils.SpecialIDParam: utils.ToString(task[ds.WorkflowSchemaDBField]),
		}, false, ds.WorkflowDBField),
	}, false)
	if err != nil {
		return users
	}
	if wn, err := schema.GetSchema(ds.DBWorkflow.Name); err == nil {
		for _, w := range wf {
			for k, v := range GetUserToDelegate(domain, task, wn.ID, utils.ToString(w[utils.SpecialIDParam]), utils.ToString(task[ds.UserDBField])) {
				users[k] = v
			}
		}
	}
	return users
}

func GetUserToDelegate(domain utils.DomainITF, record map[string]interface{}, schemaID string, id string, from string) map[string]map[string]interface{} {
	users := map[string]map[string]interface{}{}
	now := time.Now().UTC()
	start := "('" + now.Format("2006-01-02 15:04:05") + "' >= start_date"
	end := "('" + now.Format("2006-01-02 15:04:05") + "' < end_date OR end_date IS NULL))"

	founded, err := domain.GetDb().SelectQueryWithRestriction(ds.DBUserDelegation.Name, map[string]interface{}{
		ds.SchemaDBField:       schemaID,
		ds.DestTableDBField:    id,
		"hierarchy_delegation": false,
	}, false)

	if err == nil && len(founded) > 0 {
		for _, f := range founded {
			if f["field"] == nil || f["value"] == nil || utils.ToString(record[utils.ToString(f["field"])]) == utils.ToString(f["value"]) {
				if users[utils.ToString(f[ds.UserDBField])] == nil {
					users[utils.ToString(f[ds.UserDBField])] = map[string]interface{}{
						"delegated_" + ds.UserDBField: utils.ToString(f[ds.UserDBField]),
						ds.UserDBField:                from,
						"start_date":                  time.Now().Add(-1 * time.Hour),
						"delete_access":               utils.GetBool(f, "delete_access"),
					}
				}

			}
			if users[utils.ToString(f[ds.UserDBField])] == nil {
				users[utils.ToString(f[ds.UserDBField])] = map[string]interface{}{
					"delegated_" + ds.UserDBField: utils.ToString(f[ds.UserDBField]),
					ds.UserDBField:                from,
					"start_date":                  time.Now().UTC().Add(-1 * time.Hour),
					"delete_access":               utils.GetBool(f, "delete_access"),
				}
			}
		}
	}

	hierarchDels, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUserDelegation.Name, map[string]interface{}{
		ds.SchemaDBField:       schemaID,
		ds.DestTableDBField:    id,
		"hierarchy_delegation": true,
	}, false)
	if err == nil && len(hierarchDels) > 0 { // only need one
		for _, hd := range hierarchDels {
			if hd["field"] != nil && hd["value"] != nil && utils.ToString(record[utils.ToString(hd["field"])]) == utils.ToString(hd["value"]) {
				if users[utils.ToString(hd[ds.UserDBField])] == nil {
					users[utils.ToString(hd[ds.UserDBField])] = map[string]interface{}{
						"delegated_" + ds.UserDBField: utils.ToString(hd[ds.UserDBField]),
						ds.UserDBField:                from,
						"start_date":                  time.Now().Add(-1 * time.Hour),
						"delete_access":               utils.GetBool(hd, "delete_access"),
					}
				}

			} else {
				foundedHierarch, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBHierarchy.Name, map[string]interface{}{
					ds.UserDBField:  domain.GetUserID(), // TODO : moi même
					models.STARTKEY: []string{start},
					models.ENDKEY:   []string{end},
				}, false)
				if err == nil && len(foundedHierarch) > 0 {
					for _, f := range foundedHierarch {
						if users[utils.ToString(f[utils.SpecialIDParam])] == nil {
							users[utils.ToString(f[utils.SpecialIDParam])] = map[string]interface{}{
								"delegated_" + ds.UserDBField: utils.ToString(f["parent_"+ds.UserDBField]),
								ds.UserDBField:                from,
								"start_date":                  time.Now().Add(-1 * time.Hour),
								"delete_access":               utils.GetBool(hierarchDels[0], "delete_access"),
							}
						}
					}
				}
				break
			}
		}
	}
	roleDels, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBRoleDelegation.Name, map[string]interface{}{
		ds.SchemaDBField:    schemaID,
		ds.DestTableDBField: id,
	}, false)
	if err == nil && len(roleDels) > 0 {
		for _, rd := range roleDels {
			foundedRole, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
				utils.SpecialIDParam: domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRoleAttribution.Name, map[string]interface{}{
					ds.RoleDBField:  utils.ToString(rd[ds.RoleDBField]),
					models.STARTKEY: []string{start},
					models.ENDKEY:   []string{end},
					"is_hierarch":   utils.GetBool(rd, "hierarchy_delegation"),
				}, true, ds.UserDBField),
				utils.SpecialIDParam + "_1": domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBEntityUser.Name, map[string]interface{}{
					utils.SpecialIDParam: domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRoleAttribution.Name, map[string]interface{}{
						ds.RoleDBField:  utils.ToString(rd[ds.RoleDBField]),
						models.STARTKEY: []string{start},
						models.ENDKEY:   []string{end},
						"is_hierarch":   utils.GetBool(rd, "hierarchy_delegation"),
					}, false, ds.EntityDBField),
				}, true, ds.UserDBField),
			}, false)
			if err == nil && len(foundedRole) > 0 {
				for _, f := range foundedRole {
					if users[utils.ToString(f[utils.SpecialIDParam])] == nil {
						users[utils.ToString(f[utils.SpecialIDParam])] = map[string]interface{}{
							"delegated_" + ds.UserDBField: utils.ToString(f[utils.SpecialIDParam]),
							ds.UserDBField:                from,
							"start_date":                  time.Now().Add(-1 * time.Hour),
							"delete_access":               utils.GetBool(rd, "delete_access"),
						}
					}
				}
			}
		}
	}
	entDels, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBEntityDelegation.Name, map[string]interface{}{
		ds.SchemaDBField:    schemaID,
		ds.DestTableDBField: id,
	}, false)
	if err == nil && len(entDels) > 0 {
		for _, rd := range entDels {
			foundedEnt, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBEntityUser.Name, map[string]interface{}{
				ds.EntityDBField: utils.ToString(rd[ds.EntityDBField]),
				models.STARTKEY:  []string{start},
				models.ENDKEY:    []string{end},
				"is_hierarch":    utils.GetBool(rd, "hierarchy_delegation"),
			}, false)
			if err == nil && len(foundedEnt) > 0 {
				for _, f := range foundedEnt {
					if users[utils.ToString(f[ds.UserDBField])] == nil {
						users[utils.ToString(f[ds.UserDBField])] = map[string]interface{}{
							"delegated_" + ds.UserDBField: utils.ToString(f[ds.UserDBField]),
							ds.UserDBField:                from,
							"start_date":                  time.Now().Add(-1 * time.Hour),
							"delete_access":               utils.GetBool(rd, "delete_access"),
						}
					}
				}
			}
		}
	}
	return users
}

// TODO : - Create Delegation - Update Delegation - Delete Delegation
// TODO : - les perms
