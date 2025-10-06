package filter

import (
	"fmt"
	"slices"
	"sort"
	"sqldb-ws/domain/schema"
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
	fields := []string{}
	for _, field := range fromSchema.Fields {
		fields = append(fields, field.ID)
	}
	value := map[string]string{}
	if res, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBFieldCondition.Name, map[string]interface{}{
		ds.SchemaFieldDBField: fields,
	}, false); err == nil && len(res) > 0 {
		for _, cond := range res {
			if cond["from_"+ds.SchemaDBField] != nil && cond["from_"+ds.SchemaFieldDBField] != nil {
				if sche, err := sch.GetSchemaByID(utils.GetInt(cond, "from_"+ds.SchemaDBField)); err == nil {
					if field, err := sche.GetFieldByID(utils.GetInt(cond, "from_"+ds.SchemaFieldDBField)); err == nil {
						if p, ok := t.Domain.GetParams().Get(field.Name); ok {
							if p == "" && utils.GetBool(cond, "not_null") {
								return []map[string]interface{}{}
							} else if p != utils.GetString(cond, "value") && utils.GetString(cond, "value") != "" {
								return []map[string]interface{}{}
							}
							for _, f := range fields {
								value[f] = p
							}
							continue
						}
					}
				}
				return []map[string]interface{}{}
			} else if f, err := fromSchema.GetFieldByID(utils.GetInt(cond, ds.SchemaFieldDBField)); err != nil || (len(record) > 0 && record[f.Name] == nil && utils.GetBool(cond, "not_null")) || utils.GetString(record, f.Name) != utils.GetString(cond, "value") {
				return []map[string]interface{}{}
			} else {
				for _, ff := range fields {
					value[ff] = utils.ToString(record[f.Name])
				}
			}
		}
	}
	fmt.Println("DBFieldCondition", fromSchema.Name)
	if rr, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBFieldRule.Name, map[string]interface{}{
		ds.SchemaFieldDBField: fields,
		"starting_rule":       true,
	}, false); err == nil {
		for _, r := range rr {
			if r["value"] == nil || r["value"] == "" {
				r["value"] = value[utils.ToString(r[ds.SchemaFieldDBField])]
				if r["value"] == nil || r["value"] == "" {
					continue
				}
			}
			rules = append(rules, r)
		}
	}
	fmt.Println("DBFieldRule", rules)
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

func (t *FilterService) GetFieldSQL(key string, operator string, fromSchema *sm.SchemaModel, fromField *sm.FieldModel, rule map[string]interface{}, dest int64) string {
	if key == "" {
		key = "id"
	}
	rules := []map[string]interface{}{}
	if res, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBFieldRule.Name, map[string]interface{}{
		ds.FieldRuleDBField: rule[utils.SpecialIDParam],
	}, false); err == nil && len(res) > 0 {
		rules = res
	}
	if len(rules) > 0 {
		sql := ""
		for _, r := range rules {
			fieldName := "id"
			var fs *sm.SchemaModel
			var ff *sm.FieldModel
			if f, err := schema.GetFieldByID(utils.GetInt(r, ds.SchemaFieldDBField)); err == nil {
				fieldName = f.Name
				if ss, err := schema.GetSchemaByID(utils.ToInt64(f.SchemaID)); err == nil {
					fs = &ss
				}
			}
			if r["from_"+ds.SchemaDBField] != nil {
				if s, err := schema.GetSchemaByID(utils.GetInt(r, "from_"+ds.SchemaDBField)); err == nil {
					fs = &s
				}
			}
			if r["from_"+ds.SchemaFieldDBField] != nil {
				if f, err := schema.GetFieldByID(utils.GetInt(r, "from_"+ds.SchemaFieldDBField)); err == nil {
					ff = &f
				}
			}
			if r["value"] == nil && (rule["value"] != nil || rule["value"] != "") {
				r["value"] = rule["value"]
			}
			fromF := utils.SpecialIDParam
			if fromField != nil {
				fromF = fromField.Name
			}
			if fromSchema != nil {
				if len(sql) > 0 {
					if utils.GetString(r, "separator") != "" {
						sql += " " + strings.ToUpper(utils.ToString(r["separator"])) + " "
					} else {
						sql += " AND "
					}
				}
				sql += key + " " + operator + " " + "(SELECT " + fromF + " FROM " + fromSchema.Name + " WHERE " + t.GetFieldSQL(fieldName, utils.GetString(r, "operator"), fs, ff, r, utils.GetInt(r, ds.DestTableDBField)) + ")"
				continue
			}
			if len(sql) > 0 {
				if utils.GetString(r, "separator") != "" {
					sql += " " + utils.ToString(r["separator"]) + " "
				} else {
					sql += " AND "
				}
			}
			sql += key + " " + operator + " " + t.GetFieldSQL(fieldName, utils.GetString(r, "operator"), fs, ff, r, utils.GetInt(r, ds.DestTableDBField))
		}
		return "(" + sql + ")"
	} else {
		val := rule["value"]
		if fromSchema != nil && fromField != nil {
			if dest >= 0 {
				if res, err := t.Domain.GetDb().SelectQueryWithRestriction(
					fromSchema.Name, map[string]interface{}{utils.SpecialIDParam: dest}, false); err == nil && len(res) > 0 {
					val = res[0][fromField.Name]
				}
			}
		}
		if key == "id" || fromSchema == nil {
			return key + " " + operator + " " + utils.ToString(t.fromITF(val))
		} else if k, v, op, typ, link, err := fromSchema.GetTypeAndLinkForField(
			key, utils.ToString(t.fromITF(val)), operator, func(s string, search string) {}); err == nil {
			kk := utils.SpecialIDParam
			fmt.Println(typ, link, k, v, op)
			return "(SELECT " + kk + " FROM " + fromSchema.Name + " WHERE " + connector.MakeSqlItem("", typ, link, k, v, op) + ")"
		}
	}
	return ""
}

func (t *FilterService) GetFieldRestriction(fromSchema sm.SchemaModel) (string, error) {
	sql := ""
	for _, rule := range t.GetFieldCondition(fromSchema, utils.Record{}) { // SIMPLE WAY...
		fieldName := ""
		var fs *sm.SchemaModel
		var ff *sm.FieldModel
		if f, err := schema.GetFieldByID(utils.GetInt(rule, ds.SchemaFieldDBField)); err == nil {
			fieldName = f.Name
			if ss, err := schema.GetSchemaByID(utils.ToInt64(f.SchemaID)); err == nil {
				fs = &ss
			}
		}
		if rule["from_"+ds.SchemaDBField] != nil {
			if s, err := schema.GetSchemaByID(utils.GetInt(rule, "from_"+ds.SchemaDBField)); err == nil {
				fs = &s
			}
		}
		if rule["from_"+ds.SchemaFieldDBField] != nil {
			if f, err := schema.GetFieldByID(utils.GetInt(rule, "from_"+ds.SchemaFieldDBField)); err == nil {
				ff = &f
			}
		}
		if ss := utils.ToString(t.GetFieldSQL(fieldName, utils.GetString(rule, "operator"), fs, ff, rule, utils.GetInt(rule, ds.DestTableDBField))); ss != "" {
			if len(sql) > 0 {
				if utils.GetString(rule, "separator") != "" {
					sql += " " + strings.ToUpper(utils.ToString(rule["separator"])) + " "
				} else {
					sql += " AND "
				}
			}
			sql += ss
		}
	}
	fmt.Println("GetFieldRestriction", fromSchema.Name, sql)
	return sql, nil
}
