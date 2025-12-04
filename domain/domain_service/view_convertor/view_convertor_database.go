package view_convertor

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"sqldb-ws/domain/domain_service/history"
	sch "sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	sm "sqldb-ws/domain/schema/models"
	"sqldb-ws/domain/utils"
	"strings"
	"sync"
)

func (v *ViewConvertor) GetShortcuts(schemaID string, actions []string) map[string]string {
	shortcuts := map[string]string{}
	m := map[string]interface{}{
		"shortcut_on_schema": schemaID,
	}
	if results, err := v.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBView.Name, m, false); err == nil {
		for _, shortcut := range results {
			if utils.GetBool(shortcut, "is_empty") {
				scheme, err := sch.GetSchemaByID(utils.ToInt64(schemaID))
				if err != nil || !v.Domain.VerifyAuth(scheme.Name, "", "", utils.CREATE) {
					continue
				}
			}
			shortcuts[utils.GetString(shortcut, sm.NAMEKEY)] = "#" + utils.GetString(shortcut, utils.SpecialIDParam)
		}
	}
	return shortcuts
}

func (d *ViewConvertor) Shared(schemaID string, id string, from bool) []string {
	k := "shared_" + ds.UserDBField
	k2 := ds.UserDBField
	if from {
		k = ds.UserDBField
		k2 = "shared_" + ds.UserDBField
	}
	users := []string{}
	if res, err := d.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
		utils.SpecialIDParam: d.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBShare.Name, map[string]interface{}{
			k2:                  d.Domain.GetUserID(),
			ds.SchemaDBField:    schemaID,
			ds.DestTableDBField: id,
		}, false, k),
	}, false); err == nil {
		for _, r := range res {
			users = append(users, utils.GetString(r, "name"))
		}
	}
	return users
}

func (d *ViewConvertor) FetchRecord(tableName string, m map[string]interface{}) []map[string]interface{} {
	if len(m) == 0 {
		return []map[string]interface{}{}
	}
	t, err := d.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(tableName, m, false)
	if err != nil || len(t) == 0 {
		return []map[string]interface{}{}
	}
	return t
}

func (d *ViewConvertor) GetViewFields(tableName string, noRecursive bool, results utils.Results) (map[string]interface{}, int64, []string, map[string]sm.FieldModel, []string, bool) {
	tableName = sch.GetTablename(tableName)
	cols := make(map[string]sm.FieldModel)
	schemes := make(map[string]interface{})
	keysOrdered := []string{}
	additionalActions := []string{}
	schema, err := sch.GetSchema(tableName)
	if err != nil {
		return schemes, -1, keysOrdered, cols, additionalActions, true
	}
	l := sync.Mutex{}
	for _, scheme := range schema.Fields {
		if !d.Domain.IsSuperAdmin() && !d.Domain.VerifyAuth(tableName, scheme.Name, scheme.Level, utils.SELECT) {
			continue
		}
		l.Lock()
		if cols, ok := d.Domain.GetParams().Get(utils.RootColumnsParam); ok && cols != "" && !strings.Contains(cols, scheme.Name) {
			l.Unlock()
			continue
		}
		shallowField := sm.ViewFieldModel{
			ActionPath: "",
			Actions:    []string{},
		}
		cols[scheme.Name] = scheme
		l.Unlock()
		b, _ := json.Marshal(scheme)
		json.Unmarshal(b, &shallowField)

		if scheme.Name == utils.RootDestTableIDParam && !strings.Contains(utils.TransformType(scheme.Type), "link") {
			shallowField.Type = "link"
		} else {
			shallowField.Type = utils.TransformType(scheme.Type)
		}
		if scheme.GetLink() > 0 {
			d.ProcessLinkedSchema(&shallowField, scheme, tableName, schema)
		}
		if strings.Contains(scheme.Type, "upload") {
			shallowField.ActionPath = fmt.Sprintf("/%s/%s/import?rows=all&columns=%s", utils.MAIN_PREFIX, schema.Name, scheme.Name)
			shallowField.LinkPath = fmt.Sprintf("/%s/%s/import?rows=all&columns=%s", utils.MAIN_PREFIX, schema.Name, scheme.Name)
		}
		shallowField, additionalActions = d.ProcessPermissions(shallowField, scheme, tableName,
			additionalActions, schema, noRecursive, results)
		m := map[string]interface{}{}
		b, _ = json.Marshal(shallowField)
		err := json.Unmarshal(b, &m)
		if err == nil {
			m["autofill"], _ = d.GetFieldInfo(&scheme, ds.DBFieldAutoFill.Name)
			m["translatable"] = scheme.Translatable
			m["hidden"] = scheme.Hidden
			schemes[scheme.Name] = m
		}
		if !scheme.Hidden {
			keysOrdered = append(keysOrdered, scheme.Name)
		} else {
			ids := []string{}
			for _, r := range results {
				if utils.GetString(r, utils.SpecialIDParam) != "" {
					ids = append(ids, utils.GetString(r, utils.SpecialIDParam))
				}
			}
			if len(ids) > 0 && strings.Trim(strings.Join(ids, ""), " ") != "" {
				// exception when a task is active with workflow schema with filter and its id
				if res, err := d.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBFilterField.Name, map[string]interface{}{
					ds.SchemaFieldDBField: scheme.ID,
					ds.FilterDBField: d.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBWorkflowSchema.Name, map[string]interface{}{
						utils.SpecialIDParam: d.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
							ds.SchemaDBField:    schema.ID,
							ds.DestTableDBField: ids,
						}, false, ds.WorkflowSchemaDBField),
					}, false, "view_"+ds.FilterDBField),
				}, false); err == nil && len(res) > 0 {
					keysOrdered = append(keysOrdered, scheme.Name)
				}
			}
		}
	}
	sort.SliceStable(keysOrdered, func(i, j int) bool {
		return utils.ToInt64(utils.ToMap(schemes[keysOrdered[i]])["index"]) <= utils.ToInt64(utils.ToMap(schemes[keysOrdered[j]])["index"])
	})
	return schemes, schema.GetID(), keysOrdered, cols, additionalActions,
		!(slices.Contains(additionalActions, "post") && d.Domain.GetEmpty()) && !slices.Contains(additionalActions, "put")
}

func (d *ViewConvertor) ProcessLinkedSchema(shallowField *sm.ViewFieldModel, scheme sm.FieldModel, tableName string, s sm.SchemaModel) {
	schema, _ := sch.GetSchemaByID(scheme.GetLink())
	if !strings.Contains(shallowField.Type, "enum") && !strings.Contains(shallowField.Type, "many") && !strings.Contains(scheme.Type, "link") {
		shallowField.Type = "link"
	} else {
		shallowField.Type = utils.TransformType(scheme.Type)
	}
	shallowField.ActionPath = fmt.Sprintf("/%s/%s?rows=all&%s=enable", utils.MAIN_PREFIX, schema.Name, utils.RootShallow)
	if s.HasField(ds.DestTableDBField) || schema.HasField(ds.SchemaDBField) {
		shallowField.LinkPath = shallowField.ActionPath
	}
	if strings.Contains(scheme.Type, "many") {
		for _, field := range schema.Fields {
			if field.GetLink() != s.GetID() && field.GetLink() > 0 {
				schField, _ := sch.GetSchemaByID(field.GetLink())
				shallowField.LinkPath = fmt.Sprintf("/%s/%s?rows=all&%s=enable", utils.MAIN_PREFIX, schField.Name, utils.RootShallow)
				break
			}
		}
	}
}

func (d *ViewConvertor) ProcessPermissions(
	shallowField sm.ViewFieldModel,
	scheme sm.FieldModel,
	tableName string,
	additionalActions []string,
	schema sm.SchemaModel,
	noRecursive bool,
	record utils.Results) (sm.ViewFieldModel, []string) {
	for _, meth := range []utils.Method{utils.SELECT, utils.CREATE, utils.UPDATE, utils.DELETE} {
		if utils.DELETE == meth && len(record) == 1 {
			createdIds := history.GetCreatedAccessData(schema.ID, d.Domain)
			if !IsReadonly(schema.Name, record[0], createdIds, d.Domain) && !slices.Contains(additionalActions, meth.Method()) && slices.Contains(createdIds, utils.GetString(record[0], utils.SpecialIDParam)) {
				additionalActions = append(additionalActions, meth.Method())
			}
		} else if d.Domain.VerifyAuth(tableName, "", "", meth) && (((meth == utils.SELECT || meth == utils.CREATE) && d.Domain.GetEmpty()) || !d.Domain.GetEmpty()) {
			if !slices.Contains(additionalActions, meth.Method()) {
				additionalActions = append(additionalActions, meth.Method())
			}
			if meth == utils.CREATE && !slices.Contains(additionalActions, "import") {
				additionalActions = d.CheckAndAddImportAction(additionalActions, schema)
			}
		}
		if scheme.GetLink() > 0 && !noRecursive {
			shallowField = d.HandleRecursivePermissions(shallowField, scheme, meth, record)
		}

		if (meth == utils.UPDATE || meth == utils.CREATE) && d.Domain.GetEmpty() {
			shallowField.Readonly = false
		}
	}
	return shallowField, additionalActions
}

func (d *ViewConvertor) CheckAndAddImportAction(additionalActions []string, schema sm.SchemaModel) []string {
	d.Domain.GetDb().ClearQueryFilter()
	res, err := d.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBWorkflow.Name, map[string]interface{}{ds.SchemaDBField: schema.GetID()}, false)
	if err == nil && len(res) > 0 {
		ids := []string{}
		for _, rec := range res {
			ids = append(ids, utils.ToString(rec[utils.SpecialIDParam]))
		}
		d.Domain.GetDb().ClearQueryFilter()
		res, _ = d.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBWorkflow.Name, map[string]interface{}{
			utils.SpecialIDParam: ids,
		}, false)
		if len(res) == 0 {
			additionalActions = append(additionalActions, "import")
		}
	}
	return additionalActions
}

func (d *ViewConvertor) HandleRecursivePermissions(shallowField sm.ViewFieldModel, scheme sm.FieldModel, meth utils.Method, results utils.Results) sm.ViewFieldModel {
	schema, _ := sch.GetSchemaByID(scheme.GetLink())
	if d.Domain.VerifyAuth(schema.Name, "", "", meth) {
		if strings.Contains(scheme.Type, "many") {
			if s, ok := d.SchemaSeen[schema.Name]; !ok {
				sch, _, _, _, _, _ := d.GetViewFields(schema.Name, true, results)
				d.SchemaSeen[schema.Name] = sch
				shallowField.DataSchema = sch
			} else {
				shallowField.DataSchema = s
			}
		}
		if !strings.Contains(shallowField.Type, "enum") && !strings.Contains(shallowField.Type, "many") && !strings.Contains(scheme.Type, "link") {
			shallowField.Type = "link"
		} else {
			shallowField.Type = utils.TransformType(scheme.Type)
		}
		shallowField.ActionPath = fmt.Sprintf("/%s/%s?rows=%s&%s=enable", utils.MAIN_PREFIX, schema.Name, utils.ReservedParam, utils.RootShallow)
		if !slices.Contains(shallowField.Actions, meth.Method()) {
			shallowField.Actions = append(shallowField.Actions, meth.Method())
		}
	}
	return shallowField
}
