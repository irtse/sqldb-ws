package filter

import (
	"errors"
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
func (s *FilterService) GetFilterDelete(restr []string, schema sm.SchemaModel, domainParams utils.Params) []string {
	if s.Domain.GetMode() != "delete" || schema.Name == ds.DBView.Name {
		return restr
	}
	perms := 0
	if s.Domain.VerifyAuth(schema.Name, "", "", utils.DELETE) {
		perms = 1
	}
	subM := map[string]interface{}{
		utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
			ds.DestTableDBField: "main.id",
			ds.SchemaDBField:    schema.ID,
			"is_close":          false,
			utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
				ds.DestTableDBField: "main.id",
				ds.SchemaDBField:    schema.ID,
				"is_close":          false,
				ds.UserDBField:      s.Domain.GetUserID(),
			}, false, ds.RequestDBField),
		}, false, utils.SpecialIDParam),
	}
	if schema.HasField(ds.DestTableDBField) {
		subM[utils.SpecialIDParam+"_1"] = s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
			ds.DestTableDBField: "main." + ds.DestTableDBField,
			ds.SchemaDBField:    "main." + ds.SchemaDBField,
			"is_close":          false,
			utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
				ds.DestTableDBField: "main.id",
				ds.SchemaDBField:    schema.ID,
				"is_close":          false,
				ds.UserDBField:      s.Domain.GetUserID(),
			}, false, ds.RequestDBField),
		}, false, "COUNT("+utils.SpecialIDParam+")")
	}
	subMH := map[string]interface{}{
		"!0_1": perms,
		"0":    s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, subM, false, utils.SpecialIDParam),
		utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBDataAccess.Name, map[string]interface{}{
			ds.DestTableDBField: "main.id",
			ds.SchemaDBField:    schema.ID,
		}, false, utils.SpecialIDParam),
	}
	mH := map[string]interface{}{
		"write":              true,
		ds.UserDBField:       s.Domain.GetUserID(),
		utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBDataAccess.Name, subMH, true, utils.SpecialIDParam),
	}
	if schema.HasField(ds.DestTableDBField) {
		subMH[utils.SpecialIDParam+"_1"] = s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBDataAccess.Name, map[string]interface{}{
			ds.DestTableDBField: "main." + ds.DestTableDBField,
			ds.SchemaDBField:    "main." + ds.SchemaDBField,
		}, false, utils.SpecialIDParam)
	}
	restr = append(restr, "("+connector.FormatSQLRestrictionWhereByMap("", map[string]interface{}{
		"is_draft": true,
		"!0":       s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBDataAccess.Name, mH, false, "COUNT(id)"),
	}, true)+")")
	return restr
}
func (s *FilterService) GetFilterEdit(restr []string, schema sm.SchemaModel, domainParams utils.Params) []string {
	if s.Domain.GetMode() != "edit" || schema.Name == ds.DBView.Name {
		return restr
	}
	if schema.Name == ds.DBTask.Name {
		restr = append(restr, connector.FormatSQLRestrictionWhereByMap("", map[string]interface{}{
			"is_draft": true,
			utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
				utils.SpecialIDParam: "main.id",
				"is_close":           false,
				ds.UserDBField:       s.Domain.GetUserID(),
			}, false, utils.SpecialIDParam),
		}, true))
	} else {
		perms := 0
		if s.Domain.VerifyAuth(schema.Name, "", "", s.Domain.GetMethod()) {
			perms = 1
		}
		subMH := map[string]interface{}{
			"!0_1": perms,
			utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBDataAccess.Name, map[string]interface{}{
				ds.DestTableDBField: "main.id",
				ds.SchemaDBField:    schema.ID,
			}, false, utils.SpecialIDParam),
		}
		subM := map[string]interface{}{
			utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
				ds.DestTableDBField: "main.id",
				ds.SchemaDBField:    schema.ID,
				utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
					ds.DestTableDBField: "main.id",
					ds.SchemaDBField:    schema.ID,
					ds.UserDBField:      s.Domain.GetUserID(),
				}, false, ds.RequestDBField),
			}, false, utils.SpecialIDParam),
		}
		if schema.HasField(ds.DestTableDBField) {
			subM[utils.SpecialIDParam+"_1"] = s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
				ds.DestTableDBField: "main." + ds.DestTableDBField,
				ds.SchemaDBField:    "main." + ds.SchemaDBField,
			}, false, utils.SpecialIDParam)

			subMH[utils.SpecialIDParam+"_1"] = s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBDataAccess.Name, map[string]interface{}{
				ds.DestTableDBField: "main." + ds.DestTableDBField,
				ds.SchemaDBField:    "main." + ds.SchemaDBField,
			}, false, utils.SpecialIDParam)
		}
		mH := map[string]interface{}{
			"write":              true,
			ds.UserDBField:       s.Domain.GetUserID(),
			utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBDataAccess.Name, subMH, true, utils.SpecialIDParam),
		}

		restr = append(restr, "("+connector.FormatSQLRestrictionWhereByMap("", map[string]interface{}{
			"is_draft": true,
			"!0": s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
				"is_close": false,
				utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
					ds.DestTableDBField: "main." + ds.DestTableDBField,
					ds.SchemaDBField:    "main." + ds.SchemaDBField,
					ds.UserDBField:      s.Domain.GetUserID(),
				}, false, ds.RequestDBField),
				utils.SpecialIDParam + "_1": s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, subM, true, utils.SpecialIDParam),
			}, false, "COUNT(id)"),
			"0_1": s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
				"0": s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
					"is_close":           true,
					utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, subM, true, utils.SpecialIDParam),
				}, false, "COUNT(id)"),
				"!0_1": s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBDataAccess.Name, mH, false, "COUNT(id)"),
			}, false, "COUNT(id)"),
			"0_2": s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
				"0": s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
					"is_close":           true,
					utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, subM, true, utils.SpecialIDParam),
				}, false, "COUNT(id)"),
			}, false, "COUNT(id)"),
		}, true)+")")
	}
	return restr
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
			if utils.GetBool(field, "is_task_concerned") && s.Domain.GetMode() != "edit" && s.Domain.GetMode() != "delete" {
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
	condi, errC := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBFieldCondition.Name, map[string]interface{}{
		ds.SchemaFieldDBField: fields,
	}, false)
	if errC != nil || len(condi) == 0 {
		return rules
	}
	if rr, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBFieldRule.Name, map[string]interface{}{
		ds.SchemaFieldDBField: fields,
		"starting_rule":       true,
	}, false); err == nil {
		for _, r := range rr {
			for _, cond := range condi {
				if cond["from_"+ds.SchemaDBField] != nil {
					if sche, err := sch.GetSchemaByID(utils.GetInt(cond, "from_"+ds.SchemaDBField)); err == nil && cond["from_"+ds.SchemaFieldDBField] != nil {
						if field, err := sche.GetFieldByID(utils.GetInt(cond, "from_"+ds.SchemaFieldDBField)); err == nil {
							if p, ok := t.Domain.GetParams().Get(utils.GetString(r, utils.SpecialIDParam) + "_" + field.Name); ok {
								if (p == "" && utils.GetBool(cond, "not_null")) || (p != utils.GetString(cond, "value") && utils.GetString(cond, "value") != "") {
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
			if r["value"] == nil || r["value"] == "" {
				r["value"] = value[utils.ToString(r[ds.SchemaFieldDBField])]
			}
			rules = append(rules, r)
		}
	}
	return rules
}

func (t *FilterService) fromITF(val interface{}) (string, string) {
	if val == nil || val == "null" {
		return "null", ""
	}
	if slices.Contains([]string{"true", "false"}, utils.ToString(val)) {
		return "bool", utils.ToString(val == "true") // should set type
	} else if i, err := strconv.Atoi(utils.ToString(val)); err == nil && i >= 0 {
		return "float", utils.ToString(i) // should set type
	} else {
		return "string", utils.ToString(val) // should set type
	}
}

func (t *FilterService) GetFieldSQL(key string, operator string, basefromSchema *sm.SchemaModel, fromSchema *sm.SchemaModel, fromField *sm.FieldModel, rule map[string]interface{}, dest int64) (map[string]map[string][]string, string) {
	if key == "" {
		key = "id"
	}
	rules := []map[string]interface{}{}
	if res, err := t.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBFieldRule.Name, map[string]interface{}{
		ds.FieldRuleDBField: rule[utils.SpecialIDParam],
	}, false); err == nil && len(res) > 0 {
		rules = res
	}
	m := map[string]map[string][]string{}
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
				if m[key] == nil {
					m[key] = map[string][]string{}
				}
				if m[key][operator] == nil {
					m[key][operator] = []string{}
				}
				_, ff := t.GetFieldSQL(fieldName, utils.GetString(r, "operator"), fromSchema, fs, ff, r, utils.GetInt(r, ds.DestTableDBField))
				vv := "(SELECT " + fromF + " FROM " + fromSchema.Name + " WHERE " + ff + ")"
				if ff == "" {
					vv = "(SELECT " + fromF + " FROM " + fromSchema.Name + ")"
				}
				m[key][operator] = append(m[key][operator], vv)
				sql += key + " " + operator + " " + vv
				continue
			}
			if len(sql) > 0 {
				if utils.GetString(r, "separator") != "" {
					sql += " " + utils.ToString(r["separator"]) + " "
				} else {
					sql += " AND "
				}
			}
			if m[key] == nil {
				m[key] = map[string][]string{}
			}
			if m[key][operator] == nil {
				m[key][operator] = []string{}
			}
			_, t := t.GetFieldSQL(fieldName, utils.GetString(r, "operator"), fromSchema, fs, ff, r, utils.GetInt(r, ds.DestTableDBField))
			m[key][operator] = append(m[key][operator], t)
			sql += key + " " + operator + " " + t
		}
		return m, "(" + sql + ")"
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
		typ, value := t.fromITF(val)
		if value == "" {
			return m, ""
		} else if key == "id" || fromSchema == nil {
			if m[key] == nil {
				m[key] = map[string][]string{}
			}
			if m[key][operator] == nil {
				m[key][operator] = []string{}
			}
			m[key][operator] = append(m[key][operator], value)
			if strings.Contains(typ, "string") {
				if strings.Contains(value, ",") {
					arr := []string{}
					for _, a := range strings.Split(value, ",") {
						arr = append(arr, connector.Quote(a))
					}
					if strings.Contains(operator, "!") {
						return m, "(" + key + " NOT IN (" + strings.Join(arr, ",") + "))"
					}
					return m, "(" + key + " IN (" + strings.Join(arr, ",") + "))"
				} else {
					return m, "(" + key + " " + operator + " " + connector.Quote(value) + ")"
				}
			} else {
				if strings.Contains(value, ",") {
					arr := []string{}
					for _, a := range strings.Split(value, ",") {
						arr = append(arr, a)
					}
					if strings.Contains(operator, "!") {
						return m, "(" + key + " NOT IN (" + strings.Join(arr, ",") + "))"
					}
					return m, "(" + key + " IN (" + strings.Join(arr, ",") + "))"
				} else {
					return m, "(" + key + " " + operator + " " + value + ")"
				}
			}
		} else {
			subArr := []string{}
			for _, vvv := range strings.Split(value, ",") {
				if k, v, op, typ, link, err := fromSchema.GetTypeAndLinkForField(key, vvv, operator, func(s string, search string) {}); err == nil {
					if basefromSchema != nil && basefromSchema.Name == fromSchema.Name {
						kk, opp, sql, _ := connector.MakeSqlItem("", typ, link, k, v, op)
						if m[kk] == nil {
							m[kk] = map[string][]string{}
						}
						if m[kk][opp] == nil {
							m[kk][opp] = []string{}
						}
						m[kk][opp] = append(m[kk][opp], sql)
						subArr = append(subArr, "("+kk+" "+opp+" "+sql+")")
					} else {
						kk := utils.SpecialIDParam
						if m[kk] == nil {
							m[kk] = map[string][]string{}
						}
						if m[kk][op] == nil {
							m[kk][op] = []string{}
						}
						_, _, _, sql := connector.MakeSqlItem("", typ, link, k, v, op)
						subArr = append(subArr, k+" "+op+" "+"(SELECT "+kk+" FROM "+fromSchema.Name+" WHERE "+sql+")")
						m[kk][op] = append(m[kk][op], "(SELECT "+kk+" FROM "+fromSchema.Name+" WHERE "+sql+")")
					}
				}
			}
			if len(subArr) > 0 {
				return m, "(" + strings.Join(subArr, " AND ") + ")"
			}
			return m, ""
		}
	}
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
		if _, ss := t.GetFieldSQL(fieldName, utils.GetString(rule, "operator"), &fromSchema, fs, ff, rule, utils.GetInt(rule, ds.DestTableDBField)); ss != "" {
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
	return sql, nil
}

func (t *FilterService) GetFieldVerify(key string, operator string, fromSchema *sm.SchemaModel, fromField *sm.FieldModel, rule map[string]interface{}, dest int64, record map[string]interface{}, avoidVerif bool) (bool, []string, error) {
	values := []string{}
	m, _ := t.GetFieldSQL(key, operator, fromSchema, fromSchema, fromField, rule, dest)
	for k, mm := range m {
		for op, mmm := range mm {
			typ := "integer"
			if k != utils.SpecialIDParam {
				if field, err := fromSchema.GetField(k); err == nil {
					typ = field.Type
				} else {
					k = utils.SpecialIDParam
				}
			}
			for _, mmmm := range mmm {
				if len(mmmm) > 1 && fmt.Sprintf("%v", mmmm[0]) == "(" && fmt.Sprintf("%v", mmmm[len(mmmm)-1]) == ")" {

					if res, err := t.Domain.GetDb().ClearQueryFilter().QueryAssociativeArray(mmmm[1 : len(mmmm)-1]); err == nil {
						if record[k] == nil || len(res) == 0 {
							if utils.GetBool(rule, "not_null") && !avoidVerif {
								return false, []string{}, errors.New("can't validate this field assignment based on rules : should be not null <" + k + ">")
							}
							for _, r := range res {
								values = append(values, utils.GetString(r, k))
							}
						} else {
							arr := []string{}
							for _, r := range res {
								arr = append(arr, utils.GetString(r, k))
							}
							a, err := sm.CompareList(op, typ, fmt.Sprintf("%v", record[k]), arr, record)
							for _, a := range arr {
								values = append(values, fmt.Sprintf("%v", a))
							}
							if (err != nil || !a) && !avoidVerif {
								return false, values, err
							}
						}
					}
				} else {
					if record[k] == nil {
						if utils.GetBool(rule, "not_null") && !avoidVerif {
							return false, []string{}, errors.New("can't validate this field assignment based on rules : should be not null <" + k + ">")
						}
						values = append(values, mmmm)
					} else {
						if ok, err := sm.Compare(op, typ, fmt.Sprintf("%v", record[k]), mmmm, record); (err != nil || !ok) && !avoidVerif {
							return false, []string{}, errors.New("can't validate this field assignment based on rules <" + k + "> " + mmmm + " " + fmt.Sprintf("%v", record[k]))
						} else {
							values = append(values, mmmm)
						}

					}
				}
			}
		}
	}
	return true, values, nil
}

func (t *FilterService) GetOneFieldVerification(fromSchema sm.SchemaModel, record map[string]interface{}, rule map[string]interface{}, avoidVerif bool) (bool, []string, error) {
	if !utils.GetBool(rule, "verify") {
		return true, []string{}, nil
	}
	v := []string{}
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
	if ok, values, err := t.GetFieldVerify(fieldName, utils.GetString(rule, "operator"), fs, ff, rule, utils.GetInt(rule, ds.DestTableDBField), record, avoidVerif); err != nil || !ok {
		return false, values, err
	} else {
		v = append(v, values...)
	}

	return true, v, nil
}

func (t *FilterService) GetFieldVerification(fromSchema sm.SchemaModel, record map[string]interface{}) (bool, error) {
	for _, rule := range t.GetFieldCondition(fromSchema, utils.Record{}) { // SIMPLE WAY...
		if ok, _, err := t.GetOneFieldVerification(fromSchema, record, rule, false); err != nil || !ok {
			return ok, err
		}
	}
	return true, nil
}
