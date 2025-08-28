package schema_service

import (
	"fmt"
	"runtime"
	"slices"
	"sort"
	filterserv "sqldb-ws/domain/domain_service/filter"
	"sqldb-ws/domain/domain_service/history"
	"sqldb-ws/domain/domain_service/view_convertor"
	schserv "sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	"sqldb-ws/domain/schema/models"
	sm "sqldb-ws/domain/schema/models"
	servutils "sqldb-ws/domain/specialized_service/utils"
	"sqldb-ws/domain/utils"
	connector "sqldb-ws/infrastructure/connector/db"
	"strings"
)

// DONE - ~ 200 LINES - NOT TESTED
type ViewService struct {
	servutils.AbstractSpecializedService
}

func NewViewService() utils.SpecializedServiceITF {
	return &ViewService{}
}

func (s *ViewService) Entity() utils.SpecializedServiceInfo { return ds.DBView }

func (s *ViewService) VerifyDataIntegrity(record map[string]interface{}, tablename string) (map[string]interface{}, error, bool) {
	if rec, err, ok := servutils.CheckAutoLoad(tablename, record, s.Domain); ok {
		return s.AbstractSpecializedService.VerifyDataIntegrity(rec, tablename)
	} else {
		return record, err, false
	}
}

func (s *ViewService) GenerateQueryFilter(tableName string, innerestr ...string) (string, string, string, string) {
	if !s.Domain.IsSuperAdmin() {
		innerestr = append(innerestr, "only_super_admin=false")
	}
	restr, _, _, _ := filterserv.NewFilterService(s.Domain).GetQueryFilter(tableName, s.Domain.GetParams().Copy(), innerestr...)
	return restr, "", "", ""
}

func (s *ViewService) TransformToGenericView(results utils.Results, tableName string, dest_id ...string) (res utils.Results) {
	runtime.GOMAXPROCS(5)
	params := s.Domain.GetParams().Copy()
	schemas := []*models.SchemaModel{}
	if len(results) == 1 && !utils.GetBool(results[0], "is_empty") && !s.Domain.IsShallowed() {
		if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBViewSchema.Name, map[string]interface{}{
			ds.ViewDBField: results[0][utils.SpecialIDParam],
		}, false); err == nil {
			for _, r := range res {
				if sch, err := schserv.GetSchemaByID(utils.GetInt(r, ds.SchemaDBField)); err == nil {
					schemas = append(schemas, &sch)
				}
			}
		}
	}

	channel := make(chan utils.Record, len(results))
	for _, record := range results {
		go s.TransformToView(record, false, nil, params, channel, dest_id...)
	}
	for range results {
		if rec := <-channel; rec != nil {
			res = append(res, rec)
		}
	}
	if len(res) <= 1 && len(schemas) > 0 && !s.Domain.GetEmpty() && !s.Domain.IsShallowed() {
		subChan := make(chan utils.Record, len(schemas))
		for _, schema := range schemas {
			s.TransformToView(results[0], true, schema, params, subChan, dest_id...)
		}
		for _, schema := range schemas {
			newSchema := map[string]interface{}{}
			for k, v := range res[0]["schema"].(map[string]interface{}) {
				if schema.HasField(k) {
					newSchema[k] = v
				}
			}
			typ := models.ViewFieldModel{
				Label:    "type",
				Type:     "enum__" + strings.ReplaceAll(schema.Label, "_", " "),
				Index:    2,
				Readonly: true,
				Active:   true,
			}
			if utils.ToMap(res[0]["schema"])["type"] == nil {
				newSchema["type"] = typ
			} else {
				typ = utils.ToMap(res[0]["schema"])["type"].(models.ViewFieldModel)
				typ.Type += "_" + strings.ReplaceAll(schema.Label, "_", " ")
				newSchema["type"] = typ
			}
			res[0]["schema"] = newSchema
		}
		res[0]["order"] = append([]interface{}{"type"}, utils.ToList(res[0]["order"])...)
		for range schemas {
			if rec := <-subChan; rec != nil {
				for _, i := range utils.ToList(rec["items"]) {
					res[0]["items"] = append(utils.ToList(res[0]["items"]), i)
				}
				res[0]["new"] = utils.GetInt(res[0], "new") + utils.GetInt(rec, "new")
				res[0]["max"] = utils.GetInt(res[0], "max") + utils.GetInt(rec, "max")
			}
		}
	}
	sort.SliceStable(res, func(i, j int) bool {
		return utils.ToInt64(res[i]["index"]) <= utils.ToInt64(res[j]["index"])
	})
	return
}

func (s *ViewService) TransformToView(record utils.Record, multiple bool, schema *models.SchemaModel, domainParams utils.Params,
	channel chan utils.Record, dest_id ...string) {
	s.Domain.SetOwn(record.GetBool("own_view"))
	if schema == nil {
		if s, err := schserv.GetSchemaByID(utils.GetInt(record, ds.SchemaDBField)); err == nil {
			schema = &s
		}
	}

	dp := domainParams.Copy()
	if schema == nil {
		channel <- nil
	} else {
		notFound := false
		if line, ok := domainParams.Get(utils.RootFilterLine); ok {
			if val, operator := connector.GetFieldInInjection(line, "type"); val != "" {
				if strings.Contains(operator, "LIKE") {
					if strings.Contains(operator, "NOT") && (strings.Contains(schema.Label, val) || strings.Contains(schema.Name, val)) {
						notFound = true
					} else if !strings.Contains(schema.Label, val) && !strings.Contains(schema.Name, val) {

					}
				} else if schema.Name != val && schema.Label != val {
					notFound = true
				}
				dp.Set(utils.RootFilterLine, connector.DeleteFieldInInjection(line, "type"))
			}
		}
		// retrive additionnal view to combine to the main... add a type can be filtered by a filter line
		// add type onto order and schema plus verify if filter not implied.
		// may regenerate to get limits... for file... for type and for dest_table_id if needed.
		s.Domain.HandleRecordAttributes(record)
		rec := NewViewFromRecord(*schema, record)
		s.addFavorizeInfo(record, rec)

		params := utils.GetRowTargetParameters(schema.Name, s.combineDestinations(dest_id))
		params = params.EnrichCondition(dp.Values, func(k string) bool {
			_, ok := params.Values[k]
			return !ok && k != "new" && !strings.Contains(k, "dest_table") && k != "id"
		})
		sqlFilter, view, dir := s.getFilterDetails(record, schema)
		params.UpdateParamsWithFilters(view, dir)
		params.EnrichCondition(dp.Values, func(k string) bool {
			return k != utils.RootRowsParam && k != utils.SpecialIDParam && k != utils.RootTableParam
		})
		dp.Delete(func(k string) bool {
			return k == utils.RootRowsParam || k == utils.SpecialIDParam || k == utils.RootTableParam || k == utils.SpecialSubIDParam
		})
		if _, ok := record["group_by"]; ok {
			if field, err := schema.GetFieldByID(record.GetInt("group_by")); err == nil {
				params.Set(utils.RootGroupBy, field.Name)
			}
		}
		if f, ok := dp.Get(utils.RootGroupBy); ok {
			params.Set(utils.RootGroupBy, f)
			rec["group_by"] = f
		}
		datas := utils.Results{}
		if shal, ok := s.Domain.GetParams().Get(utils.RootShallow); (!ok || shal != "enable") && !notFound {
			params, datas, rec["max"] = s.fetchData(schema.Name, params, sqlFilter)
		}
		newOrder := strings.Split(view, ",")
		record, rec, newOrder = s.processData(rec, multiple, datas, schema, record, newOrder, params)
		if len(newOrder) > 0 {
			rec["order"] = newOrder
		}
		// complexify to verify if a request is active about... then... to redirect on...
		rec["export_path"] = "/" + utils.MAIN_PREFIX + "/" + fmt.Sprintf(ds.DBView.Name) + "?rows=" + utils.ToString(record[utils.SpecialIDParam])
		rec["link_path"] = "/" + utils.MAIN_PREFIX + "/" + fmt.Sprintf(ds.DBView.Name) + "?rows=" + utils.ToString(record[utils.SpecialIDParam])
		if _, ok := record["group_by"]; ok { // express by each column we are foldered TODO : if not in view add it
			field, err := schema.GetFieldByID(record.GetInt("group_by"))
			if err == nil {
				rec["group_by"] = field.Name
			}
		}
		if f, ok := domainParams.Get(utils.RootGroupBy); ok {
			rec["group_by"] = f
		}
		if !multiple && !slices.Contains(utils.ToList(rec["actions"]), "delete") {
			if s.Domain.IsOwn(false, false, s.Domain.GetMethod()) {
				rec["actions"] = append(utils.ToList(rec["actions"]), "delete")
			}
		}
		channel <- rec
	}
}

// this filter a view only with its property
func (s *ViewService) getFilter(rec utils.Record, record utils.Record, values map[string]interface{}, schema *sm.SchemaModel) (utils.Record, utils.Record, map[string]interface{}) {
	if record[ds.FilterDBField] != nil && s.Domain.GetEmpty() {
		if fields, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBFilterField.Name, map[string]interface{}{
			ds.FilterDBField + "_1": s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBFilter.Name, map[string]interface{}{
				"is_view":              false,
				"dashboard_restricted": false,
			}, false, utils.SpecialIDParam),
			ds.FilterDBField: record[ds.FilterDBField],
		}, false); err == nil && len(fields) > 0 {
			for _, f := range fields {
				ff, err := schema.GetFieldByID(utils.GetInt(f, ds.SchemaFieldDBField))
				if err != nil {
					continue
				}
				if val, ok := utils.ToMap(rec["schema"])[ff.Name]; ok {
					utils.ToMap(val)["readonly"] = true
					values[ff.Name] = f["value"]
				}
			}
		}
	}
	return rec, record, values
}

func (s *ViewService) addFavorizeInfo(record utils.Record, rec utils.Record) utils.Record {
	rec["favorize_body"] = utils.Record{
		ds.ViewDBField: record.GetInt(utils.SpecialIDParam),
		ds.UserDBField: s.Domain.GetUserID(),
	}
	rec["favorize_path"] = fmt.Sprintf("/%s/%s?%s=%s",
		utils.MAIN_PREFIX, ds.DBViewAttribution.Name, utils.RootRowsParam, utils.ReservedParam)

	attributions, _ := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(
		ds.DBViewAttribution.Name,
		map[string]interface{}{
			ds.UserDBField: s.Domain.GetUserID(),
			ds.ViewDBField: record[utils.SpecialIDParam],
		}, false)
	rec["is_favorize"] = len(attributions) > 0
	return rec
}

func (s *ViewService) combineDestinations(dest_id []string) string {
	return strings.Join(dest_id, ",")
}

func (s *ViewService) getFilterDetails(record utils.Record, schema *models.SchemaModel) (string, string, string) {
	filter := utils.GetString(record, ds.FilterDBField)
	viewFilter := utils.GetString(record, ds.ViewFilterDBField)
	sqlFilter, view, _, dir, _ := filterserv.NewFilterService(s.Domain).GetFilterForQuery(
		filter, viewFilter, *schema, s.Domain.GetParams())
	return sqlFilter, view, dir
}
func (s *ViewService) fetchData(tablename string, params utils.Params, sqlFilter string) (utils.Params, utils.Results, int64) {
	datas := utils.Results{}
	max := int64(0)
	if !s.Domain.GetEmpty() {
		sqlrestr, sqlorder, sqllimit, sqlview := filterserv.NewFilterService(s.Domain).GetQueryFilter(tablename, params, sqlFilter)
		max, _ = history.CountMaxDataAccess(tablename, []string{sqlrestr}, s.Domain)
		s.Domain.GetDb().ClearQueryFilter()
		s.Domain.GetDb().SetSQLView(sqlview)
		s.Domain.GetDb().SetSQLOrder(sqlorder)
		s.Domain.GetDb().SetSQLLimit(sqllimit)
		s.Domain.GetDb().SetSQLRestriction(sqlrestr)
		dd, _ := s.Domain.GetDb().SelectQueryWithRestriction(tablename, map[string]interface{}{}, false)
		for _, d := range dd {
			datas = append(datas, d)
		}
		//datas, _ = s.Domain.Call(params.RootRaw(), utils.Record{}, utils.SELECT, []interface{}{sqlFilter}...)
	}
	return params, datas, max
}
func (s *ViewService) processData(rec utils.Record, multiple bool, datas utils.Results, schema *sm.SchemaModel,
	record utils.Record, newOrder []string, params utils.Params) (utils.Record, utils.Record, []string) {
	if utils.Compare(record["is_empty"], true) {
		datas = append(datas, utils.Record{})
	}
	if !s.Domain.IsShallowed() {
		treated := utils.Results{}
		if !multiple {
			treated = view_convertor.NewViewConvertor(s.Domain).TransformToView(datas, schema.Name, false, params)
		} else {
			treated = view_convertor.NewViewConvertor(s.Domain).TransformMultipleSchema(datas, schema, false, params)
		}

		if len(treated) > 0 {
			if !multiple {
				rec["schema"] = s.extractSchema(utils.ToMap(treated[0]["schema"]), record, schema, params, newOrder)
			}
			for k, v := range treated[0] {
				if v != nil {
					switch k {
					case "items":
						rec, newOrder = s.extractItems(utils.ToList(v), k, rec, record, schema, params, false)
					default:
						if recValue, exists := rec[k]; !exists || recValue == "" {
							rec[k] = v
						}
					}
				}
			}
		}
	}
	return record, rec, newOrder
}

func (s *ViewService) extractSchema(value map[string]interface{}, record utils.Record, schema *sm.SchemaModel,
	params utils.Params, newOrder []string) map[string]interface{} {
	newV := map[string]interface{}{}
	for fieldName, field := range value {
		if fieldName != ds.WorkflowDBField && schema.Name == ds.DBRequest.Name && utils.Compare(record["is_empty"], true) {
			continue
		}
		col, ok := params.Get(utils.RootColumnsParam)
		utils.ToMap(field)["active"] = !ok || col == "" || slices.Contains(newOrder, fieldName) || strings.Contains(col, fieldName)
		newV[fieldName] = field
	}
	return newV
}

func (s *ViewService) extractItems(value []interface{}, key string, rec utils.Record, record utils.Record,
	schema *sm.SchemaModel, params utils.Params, main bool) (utils.Record, []string) {
	newOrder := []string{}
	for _, item := range value {
		values := utils.ToMap(item)["values"]
		utils.ToMap(item)["schema_id"] = schema.ID
		utils.ToMap(values)["type"] = schema.Label
		if len(s.Domain.DetectFileToSearchIn()) > 0 {
			for search, field := range s.Domain.DetectFileToSearchIn() {
				if utils.ToMap(values)[field] == nil || !utils.SearchInFile(utils.GetString(utils.ToMap(values), field), search) {
					continue
				}
			}
		}
		if line, ok := params.Get(utils.RootFilterLine); ok {
			if val, operator := connector.GetFieldInInjection(line, ds.DestTableDBField); val != "" && utils.GetString(utils.ToMap(values), ds.DestTableDBField) != "" {
				if schemaDest, err := schserv.GetSchemaByID(utils.GetInt(utils.ToMap(values), ds.SchemaDBField)); err == nil {
					cmd := "name" + operator + val
					if strings.Contains(operator, "LIKE") {
						cmd = "name::text " + operator + val
					}
					if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(schemaDest.Name, []string{
						"id = " + utils.GetString(utils.ToMap(values), ds.DestTableDBField), cmd,
					}, false); err == nil && len(res) == 0 {
						continue
					}
				}
			}
		}
		if !main {
			utils.ToMap(values)["type"] = schema.Label
		}
		if utils.Compare(rec["is_list"], true) {
			path := utils.RootRowsParam
			if strings.Contains(path, ds.DBView.Name) {
				path = utils.RootDestTableIDParam
			}
			utils.ToMap(item)["link_path"] = fmt.Sprintf("/%s/%s?%s=%v", utils.MAIN_PREFIX, schema.Name,
				utils.RootRowsParam, utils.ToMap(values)[utils.SpecialIDParam])
		}
		rec, record, values = s.getFilter(rec, record, utils.ToMap(values), schema)
		newOrder, values = view_convertor.GetOrder(schema, record, utils.ToMap(values), newOrder, s.Domain)
		// here its to format by filter depending on task running about this document of viewing, if enable.
	}
	if rec[key] == nil {
		rec[key] = value
	} else {
		rec[key] = append(rec[key].([]interface{}), value...)
	}
	return rec, newOrder
}
