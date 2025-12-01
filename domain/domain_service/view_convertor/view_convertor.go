package view_convertor

import (
	"fmt"
	"net/url"
	"runtime"
	"runtime/debug"
	"slices"
	"sort"
	"sqldb-ws/domain/domain_service/history"
	"sqldb-ws/domain/domain_service/triggers"
	scheme "sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	sm "sqldb-ws/domain/schema/models"
	"sqldb-ws/domain/utils"
	connector "sqldb-ws/infrastructure/connector/db"
	"strings"
)

type ViewConvertor struct {
	Domain     utils.DomainITF
	SchemaSeen map[string]map[string]interface{}
}

func NewViewConvertor(domain utils.DomainITF) *ViewConvertor {
	return &ViewConvertor{Domain: domain, SchemaSeen: map[string]map[string]interface{}{}}
}

func (v *ViewConvertor) TransformToView(results utils.Results, tableName string, isWorkflow bool, params utils.Params) utils.Results {
	schema, err := scheme.GetSchema(tableName)
	if err != nil {
		return utils.Results{}
	}
	if ids, ok := params.Get(utils.SpecialIDParam); ok || v.Domain.GetMethod() != utils.SELECT {
		if len(ids) == 0 {
			for _, r := range results {
				ids += r.GetString(utils.SpecialIDParam) + ","
			}
			ids = connector.RemoveLastChar(ids)
		}
		history.NewDataAccess(schema.GetID(), strings.Split(ids, ","), v.Domain)

	}
	if v.Domain.IsShallowed() {
		return v.transformShallowedView(results, tableName, isWorkflow)
	}
	return v.transformFullView(results, &schema, isWorkflow, params)
}

func (v *ViewConvertor) transformFullView(results utils.Results, schema *sm.SchemaModel, isWorkflow bool, params utils.Params) utils.Results {
	schemes, id, order, _, addAction, _ := v.GetViewFields(schema.Name, false, results)
	commentBody := map[string]interface{}{}
	if len(results) == 1 {
		commentBody = map[string]interface{}{
			ds.UserDBField:      utils.ToInt64(v.Domain.GetUserID()),
			ds.SchemaDBField:    utils.ToInt64(schema.ID),
			ds.DestTableDBField: utils.GetInt(results[0], utils.SpecialIDParam),
		}
		if schema.Name == ds.DBTask.Name || schema.Name == ds.DBRequest.Name {
			commentBody[ds.SchemaDBField] = utils.GetInt(results[0], ds.SchemaDBField)
			commentBody[ds.DestTableDBField] = utils.GetInt(results[0], ds.DestTableDBField)
		}
	}
	readOnly := true
	view := sm.NewView(id, schema.Name, schema.Label, schema, schema.Name, 0, []sm.ManualTriggerModel{})
	view.Redirection = getRedirection(v.Domain.GetDomainID())

	sort.SliceStable(view.Order, func(i, j int) bool {
		return utils.ToInt64(utils.ToMap(schemes[view.Order[i]])["index"]) <= utils.ToInt64(utils.ToMap(schemes[view.Order[j]])["index"])
	})
	view.Actions = addAction
	view.CommentBody = commentBody
	view.Shortcuts = v.GetShortcuts(schema.ID, addAction)
	view.Consents = v.getConsent(schema.ID, results)
	v.ProcessResultsConcurrently(results, schema, isWorkflow, &view, params)
	// if there is only one item in the view, we can set the view readonly to the item readonly
	if len(view.Items) == 1 {
		view.Readonly = view.Items[0].Readonly
	}
	view.Order, view.Schema, readOnly = CompareOrder(schema, order, schemes, results, view.Readonly, v.Domain)
	view.SchemaNew = GetNewSchema(view.SchemaID, view.Schema, v.Domain)
	if !readOnly && !slices.Contains(view.Actions, "put") {
		view.Actions = append(view.Actions, "put")
	}
	idParamsOk := len(v.Domain.GetParams().GetAsArgs(utils.SpecialSubIDParam)) > 0
	if idParamsOk && slices.Contains(ds.PUPERMISSIONEXCEPTION, schema.Name) {
		view.Readonly = true
		for _, sch := range schemes {
			utils.ToMap(sch)["active"] = true
		}
	}
	if view.Readonly { // if the view is readonly, we remove the actions
		view.Actions = []string{"get"}
	} else {
		for _, record := range results {
			view.Triggers = append(view.Triggers, triggers.NewTrigger(v.Domain).GetViewTriggers(
				record.Copy(), v.Domain.GetMethod(), schema,
				utils.GetInt(record, ds.SchemaDBField),
				utils.GetInt(record, ds.DestTableDBField))...,
			)
		}
	}
	sort.SliceStable(view.Items, func(i, j int) bool { return view.Items[i].Sort < view.Items[j].Sort })
	if len(results) == 1 {
		view.Rules = v.GetFieldsRules(schema.Name, results[0])
	}
	return utils.Results{view.ToRecord()}
}

func (v *ViewConvertor) TransformMultipleSchema(results utils.Results, schema *sm.SchemaModel, isWorkflow bool, params utils.Params) utils.Results {
	view := sm.ViewModel{
		Items: []sm.ViewItemModel{},
	}
	v.ProcessResultsConcurrently(results, schema, isWorkflow, &view, params)
	// if there is only one item in the view, we can set the view readonly to the item readonly
	sort.SliceStable(view.Items, func(i, j int) bool { return view.Items[i].Sort < view.Items[j].Sort })
	return utils.Results{view.ToRecord()}
}

func (v *ViewConvertor) ProcessResultsConcurrently(results utils.Results, schema *sm.SchemaModel, isWorkflow bool, view *sm.ViewModel, params utils.Params) {
	const maxConcurrent = 5
	runtime.GOMAXPROCS(maxConcurrent)
	channel := make(chan sm.ViewItemModel, len(results))
	defer close(channel)
	go func() {
		if err := recover(); err != nil {
			fmt.Printf("panic occurred: %v\n%v\n", err, string(debug.Stack()))
		}
	}()
	createdIds := history.GetCreatedAccessData(schema.ID, v.Domain)
	for index, record := range results {
		go v.ConvertRecordToView(len(results), index, view, channel, record, schema, v.Domain.GetEmpty(), isWorkflow, params, createdIds)
	}
	for range results {
		rec := <-channel
		if !rec.IsEmpty {
			rec = GetSharing(schema.ID, rec, v.Domain)
		}
		view.Items = append(view.Items, rec)
	}
}

func (v *ViewConvertor) transformShallowedView(results utils.Results, tableName string, isWorkflow bool) utils.Results {
	res := utils.Results{}
	max := int64(0)
	sch, err := scheme.GetSchema(tableName)
	if err != nil {
		return res
	}
	max, _ = history.CountMaxDataAccess(sch.Name, []string{}, v.Domain)
	t := tableName
	if _, ok := v.Domain.GetParams().Get(utils.SpecialIDParam); ok && len(results) == 1 && results[0][ds.SchemaDBField] != nil {
		if sch, err = scheme.GetSchemaByID(results[0].GetInt(ds.SchemaDBField)); err == nil {
			t = sch.Name
		}
	}
	scheme, id, order, _, addAction, _ := v.GetViewFields(t, false, utils.Results{})
	for _, record := range results {
		if _, ok := record["is_draft"]; ok && record.GetBool("is_draft") && !v.Domain.IsOwn(false, false, utils.SELECT) {
			continue
		}
		if record.GetString(sm.NAMEKEY) == "" {
			res = append(res, record)
			continue
		}
		newView := v.createShallowedViewItem(record, tableName, &sch, max, isWorkflow)
		if _, ok := record["is_draft"]; ok && record.GetBool("is_draft") && !slices.Contains(addAction, "put") && v.Domain.IsOwn(false, false, utils.SELECT) {
			addAction = append(addAction, "put")
		}
		newView.Actions = addAction
		if sch.Name != tableName {
			newView.SchemaID = id
			newView.Order, newView.Schema, newView.Readonly = CompareOrder(&sch, order, scheme, results, newView.Readonly, v.Domain)
			sort.SliceStable(newView.Order, func(i, j int) bool {
				return utils.ToInt64(utils.ToMap(newView.Schema[newView.Order[i]])["index"]) <= utils.ToInt64(utils.ToMap(newView.Schema[newView.Order[j]])["index"])
			})
			newView.Consents = v.getConsent(utils.ToString(id), []utils.Record{record})
			if !utils.GetBool(record, "is_draft") && !newView.Readonly {
				newView.Triggers = triggers.NewTrigger(v.Domain).GetViewTriggers(
					record, v.Domain.GetMethod(), &sch, utils.GetInt(record, ds.SchemaDBField), utils.GetInt(record, ds.DestTableDBField))
			}
		}
		res = append(res, newView.ToRecord())
	}
	return res
}

func (v *ViewConvertor) createShallowedViewItem(record utils.Record, tableName string, schema *sm.SchemaModel, max int64, isWorkflow bool) sm.ViewModel {
	ts := []sm.ManualTriggerModel{}
	label := record.GetString(sm.NAMEKEY)
	if record.GetString(sm.LABELKEY) != "" {
		label = record.GetString(sm.LABELKEY)
	}
	translatable := true
	if f, err := schema.GetField("label"); err == nil {
		translatable = f.Translatable
	} else if f, err := schema.GetField("name"); err == nil {
		translatable = f.Translatable
	}
	view := sm.NewView(record.GetInt(utils.SpecialIDParam), record.GetString(sm.NAMEKEY), label, schema, tableName, max, ts)
	view.Path = utils.BuildPath(schema.Name, utils.ReservedParam)
	view.Redirection = getRedirection(v.Domain.GetDomainID())
	view.Translatable = translatable
	view.Workflow = v.EnrichWithWorkFlowView(record, v.Domain.GetTable(), isWorkflow)
	return view
}

func (d *ViewConvertor) ConvertRecordToView(l int, index int, view *sm.ViewModel, channel chan sm.ViewItemModel,
	record utils.Record, schema *sm.SchemaModel, isEmpty bool, isWorkflow bool, params utils.Params,
	createdIds []string) {

	vals, shallowVals, manyPathVals := make(map[string]interface{}), make(map[string]interface{}), make(map[string]string)
	manyVals := make(map[string]utils.Results)
	var datapath, historyPath, commentPath, synthesisPath string = "", "", "", ""
	if !isEmpty {
		synthesisPath = d.getSynthesis(record, schema)
		if schema.Name == ds.DBTask.Name || schema.Name == ds.DBRequest.Name {
			historyPath = utils.BuildPath(ds.DBDataAccess.Name, utils.ReservedParam, utils.RootOrderParam+"=access_date", utils.RootDirParam+"=desc", utils.RootDestTableIDParam+"="+record.GetString(ds.DestTableDBField), ds.RootID(ds.DBSchema.Name)+"="+record.GetString(ds.SchemaDBField))
		} else {
			historyPath = utils.BuildPath(ds.DBDataAccess.Name, utils.ReservedParam, utils.RootOrderParam+"=access_date", utils.RootDirParam+"=desc", utils.RootDestTableIDParam+"="+record.GetString(utils.SpecialIDParam), ds.RootID(ds.DBSchema.Name)+"="+utils.ToString(schema.ID))
		}
		if record[ds.DestTableDBField] != nil && record[ds.SchemaDBField] != nil {
			commentPath = utils.BuildPath(ds.DBComment.Name, utils.ReservedParam, utils.RootDestTableIDParam+"="+record.GetString(ds.DestTableDBField), ds.RootID(ds.DBSchema.Name)+"="+record.GetString(ds.SchemaDBField))
		} else {
			commentPath = utils.BuildPath(ds.DBComment.Name, utils.ReservedParam, utils.RootDestTableIDParam+"="+record.GetString(utils.SpecialIDParam), ds.RootID(ds.DBSchema.Name)+"="+utils.ToString(schema.ID))
		}
		vals[utils.SpecialIDParam] = record.GetString(utils.SpecialIDParam)
	}
	for _, field := range schema.Fields {
		if d, s, ok := d.HandleDBSchemaField(record, field, shallowVals); ok && d != "" {
			datapath = d
			shallowVals = s
			continue
		} else {
			shallowVals = s
		}
		shallowVals, manyVals, manyPathVals = d.HandleLinkField(record, field, schema, isEmpty, shallowVals, manyVals, manyPathVals)
		if isEmpty {
			vals[field.Name] = nil
		} else if v, ok := record[field.Name]; ok {
			vals[field.Name] = v
		}
	}

	d.ApplyCommandRow(record, vals, params)
	newOrder, vals := GetOrder(schema, record, vals, []string{}, d.Domain)
	if len(newOrder) > 0 {
		view.Order = newOrder
	}
	vals = d.GetFieldsFill(schema, vals)
	channel <- sm.ViewItemModel{
		Values:        vals,
		DataPaths:     datapath,
		ValueShallow:  shallowVals,
		Sort:          int64(index),
		DataRef:       d.getLinkPath(record, schema), // to redirect
		CommentsPath:  commentPath,
		HistoryPath:   historyPath,
		ValueMany:     manyVals,
		ValuePathMany: manyPathVals,
		Readonly:      IsReadonly(schema.Name, record, createdIds, d.Domain),
		Workflow:      d.EnrichWithWorkFlowView(record, schema.Name, isWorkflow),
		Draft:         utils.GetBool(record, "is_draft"),
		Synthesis:     synthesisPath,
		MetaData:      d.getMetaData(l, record, schema),
		New:           history.GetNew(utils.GetString(record, utils.SpecialIDParam), schema.ID, d.Domain),
	}
}

func (s *ViewConvertor) getMetaData(l int, record utils.Record, schema *sm.SchemaModel) *sm.MetaData {
	if l > 1 {
		return nil
	}
	creationUser := ""
	updateUser := ""
	updateDate := ""
	creationDate := ""

	destID := record[utils.SpecialIDParam]
	schemaID := schema.ID

	if schema.Name == ds.DBTask.Name || schema.Name == ds.DBRequest.Name {
		destID = utils.GetString(record, ds.DestTableDBField)
		schemaID = utils.GetString(record, ds.SchemaDBField)
	}

	if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBDataAccess.Name, map[string]interface{}{
		"access_date": s.Domain.GetDb().BuildSelectQueryWithRestriction(ds.DBDataAccess.Name, map[string]interface{}{
			ds.DestTableDBField: destID,
			ds.SchemaDBField:    schemaID,
			"update":            true,
		}, false, "MAX(access_date)"),
		ds.DestTableDBField: destID,
		ds.SchemaDBField:    schemaID,
		"update":            true,
	}, false); err == nil && len(res) > 0 {
		updateDate = utils.GetString(res[0], "access_date")
		if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
			utils.SpecialIDParam: res[0][ds.UserDBField],
		}, false); err == nil && len(res) > 0 {
			updateUser = utils.GetString(res[0], "name")
		}
	}
	if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBDataAccess.Name, map[string]interface{}{
		"access_date": s.Domain.GetDb().BuildSelectQueryWithRestriction(ds.DBDataAccess.Name, map[string]interface{}{
			ds.DestTableDBField: destID,
			ds.SchemaDBField:    schemaID,
			"write":             true,
		}, false, "MAX(access_date)"),
		ds.DestTableDBField: destID,
		ds.SchemaDBField:    schemaID,
		"write":             true,
	}, false); err == nil && len(res) > 0 {
		creationDate = utils.GetString(res[0], "access_date")
		if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
			utils.SpecialIDParam: res[0][ds.UserDBField],
		}, false); err == nil && len(res) > 0 {
			creationUser = utils.GetString(res[0], "name")
		}
	}
	return &sm.MetaData{
		UpdatedUser: updateUser,
		CreatedUser: creationUser,
		UpdatedDate: updateDate,
		CreatedDate: creationDate,
	}
}

func (s *ViewConvertor) getLinkPath(record utils.Record, sch *sm.SchemaModel) string {
	if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
		"is_close": false,
		utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
			ds.RequestDBField: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
				ds.DestTableDBField: utils.GetString(record, utils.SpecialIDParam),
				ds.SchemaDBField:    sch.GetID(),
			}, false, utils.SpecialIDParam),
			utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
				ds.DestTableDBField: utils.GetString(record, utils.SpecialIDParam),
				ds.SchemaDBField:    sch.GetID(),
			}, false, utils.SpecialIDParam),
		}, true, utils.SpecialIDParam),
		utils.SpecialIDParam + "_1": s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
			ds.UserDBField: s.Domain.GetUserID(),
			ds.EntityDBField: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBEntityUser.Name, map[string]interface{}{
				ds.UserDBField: s.Domain.GetUserID(),
			}, false, ds.EntityDBField),
		}, true, utils.SpecialIDParam),
	}, false); err == nil && len(res) > 0 {
		firstTaskToWrap := res[0]
		if s, err := scheme.GetSchema(ds.DBTask.Name); err == nil {
			return "@" + s.ID + ":" + utils.GetString(firstTaskToWrap, utils.SpecialIDParam)
		}
	} else if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
		"is_close":          true,
		ds.DestTableDBField: utils.GetString(record, utils.SpecialIDParam),
		ds.SchemaDBField:    sch.GetID(),
		ds.UserDBField:      s.Domain.GetUserID(),
	}, false); err == nil && len(res) > 0 {
		firstTaskToWrap := res[0]
		if s, err := scheme.GetSchema(ds.DBRequest.Name); err == nil {
			return "@" + s.ID + ":" + utils.GetString(firstTaskToWrap, utils.SpecialIDParam)
		}
	}
	return "@" + sch.ID + ":" + utils.GetString(record, utils.SpecialIDParam)
}

func (s *ViewConvertor) getConsent(schemaID string, results utils.Results) []map[string]interface{} {
	if !s.Domain.GetEmpty() && len(results) != 1 {
		return []map[string]interface{}{}
	}
	if ds.DBRequest.Name == s.Domain.GetTable() && utils.GetBool(results[0], "is_close") {
		return []map[string]interface{}{}
	} else if len(results) > 0 {
		if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
			"is_close":          true,
			ds.SchemaDBField:    schemaID,
			ds.DestTableDBField: results[0][utils.SpecialIDParam],
		}, false); err == nil && len(res) > 0 {
			return []map[string]interface{}{}
		}
	}

	key := "on_create"
	if s.Domain.GetMethod() == utils.UPDATE || (s.Domain.GetMethod() == utils.SELECT && !s.Domain.GetEmpty() && !utils.GetBool(results[0], "is_draft")) {
		key = "on_update"
	}
	if consents, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBConsent.Name, map[string]interface{}{
		ds.SchemaDBField: schemaID,
		utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBConsent.Name, map[string]interface{}{
			key: true,
		}, true, utils.SpecialIDParam),
	}, false); err == nil {
		if len(results) > 0 {
			for _, c := range consents {
				if consentsResp, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(
					ds.DBConsentResponse.Name,
					map[string]interface{}{
						ds.SchemaDBField:    schemaID,
						ds.DestTableDBField: results[0][utils.SpecialIDParam],
						ds.ConsentDBField:   utils.GetString(c, utils.SpecialIDParam),
					}, false); err == nil && len(consentsResp) > 0 {
					return []map[string]interface{}{}
				}
			}
		}
		cst := []map[string]interface{}{}
		for _, r := range consents {
			c := map[string]interface{}{}
			c["name"] = utils.GetString(r, "name")
			c["optionnal"] = utils.GetBool(r, "optionnal")
			c["body"] = map[string]interface{}{
				ds.SchemaDBField:  r[ds.SchemaDBField],
				ds.ConsentDBField: r[utils.SpecialIDParam],
			}
			c["action_path"] = fmt.Sprintf("/%s/%s?%s=%s", utils.MAIN_PREFIX, ds.DBConsentResponse.Name, utils.RootRowsParam, utils.ReservedParam)
			cst = append(cst, c)
		}
		return cst
	}
	return []map[string]interface{}{}
}

func (s *ViewConvertor) getSynthesis(record utils.Record, schema *sm.SchemaModel) string {
	taskIDs := ""
	if schema.Name == ds.DBTask.Name {
		if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
			utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
				ds.UserDBField: s.Domain.GetUserID(),
				ds.EntityDBField: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBEntityUser.Name, map[string]interface{}{
					ds.UserDBField: s.Domain.GetUserID(),
				}, false, ds.EntityDBField),
			}, true, utils.SpecialIDParam),
			ds.RequestDBField: record[ds.RequestDBField],
		}, false); err == nil {
			is := []string{}
			for _, r := range res {
				is = append(is, utils.GetString(r, utils.SpecialIDParam))
			}
			if !slices.Contains(is, utils.GetString(record, utils.SpecialIDParam)) {
				is = append(is, utils.GetString(record, utils.SpecialIDParam))
			}
			if len(is) > 0 {
				taskIDs = strings.Join(is, ",")
			}

		}
	} else if schema.Name == ds.DBRequest.Name {
		if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
			ds.RequestDBField: record[utils.SpecialIDParam],
		}, false); err == nil && len(res) > 0 {
			is := []string{}
			for _, r := range res {
				is = append(is, utils.GetString(r, utils.SpecialIDParam))
			}
			if len(is) > 0 {
				taskIDs = strings.Join(is, ",")
			}
		}
	} else {
		if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
			utils.SpecialIDParam: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
				ds.DestTableDBField: record[utils.SpecialIDParam],
				ds.SchemaDBField:    schema.ID,
			}, false, utils.SpecialIDParam),
			utils.SpecialIDParam + "_1": s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
				ds.RequestDBField: s.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
					ds.DestTableDBField: record[utils.SpecialIDParam],
					ds.SchemaDBField:    schema.ID,
				}, false, utils.SpecialIDParam),
			}, false, utils.SpecialIDParam),
		}, true); err == nil && len(res) > 0 {
			is := []string{}
			for _, r := range res {
				is = append(is, utils.GetString(r, utils.SpecialIDParam))
			}
			if len(is) > 0 {
				taskIDs = strings.Join(is, ",")
			}
		}
	}
	if taskIDs != "" { // means there is actually running task effective on these data
		return fmt.Sprintf("/%s/%s?%s=%s&scope=enable&%s=%s",
			utils.MAIN_PREFIX, ds.DBTask.Name,
			utils.RootRowsParam, taskIDs,
			utils.RootColumnsParam, "name,state,opening_date,closing_date,dbuser_id,dbentity_id",
		)
	}
	return ""
}

func (d *ViewConvertor) HandleDBSchemaField(record utils.Record, field sm.FieldModel, shallowVals map[string]interface{}) (string, map[string]interface{}, bool) {
	datapath := ""
	id, idOk := record[field.Name]
	dest, destOk := record[ds.DestTableDBField]
	if !strings.Contains(field.Name, ds.DBSchema.Name) || !idOk || id == nil {
		return datapath, shallowVals, false
	}
	schema, err := scheme.GetSchemaByID(utils.ToInt64(id))
	if err != nil {
		return datapath, shallowVals, false
	}
	shallowVals[ds.SchemaDBField] = utils.Record{"id": utils.ToString(schema.ID), "name": utils.ToString(schema.Name), "label": utils.ToString(schema.Label)}
	if destOk && dest != nil {
		datapath = utils.BuildPath(schema.Name, utils.ToString(dest))
		if t, err := d.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(schema.Name, map[string]interface{}{
			utils.SpecialIDParam: dest,
		}, false); err == nil && len(t) > 0 {
			shallowVals[ds.DestTableDBField] = utils.Record{
				utils.SpecialIDParam: utils.ToString(t[0][utils.SpecialIDParam]),
				sm.NAMEKEY:           utils.ToString(t[0][sm.NAMEKEY]),
				sm.LABELKEY:          utils.ToString(t[0][sm.NAMEKEY]),
				"data_ref":           "@" + utils.ToString(schema.ID) + ":" + utils.ToString(t[0][utils.SpecialIDParam]),
				"values_path":        utils.BuildPath(utils.ToString(schema.ID), utils.ToString(t[0][utils.SpecialIDParam]), utils.RootShallow+"=enable"),
			}
		}
	}
	return datapath, shallowVals, true
}

func (d *ViewConvertor) HandleLinkField(record utils.Record, field sm.FieldModel, schema *sm.SchemaModel, shallow bool,
	shallowVals map[string]interface{}, manyVals map[string]utils.Results, manyPathVals map[string]string) (map[string]interface{}, map[string]utils.Results, map[string]string) {
	if (record.GetString(field.Name) == "" && !strings.Contains(field.Type, "many")) || field.GetLink() <= 0 || shallow {
		return shallowVals, manyVals, manyPathVals
	}
	link := scheme.GetTablename(utils.ToString(field.Link))

	if strings.Contains(field.Type, "many") {
		manyVals, manyPathVals = d.HandleManyField(record, field, schema, link, manyVals, manyPathVals)
		return shallowVals, manyVals, manyPathVals
	}
	shallowVals = d.HandleOneField(record, field, link, shallowVals)
	return shallowVals, manyVals, manyPathVals
}

func (d *ViewConvertor) recursiveFoundNameOneToMany(bfTable sm.SchemaModel, field sm.FieldModel, manyVals map[string]utils.Results, subTable sm.SchemaModel, subField sm.FieldModel, sudId string) map[string]utils.Results {
	fmt.Println("SUBFIELD BEFORE", subField.Name, bfTable.Name, subTable.Name, subTable.HasField("name"))

	if subField.GetLink() != bfTable.GetID() {
		return manyVals
	}
	if subTable.HasField("name") {
		fmt.Println("SUBFIELD", subField.Name, subTable.Name)
		if !subTable.HasField(subField.Name) {
			return manyVals
		}
		if res, err := d.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(subTable.Name, map[string]interface{}{
			subField.Name: sudId,
		}, false); err == nil {
			if _, ok := manyVals[field.Name]; !ok {
				manyVals[field.Name] = utils.Results{}
			}
			for _, r := range res {
				manyVals[field.Name] = append(manyVals[field.Name], utils.Record{"name": utils.GetString(r, "name")})
			}
		}
	} else {
		for _, f := range subTable.Fields {
			if !subTable.HasField(subField.Name) {
				continue
			}
			if sch, err := scheme.GetSchemaByID(f.GetLink()); err == nil && !strings.Contains(strings.ToLower(f.Type), strings.ToLower(sm.ONETOMANY.String())) {
				if res, err := d.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(subTable.Name, map[string]interface{}{
					subField.Name: sudId,
				}, false); err == nil {
					for _, ff := range sch.Fields {
						if ff.GetLink() == subTable.GetID() {
							subField = ff
						}
					}
					for _, r := range res {
						manyVals = d.recursiveFoundNameOneToMany(subTable, field, manyVals, sch, subField, utils.GetString(r, utils.SpecialIDParam))
					}
				}
			}

		}
	}
	return manyVals
}

func (d *ViewConvertor) HandleManyField(record utils.Record, field sm.FieldModel, schema *sm.SchemaModel, link string,
	manyVals map[string]utils.Results, manyPathVals map[string]string) (map[string]utils.Results, map[string]string) {
	if !d.Domain.IsShallowed() {
		l, _ := scheme.GetSchemaByID(field.GetLink())
		for _, f := range l.Fields {
			if strings.Contains(strings.ToUpper(field.Type), strings.ToUpper(sm.ONETOMANY.String())) {
				if f.GetLink() == schema.GetID() {
					manyPathVals[field.Name] = utils.BuildPath(
						link, utils.ReservedParam,
						f.Name+"="+record.GetString(utils.SpecialIDParam))
				}
				manyVals = d.recursiveFoundNameOneToMany(*schema, field, manyVals, l, f, utils.GetString(record, utils.SpecialIDParam))
				continue
			}
			if strings.Contains(f.Name, schema.Name) || f.Name == utils.SpecialIDParam || f.GetLink() <= 0 {
				continue
			}
			lid, _ := scheme.GetSchemaByID(f.GetLink())
			if _, ok := manyVals[field.Name]; !ok {
				manyVals[field.Name] = utils.Results{}
			}
			// field link is a many to many... such as authors
			// link is related tableName : demo_authors
			// f is the field from some_authors that not correspond to the schema.Name _ id : exemple demo_id -> demo
			// lid is the link of this field for exemple : user & rootID(lid.Name) == user_id

			// on veut former une requête comme suit : SELECT * FROM dbuser WHERE id IN (SELECT dbuser_id FROM demo_authors WHERE dbdemo_id = ?)
			// HERE IS REGULARY MALFORMED REQUEST FOR AUTHORS
			// SELECT * FROM article_authors WHERE id IN (SELECT id FROM article_affiliation_authors WHERE dbarticle_id=197 AND dbarticle_authors_id IS NOT NULL) pq: column "dbarticle_authors_id" does not exist
			if res, err := d.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(lid.Name, map[string]interface{}{
				utils.SpecialIDParam: d.Domain.GetDb().BuildSelectQueryWithRestriction(link, map[string]interface{}{
					"!" + ds.RootID(lid.Name): nil,
					ds.RootID(schema.Name):    record.GetString(utils.SpecialIDParam),
				}, false, ds.RootID(lid.Name))}, false); err == nil {
				for _, r := range res {
					manyVals[field.Name] = append(manyVals[field.Name], r)
				}
			}

			if linkTable, err := scheme.GetSchema(link); err != nil || !linkTable.HasField("name") {
				continue
			}
			if res, err := d.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(link, map[string]interface{}{
				"!name":                nil,
				ds.RootID(lid.Name):    nil, // should be nil
				ds.RootID(schema.Name): record.GetString(utils.SpecialIDParam),
			}, false); err == nil {
				for _, r := range res {
					manyVals[field.Name] = append(manyVals[field.Name], utils.Record{"name": utils.GetString(r, "name")})
				}
			}
		}
	}
	return manyVals, manyPathVals
}

func (d *ViewConvertor) HandleOneField(record utils.Record, field sm.FieldModel, link string, shallowVals map[string]interface{}) map[string]interface{} {
	v := record.GetString(field.Name)
	if r, err := d.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(link, map[string]interface{}{
		utils.SpecialIDParam: v,
	}, false); err == nil && len(r) > 0 {
		ref := fmt.Sprintf("@%v:%v", field.Link, r[0][utils.SpecialIDParam])
		shallowVals[field.Name] = utils.Record{
			utils.SpecialIDParam: r[0][utils.SpecialIDParam],
			sm.NAMEKEY:           r[0][sm.NAMEKEY],
			"data_ref":           ref,
		}
		if _, ok := r[0][sm.LABELKEY]; ok {
			shallowVals[field.Name].(utils.Record)[sm.LABELKEY] = r[0][sm.LABELKEY]
		}
	}
	return shallowVals
}

func (d *ViewConvertor) ApplyCommandRow(record utils.Record, vals map[string]interface{}, params utils.Params) {
	if cmd, ok := params.Get(utils.RootCommandRow); ok {
		decodedLine, _ := url.QueryUnescape(cmd)
		matches := strings.Split(decodedLine, " as ")
		if len(matches) > 1 {
			vals[matches[len(matches)-1]] = record[matches[len(matches)-1]]
		}
	}
}

func IsReadonly(tableName string, record utils.Record, createdIds []string, d utils.DomainITF) bool {
	if d.GetEmpty() || utils.GetBool(record, "is_draft") {
		return false
	}
	// TODO when no field readable
	if sch, err := scheme.GetSchema(tableName); err == nil {
		if tableName == ds.DBTask.Name {
			if !utils.GetBool(record, "is_close") && (utils.GetString(record, ds.UserDBField) == d.GetUserID()) || slices.Contains(createdIds, record.GetString(utils.SpecialIDParam)) {
				return false // if its my task and currently working allow it
			}
		} else { // if no task then follow this
			subMapTrue := map[string]interface{}{
				utils.SpecialIDParam: d.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
					ds.DestTableDBField: record[utils.SpecialIDParam],
					ds.SchemaDBField:    sch.ID,
				}, false, utils.SpecialIDParam),
			}
			subMap := map[string]interface{}{
				utils.SpecialIDParam: d.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
					ds.DestTableDBField: record[utils.SpecialIDParam],
					ds.SchemaDBField:    sch.ID,
					utils.SpecialIDParam: d.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
						"is_close":     false,
						ds.UserDBField: d.GetUserID(),
						ds.EntityDBField: d.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBEntityUser.Name, map[string]interface{}{
							ds.UserDBField: d.GetUserID(),
						}, false, ds.EntityDBField),
					}, false, ds.RequestDBField),
				}, false, utils.SpecialIDParam),
			}
			if record[ds.DestTableDBField] != nil {
				subMapTrue[utils.SpecialIDParam+"_1"] = d.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
					ds.DestTableDBField: record[ds.DestTableDBField],
					ds.SchemaDBField:    record[ds.SchemaDBField],
				}, false, utils.SpecialIDParam)
				subMap[utils.SpecialIDParam+"_1"] = d.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{
					ds.DestTableDBField: record[ds.DestTableDBField],
					ds.SchemaDBField:    record[ds.SchemaDBField],
					utils.SpecialIDParam: d.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
						"is_close":     false,
						ds.UserDBField: d.GetUserID(),
						ds.EntityDBField: d.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBEntityUser.Name, map[string]interface{}{
							ds.UserDBField: d.GetUserID(),
						}, false, ds.EntityDBField),
					}, false, ds.RequestDBField),
				}, false, utils.SpecialIDParam)
			}
			// MISSING SHARED
			if res, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{ // then if there is request in run it should not be readonly
				"is_close":           false,
				utils.SpecialIDParam: d.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, subMap, true, utils.SpecialIDParam),
			}, false); err == nil && len(res) > 0 {
				return false
			} else if rr, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBRequest.Name, map[string]interface{}{ //then no request are active, if there some closed protecting data, then readonly
				"is_close":           true,
				utils.SpecialIDParam: d.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBRequest.Name, subMapTrue, true, utils.SpecialIDParam),
			}, false); err != nil || len(rr) > 0 {
				return true // if a request about this data is end up, only one
			} else { // in case of no request at all !
				if res, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBShare.Name, map[string]interface{}{
					"shared_" + ds.UserDBField: d.GetUserID(),
					ds.DestTableDBField:        record[utils.SpecialIDParam],
					ds.SchemaDBField:           sch.ID,
					"update_access":            true,
				}, false); err == nil && len(res) > 0 {
					return false
				}
				for k, _ := range d.GetParams().Values {
					if sch.HasField(k) { // a method to override per params
						return false
					}
				}
				if slices.Contains(createdIds, record.GetString(utils.SpecialIDParam)) { // if created it's our own we can update it
					return false
				}
			}
		}
	}
	// if nothing occurs before... then... check if there is permission allowing you to update
	for _, meth := range []utils.Method{utils.CREATE, utils.UPDATE} {
		if (meth == utils.CREATE && d.GetEmpty()) || meth == utils.UPDATE {
			if d.VerifyAuth(tableName, "", "", meth, record.GetString(utils.SpecialIDParam)) {
				return false // if allowed then not readonly, like in a superadmin non requestable data
			}
		}
	}
	return true
}
