package favorite_service

import (
	"fmt"
	"sort"
	"sqldb-ws/domain/domain_service/filter"
	"sqldb-ws/domain/domain_service/view_convertor"
	schserv "sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	sm "sqldb-ws/domain/schema/models"
	servutils "sqldb-ws/domain/specialized_service/utils"
	utils "sqldb-ws/domain/utils"
	connector "sqldb-ws/infrastructure/connector/db"
	"strconv"
	"strings"
)

// DONE - ~ 200 LINES - PARTIALLY TESTED
type FilterService struct {
	servutils.AbstractSpecializedService
	Fields       []map[string]interface{}
	UpdateFields bool
}

func NewFilterService() utils.SpecializedServiceITF {
	return &FilterService{}
}

func (s *FilterService) Entity() utils.SpecializedServiceInfo { return ds.DBFilter }
func (s *FilterService) SpecializedDeleteRow(results []map[string]interface{}, tableName string) {
	for _, record := range results {
		s.Domain.GetDb().DeleteQueryWithRestriction(ds.DBFilterField.Name, map[string]interface{}{
			ds.FilterDBField: utils.ToString(record[utils.SpecialIDParam]),
		}, false)
	}
}

func (s *FilterService) SpecializedUpdateRow(results []map[string]interface{}, record map[string]interface{}) {
	s.Write(record, ds.DBFilter.Name)
	s.AbstractSpecializedService.SpecializedUpdateRow(results, record)
}
func (s *FilterService) SpecializedCreateRow(record map[string]interface{}, tableName string) {
	s.Write(record, tableName)
	s.AbstractSpecializedService.SpecializedCreateRow(record, tableName)
}

func (s *FilterService) Write(record utils.Record, tableName string) {
	if s.UpdateFields {
		s.Domain.GetDb().ClearQueryFilter().DeleteQueryWithRestriction(ds.DBFilterField.Name, map[string]interface{}{
			ds.FilterDBField: record[utils.SpecialIDParam],
		}, false)
	}

	for _, field := range s.Fields {
		if schema, err := schserv.GetSchemaByID(utils.ToInt64(record[ds.SchemaDBField])); err == nil && field["name"] != nil {
			field[ds.FilterDBField] = record[utils.SpecialIDParam]
			f, err := schema.GetField(utils.ToString(field["name"]))
			delete(field, utils.SpecialIDParam)
			if err == nil {
				field[ds.SchemaFieldDBField] = f.ID
			}
			s.Domain.GetDb().ClearQueryFilter().CreateQuery(ds.DBFilterField.Name, field, func(v string) (string, bool) {
				return "", true
			})
		}
	}
}

func (s *FilterService) TransformToGenericView(results utils.Results, tableName string, dest_id ...string) (res utils.Results) {
	selected := make(map[string]bool)
	for _, rec := range results { // memorize selected filters
		id := rec.GetString(utils.SpecialIDParam)
		selected[id] = rec["is_selected"] == nil || utils.Compare(rec["is_selected"], true)
	}
	for _, rec := range view_convertor.NewViewConvertor(s.Domain).TransformToView(results, tableName, true, s.Domain.GetParams().Copy()) { // transform to generic view
		if rec == nil {
			continue
		}
		rec["is_selected"] = selected[rec.GetString(utils.SpecialIDParam)] // restore selected filters
		schema, err := schserv.GetSchemaByID(rec.GetInt("schema_id"))
		if fields, err2 := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction( // get filter fields
			ds.DBFilterField.Name,
			map[string]interface{}{ds.FilterDBField: rec.GetInt(utils.SpecialIDParam)},
			false,
		); err == nil && err2 == nil { // sort fields by index
			sort.SliceStable(fields, func(i, j int) bool {
				return utils.ToInt64(fields[i]["index"]) <= utils.ToInt64(fields[j]["index"])
			})
			filterFields := []sm.FilterModel{} // add fields to filter
			for _, field := range fields {
				if ff, err := schema.GetFieldByID(utils.GetInt(field, ds.SchemaFieldDBField)); err == nil {
					model := sm.FilterModel{
						ID:        utils.GetInt(rec, utils.SpecialIDParam),
						Name:      ff.Name,
						Label:     ff.Label,
						Index:     float64(utils.ToInt64(field["index"])),
						Type:      ff.Type,
						Value:     utils.ToString(field["value"]),
						Separator: utils.ToString(field["separator"]),
						Operator:  utils.ToString(field["operator"]),
						Dir:       utils.ToString(field["dir"]),
					}
					if width, err := strconv.ParseFloat(utils.ToString(field["width"]), 64); err == nil {
						model.Width = width
					}
					filterFields = append(filterFields, model)
				} else if field[ds.SchemaFieldDBField] == nil {
					model := sm.FilterModel{
						Name:      "id",
						Label:     "id",
						Index:     0,
						Type:      "integer",
						Value:     utils.ToString(field["value"]),
						Separator: utils.ToString(field["separator"]),
						Operator:  utils.ToString(field["operator"]),
						Dir:       utils.ToString(field["dir"]),
					}
					if field["name"] != nil {
						model.Name = utils.ToString(field["name"])
						model.Label = utils.ToString(field["name"])
					}
					if width, err := strconv.ParseFloat(utils.ToString(field["width"]), 64); err == nil {
						model.Width = width
					}
					filterFields = append(filterFields, model)
				}
			}
			rec["filter_fields"] = filterFields
			if rec["elder"] == nil { // get elder filter
				if fils, _ := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBFilter.Name,
					map[string]interface{}{"id": rec[utils.SpecialIDParam]}, false); len(fils) > 0 {
					rec["elder"] = fils[0]["elder"]
				} else {
					rec["elder"] = "all"
				}
			}
		}
		res = append(res, rec)
	}
	return
}

func (s *FilterService) GenerateQueryFilter(tableName string, innerestr ...string) (string, string, string, string) {
	if !strings.Contains(strings.Join(innerestr, ","), "dashboard_restricted") {
		innerestr = append(innerestr, "dashboard_restricted=false") // add dashboard_restricted filter if not present AD
	}
	innerestr = append(innerestr, "hidden=false") // add dashboard_restricted filter if not present AD
	return filter.NewFilterService(s.Domain).GetQueryFilter(tableName, s.Domain.GetParams().Copy(), false, innerestr...)
}

func (s *FilterService) VerifyDataIntegrity(record map[string]interface{}, tablename string) (map[string]interface{}, error, bool) {
	record["hidden"] = false
	s.UpdateFields = false
	method := s.Domain.GetMethod()
	if method != utils.DELETE {
		if err := s.ProcessLink(record); err != nil {
			return record, err, false
		}
		s.ProcessName(record)
		s.Fields = make([]map[string]interface{}, 0)
		s.ProcessFields(record, "view_fields")
		s.ProcessFields(record, "filter_fields")

		if method == utils.CREATE {
			s.HandleCreate(record)
		}
	}
	if method == utils.UPDATE {
		delete(record, "name")
	}
	s.ProcessSelection(record)
	return s.AbstractSpecializedService.VerifyDataIntegrity(record, tablename)
}

func (s *FilterService) ProcessLink(record map[string]interface{}) error {
	if link, ok := record["link"]; ok && link != nil && link != "" {
		schema, err := schserv.GetSchema(utils.ToString(link))
		delete(record, "link")
		if err != nil {
			return err
		}
		record[ds.SchemaDBField] = schema.ID
	}
	return nil
}

func (s *FilterService) ProcessName(record map[string]interface{}) {
	if name, ok := record["name"]; ok {
		if result, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBFilter.Name, map[string]interface{}{
			ds.SchemaDBField: record[ds.SchemaDBField],
			"name":           connector.Quote(utils.ToString(name)),
		}, false); err == nil && len(result) > 0 {
			record[utils.SpecialIDParam] = result[0][utils.SpecialIDParam]
		}
	}
}

func (s *FilterService) ProcessFields(record map[string]interface{}, fieldType string) {
	if fields, ok := record[fieldType]; ok {
		s.UpdateFields = true
		for _, field := range utils.ToList(fields) {
			s.Fields = append(s.Fields, utils.ToMap(field))
		}
	}
}

func (s *FilterService) HandleCreate(record map[string]interface{}) {
	name := utils.GetString(record, sm.NAMEKEY)
	if _, ok := record["view_fields"]; ok { // is a view filter
		name += "view "
		record["is_view"] = true
	}
	if schemaID := record[ds.SchemaDBField]; schemaID != nil {
		schema, _ := schserv.GetSchemaByID(utils.ToInt64(schemaID))
		if _, ok := record[ds.DBEntity.Name]; !ok {
			s.HandleUserFilterNaming(record, schema, &name)
		} else {
			s.HandleEntityFilterNaming(record, schema, &name)
		}
	}
	record[sm.NAMEKEY] = name
}

func (s *FilterService) HandleUserFilterNaming(record map[string]interface{}, schema sm.SchemaModel, name *string) {
	if s.Domain.GetAutoload() {
		return
	}
	record[ds.UserDBField] = s.Domain.GetUserID()
	if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBFilter.Name, map[string]interface{}{
		ds.UserDBField:   s.Domain.GetUserID(),
		ds.SchemaDBField: schema.ID,
	}, false); err == nil {
		*name += fmt.Sprintf("%s filter n°%d", schema.Label, len(res)+1)
	}
}

func (s *FilterService) HandleEntityFilterNaming(record map[string]interface{}, schema sm.SchemaModel, name *string) {
	if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBFilter.Name, map[string]interface{}{
		ds.EntityDBField: utils.GetString(record, ds.EntityDBField),
		ds.SchemaDBField: schema.ID,
	}, false); err == nil {
		s.RecursiveHandleEntityFilterNaming(schema.Label, len(res)+1, name)
	}
}

func (s *FilterService) RecursiveHandleEntityFilterNaming(label string, index int, name *string) {
	if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBFilter.Name, map[string]interface{}{
		"name": fmt.Sprintf("%s filter n°%d", label, index),
	}, false); err == nil && len(res) == 0 {
		*name += fmt.Sprintf("%s filter n°%d", label, index)
	} else {
		s.RecursiveHandleEntityFilterNaming(label, index+1, name)
	}
}

func (s *FilterService) ProcessSelection(record map[string]interface{}) {
	/*if sel, ok := record["is_selected"]; ok && utils.Compare(sel, true) { // TODO
		s.Domain.GetDb().UpdateQuery(ds.DBFilter.Name, utils.Record{
			"is_selected": false,
		}, map[string]interface{}{
			ds.FilterDBField: record[ds.FilterDBField],
		}, true)
	}*/
	delete(record, "filter_fields")
	delete(record, "view_fields")
}
