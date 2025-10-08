package task_service

import (
	"errors"
	"fmt"
	"sqldb-ws/domain/domain_service/filter"
	"sqldb-ws/domain/domain_service/view_convertor"
	"sqldb-ws/domain/schema"
	schserv "sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	sm "sqldb-ws/domain/schema/models"
	servutils "sqldb-ws/domain/specialized_service/utils"
	"sqldb-ws/domain/utils"
	connector "sqldb-ws/infrastructure/connector/db"
)

type RequestService struct {
	servutils.AbstractSpecializedService
}

func NewRequestService() utils.SpecializedServiceITF {
	return &RequestService{}
}

func (s *RequestService) TransformToGenericView(results utils.Results, tableName string, dest_id ...string) utils.Results {
	// TODO: here send back my passive task...
	res := view_convertor.NewViewConvertor(s.Domain).TransformToView(results, tableName, true, s.Domain.GetParams().Copy())
	if len(results) == 1 && s.Domain.GetMethod() == utils.CREATE {
		// retrieve... tasks affected to you
		if r, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
			ds.RequestDBField: results[0][utils.SpecialIDParam],
			"is_close":        false,
			utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
				ds.UserDBField: s.Domain.GetUserID(),
				ds.EntityDBField: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBEntityUser.Name, map[string]interface{}{
					ds.UserDBField: s.Domain.GetUserID(),
				}, false, ds.EntityDBField),
			}, true, utils.SpecialIDParam),
		}, false); err == nil && len(r) > 0 {
			if sch, err := schema.GetSchema(ds.DBTask.Name); err == nil {
				res[0]["inner_redirection"] = utils.BuildPath(sch.ID, utils.GetString(r[0], utils.SpecialIDParam))
			}
		} else if sch, err := schema.GetSchemaByID(utils.GetInt(results[0], ds.SchemaDBField)); err == nil {
			res[0]["inner_redirection"] = utils.BuildPath(sch.ID, utils.GetString(results[0], ds.DestTableDBField))
		}
	} // inner_redirection is the way to redirect any closure... to next data or data
	return res
}
func (s *RequestService) GenerateQueryFilter(tableName string, innerestr ...string) (string, string, string, string) {
	n := []string{}
	f := filter.NewFilterService(s.Domain)
	if !s.Domain.IsSuperCall() {
		n = append(n, "("+connector.FormatSQLRestrictionWhereByMap("", map[string]interface{}{
			ds.UserDBField: s.Domain.GetUserID(),
			ds.UserDBField + "_1": s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBHierarchy.Name, map[string]interface{}{
				"parent_" + ds.UserDBField: s.Domain.GetUserID(),
			}, true, ds.UserDBField),
		}, true)+")")
	}
	n = append(n, innerestr...)
	return f.GetQueryFilter(tableName, s.Domain.GetParams().Copy(), n...)
}

func GetHierarchical(domain utils.DomainITF) ([]map[string]interface{}, error) {
	return domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBHierarchy.Name, map[string]interface{}{
		ds.UserDBField: domain.GetUserID(),
		ds.EntityDBField: domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(
			ds.DBEntityUser.Name,
			map[string]interface{}{
				ds.UserDBField: domain.GetUserID(),
			}, true, ds.EntityDBField),
	}, true)
}

func (s *RequestService) Entity() utils.SpecializedServiceInfo                                    { return ds.DBRequest }
func (s *RequestService) SpecializedDeleteRow(results []map[string]interface{}, tableName string) {}
func (s *RequestService) VerifyDataIntegrity(record map[string]interface{}, tablename string) (map[string]interface{}, error, bool) {
	fmt.Println("VERIFY", record)
	if s.Domain.GetMethod() == utils.CREATE {
		if _, ok := record[utils.RootDestTableIDParam]; !ok {
			return record, errors.New("missing related data"), false
		}
		record[ds.UserDBField] = s.Domain.GetUserID()
		if hierarchy, err := GetHierarchical(s.Domain); err != nil || len(hierarchy) > 0 {
			record["current_index"] = 0
			if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBWorkflowSchema.Name, map[string]interface{}{
				"index":                          1,
				"before_hierarchical_validation": true,
				ds.WorkflowDBField:               record[ds.WorkflowDBField],
			}, false); err == nil && len(res) == 0 {
				record["current_index"] = 1
			}
		} else {
			record["current_index"] = 1
		}
		if wf, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBWorkflow.Name, map[string]interface{}{
			utils.SpecialIDParam: record[ds.WorkflowDBField],
		}, false); err != nil || len(wf) == 0 {
			return record, nil, true
		} else {
			record["name"] = wf[0][sm.NAMEKEY]
			record[ds.SchemaDBField] = wf[0][ds.SchemaDBField]
		}

	} else if s.Domain.GetMethod() == utils.UPDATE {
		record = SetClosureStatus(record)
	}
	if s.Domain.GetMethod() != utils.DELETE {
		if rec, err, ok := servutils.CheckAutoLoad(tablename, record, s.Domain); ok {
			return s.AbstractSpecializedService.VerifyDataIntegrity(rec, tablename)
		} else {
			return record, err, false
		}
	}
	fmt.Println("VERIFY 2", record)
	return record, nil, true
}
func (s *RequestService) SpecializedUpdateRow(results []map[string]interface{}, record map[string]interface{}) {
	if _, ok := record["is_draft"]; ok && utils.GetBool(record, "is_draft") {
		return
	}
	s.AbstractSpecializedService.SpecializedUpdateRow(results, record)
	for _, rec := range results {
		p := utils.AllParams(ds.DBNotification.Name)
		p.Set(ds.UserDBField, utils.ToString(rec[ds.UserDBField]))
		p.Set(ds.DestTableDBField, utils.ToString(rec[utils.SpecialIDParam]))
		switch rec["state"] {
		case "dismiss":
		case "refused":
			rec["state"] = "refused"
			p.Set(sm.NAMEKEY, "Rejected "+utils.GetString(rec, sm.NAMEKEY))
			p.Set("description", utils.GetString(rec, sm.NAMEKEY)+" is rejected and closed.")
		case "completed":
			p.Set(sm.NAMEKEY, "Validated "+utils.GetString(rec, sm.NAMEKEY))
			p.Set("description", utils.GetString(rec, sm.NAMEKEY)+" is accepted and closed.")
		}
		schema, err := schserv.GetSchema(ds.DBRequest.Name)
		if err == nil && !utils.Compare(rec["is_meta"], true) && CheckStateIsEnded(rec["state"]) {
			if t, err := s.Domain.SuperCall(p.RootRaw(), utils.Record{}, utils.SELECT, false); err == nil && len(t) > 0 {
				return
			}
			p.SimpleDelete(utils.RootTableParam)
			p.SimpleDelete(utils.RootRowsParam)
			rec := p.Anonymized()
			rec["link_id"] = schema.ID
			s.Domain.GetDb().ClearQueryFilter().CreateQuery(ds.DBNotification.Name, rec, func(s string) (string, bool) { return s, true })
		}
		if utils.Compare(rec["is_close"], true) {
			p := utils.AllParams(ds.DBTask.Name)
			p.Set("meta_"+ds.RequestDBField, utils.ToString(rec[utils.SpecialIDParam]))
			res, err := s.Domain.SuperCall(p.RootRaw(), utils.Record{}, utils.SELECT, false)
			if err == nil && len(res) > 0 {
				for _, task := range res {
					task := SetClosureStatus(task)
					s.Domain.UpdateSuperCall(utils.AllParams(ds.DBTask.Name).RootRaw(), task)
				}
			}
		}
	}
}

func (s *RequestService) Write(record utils.Record, tableName string) {
	if utils.GetBool(record, "is_draft") {
		return
	}
	if utils.GetInt(record, "current_index") == 0 {
		found := false
		if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBWorkflowSchema.Name, map[string]interface{}{
			"index":            1,
			ds.WorkflowDBField: record[ds.WorkflowDBField],
		}, false); err == nil {
			for _, rec := range res {
				if utils.GetBool(rec, "before_hierarchical_validation") {
					found = true
					break
				}
			}
		}
		if found {
			record["current_index"] = 0.9
			record = HandleHierarchicalVerification(s.Domain, record, record)
		} else {
			record["current_index"] = 1
		}
	}
	if utils.GetInt(record, "current_index") == 1 {
		s.handleInitialWorkflow(record)
	}
}

func (s *RequestService) SpecializedCreateRow(record map[string]interface{}, tableName string) {
	s.Write(record, tableName)
	s.AbstractSpecializedService.SpecializedCreateRow(record, tableName)
}

func (s *RequestService) handleInitialWorkflow(record map[string]interface{}) {
	wfs, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBWorkflowSchema.Name, map[string]interface{}{
		"index":            1,
		ds.WorkflowDBField: record[ds.WorkflowDBField],
	}, false)
	if err != nil || len(wfs) == 0 {
		s.Domain.GetDb().DeleteQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
			utils.SpecialIDParam: utils.GetString(record, utils.SpecialIDParam),
		}, false)
		return
	}

	for _, newTask := range wfs {
		PrepareAndCreateTask(newTask, record, record, s.Domain, false)
	}
}
