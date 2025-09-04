package task_service

import (
	"sqldb-ws/domain/domain_service/filter"
	"sqldb-ws/domain/domain_service/view_convertor"
	"sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	servutils "sqldb-ws/domain/specialized_service/utils"
	utils "sqldb-ws/domain/utils"
)

// DONE - UNDER 100 LINES - NOT TESTED
type WorkflowService struct {
	servutils.AbstractSpecializedService
}

func NewWorkflowService() utils.SpecializedServiceITF {
	return &WorkflowService{}
}

func (s *WorkflowService) Entity() utils.SpecializedServiceInfo { return ds.DBWorkflow }

func (s *WorkflowService) TransformToGenericView(results utils.Results, tableName string, dest_id ...string) utils.Results {
	res := utils.Results{}
	for _, rec := range results { // filter by allowed schemas
		schema, err := schema.GetSchemaByID(utils.ToInt64(rec[SchemaDBField]))
		if err == nil && s.Domain.VerifyAuth(schema.Name, "", "", utils.CREATE) {
			res = append(res, rec)
		}
	}
	rr := view_convertor.NewViewConvertor(s.Domain).TransformToView(res, tableName, true, s.Domain.GetParams().Copy())
	if _, ok := s.Domain.GetParams().Get(utils.SpecialIDParam); ok && len(results) == 1 && len(rr) == 1 {
		r := results[0]
		if i, ok := r["view_"+ds.FilterDBField]; ok && i != nil {
			schema := rr[0]["schema"].(map[string]interface{})
			newSchema := map[string]interface{}{}
			if fields, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBSchemaField.Name,
				map[string]interface{}{
					utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBFilterField.Name,
						map[string]interface{}{
							ds.FilterDBField: i,
						}, false, ds.SchemaFieldDBField),
				}, false); err == nil {
				for _, f := range fields {
					newSchema[utils.GetString(f, "name")] = schema[utils.GetString(f, "name")]
				}
			}
			rr[0]["schema"] = newSchema
		}
	}
	vc := view_convertor.NewViewConvertor(s.Domain)
	for _, r := range rr {
		r["rules"] = vc.GetFieldsRules(utils.ToString(rr[0]["schema_name"]), r)
	}
	return rr
}

func (s *WorkflowService) GenerateQueryFilter(tableName string, innerestr ...string) (string, string, string, string) {
	s1, s2, s3, s4 := filter.NewFilterService(s.Domain).GetQueryFilter(tableName, s.Domain.GetParams().Copy(), innerestr...)
	return s1, s2, s3, s4
}

func (s *WorkflowService) VerifyDataIntegrity(record map[string]interface{}, tablename string) (map[string]interface{}, error, bool) {
	if rec, err, ok := servutils.CheckAutoLoad(tablename, record, s.Domain); ok {
		return s.AbstractSpecializedService.VerifyDataIntegrity(rec, tablename)
	} else {
		return record, err, false
	}
}
