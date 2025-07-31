package filter

import (
	"slices"
	"sort"
	sch "sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	sm "sqldb-ws/domain/schema/models"
	"sqldb-ws/domain/utils"
	connector "sqldb-ws/infrastructure/connector/db"
	"strings"
)

// DONE - ~ 100 LINES - NOT TESTED
func (s *FilterService) GetFilterFields(viewfilterID string, schemaID string) []map[string]interface{} {
	if viewfilterID == "" {
		return []map[string]interface{}{}
	}
	restriction := map[string]interface{}{}
	if schemaID != "" {
		restriction[ds.SchemaFieldDBField] = s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(
			ds.DBSchemaField.Name,
			map[string]interface{}{ds.SchemaDBField: schemaID}, false, utils.SpecialIDParam)
	}
	restriction[ds.FilterDBField] = viewfilterID
	if fields, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBFilterField.Name, restriction, false); err == nil {
		sort.SliceStable(fields, func(i, j int) bool {
			return utils.ToInt64(fields[i]["index"]) <= utils.ToInt64(fields[j]["index"])
		})
		return fields
	}
	return []map[string]interface{}{}
}

func (s *FilterService) GetFilterForQuery(filterID string, viewfilterID string, schema sm.SchemaModel, domainParams utils.Params) (string, string, string, string, string) {
	view, order, dir := s.ProcessViewAndOrder(viewfilterID, schema.ID, domainParams)
	filter := s.ProcessFilterRestriction(filterID, schema)
	state := ""
	if filterID != "" {
		if fils, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBFilter.Name,
			map[string]interface{}{
				utils.SpecialIDParam: filterID,
			}, false); err == nil && len(fils) > 0 {
			state = utils.ToString(fils[0]["elder"]) // get elder filter
		}
	}
	return filter, view, order, dir, state
}

func (s *FilterService) ProcessFilterRestriction(filterID string, schema sm.SchemaModel) string {
	if filterID == "" {
		return ""
	}
	var filter []string
	var orFilter []string
	restriction := map[string]interface{}{
		ds.FilterDBField: filterID,
	}
	s.Domain.GetDb().ClearQueryFilter()
	fields, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBFilterField.Name, restriction, true)
	if err == nil && len(fields) > 0 {
		for _, field := range fields {
			if utils.GetBool(field, "is_task_concerned") {
				filter = append(filter, "("+connector.FormatSQLRestrictionWhereByMap("", map[string]interface{}{
					"!0": s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
						"is_close":          false,
						ds.SchemaDBField:    schema.ID,
						ds.DestTableDBField: "main.id",
					}, false, "COUNT(id)"),
					"!0_1": s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
						"is_close":          false,
						ds.SchemaDBField:    schema.ID,
						ds.DestTableDBField: "main.id",
					}, false, "COUNT(id)"),
				}, true)+")")
			}
			if f, err := schema.GetFieldByID(utils.GetInt(field, ds.SchemaFieldDBField)); err == nil {
				if utils.GetBool(field, "is_own") && len(s.RestrictionByEntityUser(schema, orFilter, true)) > 0 {
					if field["separator"] == "or" {
						orFilter = append(orFilter, s.RestrictionByEntityUser(schema, orFilter, true)...)
					} else {
						filter = append(filter, s.RestrictionByEntityUser(schema, filter, true)...)
					}
				} else if connector.FormatOperatorSQLRestriction(field["operator"], field["separator"], f.Name, field["value"], f.Type) != "" {
					if field["separator"] == "or" {
						orFilter = append(orFilter,
							"("+connector.FormatOperatorSQLRestriction(field["operator"], field["separator"], f.Name, field["value"], f.Type)+")")
					} else {
						filter = append(filter,
							"("+connector.FormatOperatorSQLRestriction(field["operator"], field["separator"], f.Name, field["value"], f.Type)+")")
					}
				}

			}
		}
	}
	if len(orFilter) > 0 {
		filter = append(filter, "("+strings.Join(orFilter, " OR ")+")")
	}
	return strings.Join(filter, " AND ")
}

func (s *FilterService) ProcessViewAndOrder(viewfilterID string, schemaID string, domainParams utils.Params) (string, string, string) {
	var viewFilter, order, dir []string = []string{}, []string{}, []string{}
	cols, ok := domainParams.Get(utils.RootColumnsParam)
	fields := []sm.FieldModel{}
	if viewfilterID != "" {
		for _, field := range s.GetFilterFields(viewfilterID, schemaID) {
			if f, err := sch.GetFieldByID(utils.GetInt(field, ds.RootID(ds.DBSchemaField.Name))); err == nil {
				fields = append(fields, f)
			}
		}
	}
	sort.SliceStable(fields, func(i, j int) bool {
		return fields[i].Index <= fields[j].Index
	})
	for _, field := range fields {
		if field.Name != "id" && (!ok || cols == "" || (strings.Contains(cols, field.Name))) && !field.Hidden {
			viewFilter = append(viewFilter, field.Name)
			if field.Dir != "" {
				dir = append(dir, strings.ToUpper(field.Dir))
			} else if !slices.Contains(order, field.Name) {
				dir = append(dir, field.Name+" ASC")
			}
		}
	}
	if p, ok := domainParams.Get(utils.RootGroupBy); ok {
		if len(viewFilter) != 0 && !slices.Contains(viewFilter, p) {
			viewFilter = append(viewFilter, p)
		}
		if !slices.Contains(order, p) {
			order = append(order, p)
			dir = append(dir, "ASC")
		}
	}
	return strings.Join(viewFilter, ","), strings.Join(order, ","), strings.Join(dir, ",")
}

func (d *FilterService) LifeCycleRestriction(tableName string, SQLrestriction []string, state string) []string {
	if state == "all" || tableName == ds.DBView.Name {
		return SQLrestriction
	}
	if state == "draft" {
		for _, restr := range SQLrestriction {
			restr = strings.ReplaceAll(restr, "is_draft=false OR", "")
			restr = strings.ReplaceAll(restr, "is_draft=false AND", "")
			restr = strings.ReplaceAll(restr, "AND is_draft=false", "")
			restr = strings.ReplaceAll(restr, "OR is_draft=false", "")
		}
		SQLrestriction = append(SQLrestriction, "is_draft=true")
	} else {
		k := utils.SpecialIDParam
		if state == "new" {
			k = "!" + k
		}
		SQLrestriction = append(SQLrestriction, connector.FormatSQLRestrictionWhereByMap("", map[string]interface{}{
			k: d.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBDataAccess.Name,
				map[string]interface{}{
					"write":  false,
					"update": false,
					ds.SchemaDBField: d.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(
						ds.DBSchema.Name, map[string]interface{}{
							"name": connector.Quote(tableName),
						}, false, "id"),
					ds.UserDBField: d.Domain.GetUserID(),
				}, false, ds.DestTableDBField),
		}, false))
	}
	return SQLrestriction
}
