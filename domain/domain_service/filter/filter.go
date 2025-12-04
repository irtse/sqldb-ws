package filter

import (
	"net/url"
	"slices"
	sch "sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	sm "sqldb-ws/domain/schema/models"
	"sqldb-ws/domain/utils"
	connector "sqldb-ws/infrastructure/connector/db"
	"strings"
)

// DONE - ~ 260 LINES - NOT TESTED
type FilterService struct {
	Domain utils.DomainITF
}

func NewFilterService(domain utils.DomainITF) *FilterService {
	return &FilterService{Domain: domain}
}

func (f *FilterService) GetQueryFilter(tableName string, domainParams utils.Params, avoidUser bool, innerRestriction ...string) (string, string, string, string) {
	schema, err := sch.GetSchema(tableName)
	if err != nil {
		return "", "", "", ""
	}
	var SQLview, SQLrestriction, SQLOrder []string = []string{}, []string{}, []string{}
	var SQLLimit string

	restr, view, order, dir, state, hierarchy := f.GetFilterForQuery("", "", schema, domainParams)
	if restr != "" && !f.Domain.IsSuperAdmin() {
		SQLrestriction = append(SQLrestriction, restr)
	}
	later := []string{}
	for _, restr := range innerRestriction {
		if strings.Contains(restr, " IN ") {
			later = append(later, restr)
			continue
		}
		if restr != "" {
			r := []string{"(" + restr + ")"}
			r = append(r, SQLrestriction...)
			SQLrestriction = r
		}
	}
	if view != "" {
		domainParams.Add(utils.RootColumnsParam, view, func(v string) bool { return !f.Domain.IsSuperAdmin() })
	}
	if order != "" {
		domainParams.Add(utils.RootOrderParam, order, func(v string) bool { return true })
		if dir != "" {
			domainParams.Add(utils.RootDirParam, dir, func(v string) bool { return true })
		}
	}
	SQLrestriction = f.RestrictionBySchema(tableName, SQLrestriction, domainParams)
	SQLOrder = domainParams.GetOrder(func(el string) bool {
		if f, err := schema.GetField(el); err == nil {
			return !strings.Contains(strings.ToLower(f.Type), "many")
		}
		return schema.HasField(el)
	}, SQLOrder)
	SQLLimit = domainParams.GetLimit(SQLLimit)
	SQLview = f.viewbyFields(schema, domainParams)

	if f.Domain.IsShallowed() {
		if sql, err := f.GetFieldRestriction(schema); err == nil && sql != "" {
			SQLrestriction = append(SQLrestriction, sql)
		}
	}

	if s, ok := domainParams.Get(utils.RootFilterNewState); ok && s != "" {
		state = s
	}
	SQLrestriction = append(SQLrestriction, later...)
	if len(SQLview) > 0 {
		SQLview = append(SQLview, "is_draft")
	}
	if state != "" {
		SQLrestriction = f.LifeCycleRestriction(tableName, SQLrestriction, state)
	}
	if !(f.Domain.IsSuperAdmin() && !f.Domain.IsOwn(false, true, f.Domain.GetMethod())) {
		if id, _ := f.Domain.GetParams().Get(utils.SpecialIDParam); id != "" && f.Domain.GetTable() != ds.DBView.Name {
			SQLrestriction = append(SQLrestriction, "id="+id)
		} else if id, _ := domainParams.Get(utils.SpecialIDParam); id != "" {
			SQLrestriction = append(SQLrestriction, "id="+id)
		}
		if f.Domain.GetMethod() != utils.DELETE && !avoidUser && !schema.IsAssociated {
			SQLrestriction = f.RestrictionByEntityUser(schema, SQLrestriction, false, hierarchy) // admin can see all on admin view
		}
		SQLrestriction = f.GetFilterEdit(SQLrestriction, schema)
		SQLrestriction = f.GetFilterDelete(SQLrestriction, schema)
	}

	return strings.Join(SQLrestriction, " AND "), strings.Join(SQLOrder, ","), SQLLimit, strings.Join(SQLview, ",")
}

func (d *FilterService) RestrictionBySchema(tableName string, restr []string, domainParams utils.Params) []string {
	restriction := map[string]interface{}{}
	restriction["active"] = true
	if schema, err := sch.GetSchema(tableName); err == nil {
		if schema.HasField("is_meta") && !d.Domain.IsSuperAdmin() {
			restriction["is_meta"] = false
		}
		alterRestr := []string{}
		f := func(s string, search string) {
			splitted := strings.Split(s, ",")
			for _, str := range splitted {
				d.Domain.AddDetectFileToSearchIn(str, search)
			}
		}
		if line, ok := domainParams.Get(utils.RootFilterLine); ok {
			if connector.FormatSQLRestrictionWhereInjection(line, schema.GetTypeAndLinkForField, f) != "" && tableName != ds.DBView.Name {
				alterRestr = append(alterRestr, connector.FormatSQLRestrictionWhereInjection(line, schema.GetTypeAndLinkForField, f))
			}
		}
		for key, val := range domainParams.Values {
			key, val, _, typ, foreign, err := schema.GetTypeAndLinkForField(key, val, "", f)
			if err != nil && key != utils.SpecialIDParam {
				continue
			}
			newSTR := ""
			ors := strings.Split(utils.ToString(val), ",")
			for _, or := range ors {
				if len(newSTR) > 0 {
					newSTR += " OR "
				}
				_, _, _, s := connector.MakeSqlItem("", typ, foreign, key, or, "=")
				newSTR += s
			}
			if newSTR != "" {
				alterRestr = append(alterRestr, "("+newSTR+")")
			}
		}
		if d.Domain.GetMethod() != utils.DELETE {
			newRestr := []string{}
			for _, alt := range alterRestr {
				if alt != "" {
					newRestr = append(newRestr, alt)
				}
			}
			restr = append(newRestr, restr...)

			if schema.HasField(ds.SchemaDBField) && !d.Domain.IsSuperAdmin() {
				except := []string{ds.DBRequest.Name, ds.DBTask.Name, ds.DBDelegation.Name}
				enum := []string{}
				for _, s := range sm.SchemaRegistry {
					notOK := !d.Domain.IsSuperAdmin() && ds.IsRootDB(s.Name) && !slices.Contains(except, s.Name)
					notOK2 := !d.Domain.VerifyAuth(s.Name, "", sm.LEVELNORMAL, utils.SELECT)
					if !notOK && !notOK2 {
						enum = append(enum, utils.ToString(s.ID))
					}
				}
				if connector.FormatSQLRestrictionWhereByMap(
					"", map[string]interface{}{ds.SchemaDBField: enum}, false) != "" && len(enum) != 0 {
					restr = append(restr, connector.FormatSQLRestrictionWhereByMap(
						"", map[string]interface{}{ds.SchemaDBField: enum}, false))
				}

			}
		} else {
			for _, alt := range alterRestr {
				if alt != "" {
					restr = append(restr, alt)
				}
			}
		}
	}
	if strings.Trim(connector.FormatSQLRestrictionWhereByMap("", restriction, false), " ") != "" {
		restr = append(restr, strings.Trim(connector.FormatSQLRestrictionWhereByMap("", restriction, false), " "))
	}
	return restr
}

func (s *FilterService) RestrictionByEntityUser(schema sm.SchemaModel, restr []string, overrideOwn bool, hierarch bool) []string {
	if s.Domain.GetMethod() == utils.UPDATE || s.Domain.GetMethod() == utils.DELETE {
		return restr
	}
	newRestr := map[string]interface{}{}
	restrictions := map[string]interface{}{}
	if s.Domain.IsOwn(false, true, s.Domain.GetMethod()) {
		if !s.Domain.IsShallowed() && ds.DBView.Name != schema.Name {
			m := map[string]interface{}{
				utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBDataAccess.Name+" as d",
					map[string]interface{}{
						"d." + ds.SchemaDBField:    schema.ID,
						"d." + ds.DestTableDBField: "main.id",
						"d." + ds.UserDBField:      s.Domain.GetUserID(),
						"d.write":                  true,
					}, false, ds.DestTableDBField),
			}
			m[utils.SpecialIDParam+"_2"] = s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBShare.Name+" as d2", map[string]interface{}{
				"d2." + ds.SchemaDBField:      schema.ID,
				"d2." + ds.DestTableDBField:   "main.id",
				"d2.read_access":              true,
				"d2.shared_" + ds.UserDBField: s.Domain.GetUserID(),
			}, false, ds.DestTableDBField)
			if hierarch {
				m[utils.SpecialIDParam+"_1"] = s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBDataAccess.Name+" as d1",
					map[string]interface{}{
						"d1." + ds.SchemaDBField:    schema.ID,
						"d1." + ds.DestTableDBField: "main.id",
						"d1." + ds.UserDBField: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBHierarchy.Name, map[string]interface{}{
							"parent_" + ds.UserDBField: s.Domain.GetUserID(),
						}, false, ds.UserDBField),
						"d.write": true,
					}, false, ds.DestTableDBField)
			}
			restr = append(restr, "("+connector.FormatSQLRestrictionWhereByMap("", m, true)+")")
		} else {
			restr = append(restr, "is_draft=false")
		}
	}
	if slices.Contains(ds.AVOIDUSERPERMISSIONEXCEPTION, schema.Name) && utils.SELECT == s.Domain.GetMethod() {
		return restr
	}

	isUser := false
	isUser = (schema.HasField(ds.UserDBField))
	if scope, ok := s.Domain.GetParams().Get(utils.RootScope); !(ok && scope == "enable" && schema.Name == ds.DBTask.Name) && !(ok && scope == "disable" && schema.Name == ds.DBUser.Name) {
		if isUser {
			key := ds.UserDBField
			if s.Domain.GetTable() == ds.DBUser.Name {
				key = utils.SpecialIDParam
			}
			if s.Domain.GetUserID() != "" {
				if scope, ok := s.Domain.GetParams().Get(utils.RootScope); ok && scope == "enable" {
					restrictions[key] = s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBHierarchy.Name, map[string]interface{}{
						"parent_" + ds.UserDBField: s.Domain.GetUserID(),
					}, true, ds.UserDBField)
				} else {
					if hierarch {
						restrictions[key] = connector.FormatSQLRestrictionWhereByMap("", map[string]interface{}{
							ds.UserDBField: s.Domain.GetUserID(),
							ds.UserDBField + "_1": s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBHierarchy.Name, map[string]interface{}{
								"parent_" + ds.UserDBField: s.Domain.GetUserID(),
							}, false, ds.UserDBField),
						}, true)
					} else {
						restrictions[key] = s.Domain.GetUserID()
					}

				}
			}
		}
		if schema.HasField(ds.EntityDBField) {
			key := ds.EntityDBField
			if s.Domain.GetTable() == ds.DBEntity.Name {
				if !ok {
					key = utils.SpecialIDParam
				}
			}
			if s.Domain.GetUserID() != "" {
				restrictions[key] = s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(
					ds.DBEntityUser.Name,
					map[string]interface{}{
						ds.UserDBField: s.Domain.GetUserID(),
					}, true, ds.EntityDBField)
			}
		}
	}
	if len(newRestr) > 0 {
		for k, r := range newRestr {
			if r != "" {
				restrictions[k] = r
			}
		}
	}
	idParamsOk := len(s.Domain.GetParams().GetAsArgs(utils.SpecialIDParam)) > 0
	if len(restrictions) > 0 && !(idParamsOk && slices.Contains(ds.PUPERMISSIONEXCEPTION, schema.Name)) {
		var s = connector.FormatSQLRestrictionWhereByMap("", restrictions, true)
		if s != "" {
			restr = append(restr, "("+s+")")
		}

	}
	return restr
}

func (d *FilterService) viewbyFields(schema sm.SchemaModel, domainParams utils.Params) []string {
	SQLview := []string{}
	views, _ := domainParams.Get(utils.RootColumnsParam)

	for _, field := range schema.Fields {
		manyOK := strings.Contains(field.Type, "many")
		if len(views) > 0 && !strings.Contains(views, field.Name) || manyOK {
			continue
		}
		if d.Domain.VerifyAuth(d.Domain.GetTable(), field.Name, field.Level, utils.SELECT) {
			SQLview = append(SQLview, field.Name)
		}
	}
	if p, ok := domainParams.Get(utils.RootCommandRow); ok {
		decodedLine, err := url.QueryUnescape(p)
		if err == nil {
			SQLview = append(SQLview, decodedLine)
		}
	}
	if len(SQLview) > 0 {
		SQLview = append(SQLview, "id")
	}
	return SQLview
}
