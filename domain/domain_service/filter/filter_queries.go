package filter

import (
	"fmt"
	"slices"
	"sort"
	sch "sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	sm "sqldb-ws/domain/schema/models"
	"sqldb-ws/domain/utils"
	connector "sqldb-ws/infrastructure/connector/db"
	"strconv"
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

func (t *FilterService) GetFieldCondition(fromSchema sm.SchemaModel, record utils.Record) []map[string]interface{} {
	rules := []map[string]interface{}{}
	for _, field := range fromSchema.Fields {
		if res, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBFieldCondition.Name, map[string]interface{}{
			ds.SchemaFieldDBField: field.ID,
		}, false); err == nil && len(res) > 0 {
			for _, cond := range res {
				if cond[ds.SchemaFieldDBField] == nil && utils.GetString(record, utils.SpecialIDParam) != utils.GetString(cond, "value") {
					return []map[string]interface{}{}
				}
				if f, err := fromSchema.GetFieldByID(utils.GetInt(cond, ds.SchemaFieldDBField)); err != nil || (record[f.Name] == nil && utils.GetBool(cond, "not_null")) || utils.GetString(record, f.Name) != utils.GetString(cond, "value") {
					return []map[string]interface{}{}
				}
			}
		}
		if rr, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBFieldRule.Name, map[string]interface{}{
			ds.SchemaFieldDBField: field.ID,
		}, false); err != nil {
			rules = append(rules, rr...)
		}
	}
	return rules
}

func (t *FilterService) fromITF(val interface{}) interface{} {
	if slices.Contains([]string{"true", "false"}, utils.ToString(val)) {
		return val == "true" // should set type
	} else if i, err := strconv.Atoi(utils.ToString(val)); err == nil && i >= 0 {
		return i // should set type
	} else {
		return utils.ToString(val) // should set type
	}
}
func (t *FilterService) GetFieldSQL(fromSchema sm.SchemaModel, fromField *sm.FieldModel, rule map[string]interface{}, dest int64) string {
	SQLRestriction := ""
	if val, ok := rule["value"]; ok && val != nil { // SIMPLE WAY...
		if fromField != nil {
			k, v, op, typ, link, err := fromSchema.GetTypeAndLinkForField(
				fromField.Name, utils.ToString(t.fromITF(val)), utils.GetString(rule, "operator"), func(s string, search string) {})
			if err == nil {
				SQLRestriction += connector.MakeSqlItem("", typ, link, k, v, op)
			}
		} else {
			k, v, op, typ, link, err := fromSchema.GetTypeAndLinkForField(
				utils.SpecialIDParam, utils.ToString(t.fromITF(val)), utils.GetString(rule, "operator"), func(s string, search string) {})
			if err == nil {
				SQLRestriction += connector.MakeSqlItem("", typ, link, k, v, op)
			}
		}
	}
	if rule[ds.FieldRuleDBField] == nil {
		m := map[string]interface{}{}
		if dest != -1 {
			m[utils.SpecialIDParam] = dest
		}

		key := utils.SpecialIDParam
		if fromField != nil {
			key = fromField.Name
		}
		val := ""
		if s, ok := t.Domain.GetParams().Get(fromSchema.Name + "." + key); ok {
			val = s
		} else if res, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(fromSchema.Name, m, false); err == nil && len(res) > 0 {
			val = utils.GetString(res[0], key)
		} else {
			return SQLRestriction
		}

		k, v, op, typ, link, err := fromSchema.GetTypeAndLinkForField(
			key, utils.ToString(t.fromITF(val)), utils.GetString(rule, "operator"), func(s string, search string) {})
		if err == nil {
			if len(SQLRestriction) > 0 {
				SQLRestriction += " " + utils.GetString(rule, "separator") + " "
			}
			SQLRestriction += connector.MakeSqlItem("", typ, link, k, v, op)
		}
	} else {
		if res, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.FieldRuleDBField, map[string]interface{}{
			utils.SpecialIDParam: rule[ds.FieldRuleDBField],
		}, false); err == nil && len(res) > 0 {
			r := res[0]
			newRestr := ""
			if schFrom, err := sch.GetSchemaByID(utils.ToInt64(r["from_"+ds.SchemaDBField])); err == nil {
				o := "IN"
				if strings.Contains(utils.GetString(r, "operator"), "!") {
					o = "NOT " + o
				}
				if ff, err := schFrom.GetFieldByID(utils.GetInt(r, "from_"+ds.SchemaFieldDBField)); err == nil {
					sql := t.GetFieldSQL(schFrom, &ff, r, utils.GetInt(r, "from_"+ds.DestTableDBField))
					if sql != "" {
						if len(SQLRestriction) > 0 {
							SQLRestriction += " " + utils.GetString(rule, "separator") + " "
						}
						if fromField != nil {
							SQLRestriction = SQLRestriction + " " + o + " (SELECT " + ff.Name + " FROM " + schFrom.Name + " WHERE " + sql + ")"
						} else {
							SQLRestriction = SQLRestriction + "id " + o + " (SELECT " + ff.Name + " FROM " + schFrom.Name + " WHERE " + sql + ")"
						}
					}

				} else {
					sql := t.GetFieldSQL(schFrom, nil, r, utils.GetInt(r, "from_"+ds.DestTableDBField))
					if sql != "" {
						if len(SQLRestriction) > 0 {
							SQLRestriction += " " + utils.GetString(rule, "separator") + " "
						}
						if fromField != nil {
							SQLRestriction = SQLRestriction + fromField.Name + " " + o + " (SELECT id FROM " + schFrom.Name + " WHERE " + sql + ")"
						} else {
							SQLRestriction = SQLRestriction + "id " + o + " (SELECT id FROM " + schFrom.Name + " WHERE " + sql + ")"
						}
					}
				}
			} else if val, ok := r["value"]; ok && val != nil {
				if fromField == nil {
					newRestr += utils.SpecialIDParam + utils.ToString(r["operator"]) + utils.ToString(val)
				} else {
					k, v, op, typ, link, err := fromSchema.GetTypeAndLinkForField(
						fromField.Name, utils.ToString(t.fromITF(val)), utils.GetString(r, "operator"), func(s string, search string) {})
					if err == nil {
						if len(SQLRestriction) > 0 {
							SQLRestriction += " " + utils.GetString(rule, "separator") + " "
						}
						SQLRestriction += connector.MakeSqlItem("", typ, link, k, v, op)
					}
				}
			}
		}
	}
	fmt.Println("SQLRestriction", SQLRestriction)
	return SQLRestriction
}

func (t *FilterService) GetFieldRestriction(fromSchema sm.SchemaModel) (string, error) {
	sql := ""
	for _, rule := range t.GetFieldCondition(fromSchema, utils.Record{}) {
		f, err := fromSchema.GetFieldByID(utils.GetInt(rule, ds.SchemaFieldDBField))
		if err == nil {
			if val, ok := rule["value"]; ok && val != nil { // SIMPLE WAY...
				if len(sql) > 0 {
					sql = " " + utils.ToString(rule["separator"]) + " "
				}
				sql += f.Name + " " + utils.ToString(rule["operator"]) + " " + utils.ToString(t.fromITF(val))
			} else if schFrom, err := sch.GetSchemaByID(utils.ToInt64(rule["from_"+ds.SchemaDBField])); err == nil {
				if len(sql) > 0 {
					sql = " " + utils.ToString(rule["separator"]) + " "
				}
				if ff, err := schFrom.GetFieldByID(utils.GetInt(rule, "from_"+ds.SchemaFieldDBField)); err == nil {
					sql = f.Name + " " + utils.ToString(rule["operator"]) + " " + t.GetFieldSQL(schFrom, &ff, rule, utils.GetInt(rule, "from_"+ds.DestTableDBField))
				} else {
					sql = f.Name + " " + utils.ToString(rule["operator"]) + " " + t.GetFieldSQL(schFrom, nil, rule, utils.GetInt(rule, "from_"+ds.DestTableDBField))
				}
			}
		}
	}
	return sql, nil
}
