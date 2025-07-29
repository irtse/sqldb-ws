package view_convertor

import (
	"slices"
	ds "sqldb-ws/domain/schema/database_resources"
	sm "sqldb-ws/domain/schema/models"
	"sqldb-ws/domain/utils"
	"strings"
)

func (d *ViewConvertor) EnrichWithWorkFlowView(record utils.Record, tableName string, isWorkflow bool) *sm.WorkflowModel {
	if !isWorkflow {
		return nil
	}
	workflow := sm.WorkflowModel{Position: "0", Current: "0", Steps: make(map[string][]sm.WorkflowStepModel)}
	id, requestID, nexts := "", "", []string{}

	switch tableName {
	case ds.DBWorkflow.Name:
		id = record.GetString(utils.SpecialIDParam)
	case ds.DBRequest.Name:
		id = utils.ToString(record[ds.WorkflowDBField])
		requestID = utils.ToString(record[utils.SpecialIDParam])
		workflow = d.InitializeWorkflow(record)
	case ds.DBTask.Name:
		if workflow, id, requestID, nexts = d.handleTaskWorkflow(record); id == "" {
			return nil
		}
	default:
		return nil
	}

	if id == "" || id == "<nil>" {
		return nil
	}

	return d.populateWorkflowSteps(&workflow, id, requestID, nexts)
}

func (d *ViewConvertor) InitializeWorkflow(record map[string]interface{}) sm.WorkflowModel {
	return sm.WorkflowModel{
		IsDismiss: record["state"] == "dismiss" || record["state"] == "refused" || record["state"] == "canceled",
		Current:   utils.ToString(record["current_index"]),
		Position:  utils.ToString(record["current_index"]),
		IsClose:   utils.GetBool(record, "is_close") || record["state"] == "dismiss" || record["state"] == "refused" || record["state"] == "completed" || record["state"] == "canceled",
	}
}

func (d *ViewConvertor) handleTaskWorkflow(record utils.Record) (sm.WorkflowModel, string, string, []string) {
	var workflow sm.WorkflowModel
	reqRecord := d.FetchRecord(ds.DBRequest.Name,
		map[string]interface{}{
			utils.SpecialIDParam: utils.GetString(record, ds.RequestDBField),
		})
	if len(reqRecord) > 0 {
		workflow.IsDismissable = !utils.GetBool(reqRecord[0], "is_close")
		workflow = d.InitializeWorkflow(reqRecord[0])
	}

	nexts := d.ParseNextSteps(record)
	requestID := record.GetString(ds.RootID(ds.DBRequest.Name))
	workflow.CurrentDismiss = record["state"] == "dismiss"
	workflow.CurrentClose = record["state"] == "completed" || record["state"] == "dismiss" || record["state"] == "refused" || record["state"] == "canceled"

	schemaRecord := d.FetchRecord(ds.DBWorkflowSchema.Name, map[string]interface{}{
		utils.SpecialIDParam: record.GetInt(ds.WorkflowSchemaDBField),
	})
	if len(schemaRecord) > 0 {
		workflow.IsDismissable = !utils.GetBool(schemaRecord[0], "optionnal")
		workflow.Current = utils.GetString(schemaRecord[0], "index")
		workflow.CurrentHub = utils.Compare(schemaRecord[0]["hub"], true)
		return workflow, utils.ToString(schemaRecord[0][ds.RootID(ds.DBWorkflow.Name)]), requestID, nexts
	}
	return workflow, "", "", nil
}

func (d *ViewConvertor) ParseNextSteps(record map[string]interface{}) []string {
	if record["nexts"] == "all" || record["nexts"] == "" || record["nexts"] == nil {
		return nil
	}
	return strings.Split(utils.ToString(record["nexts"]), ",")
}

func (d *ViewConvertor) populateWorkflowSteps(workflow *sm.WorkflowModel, id string, requestID string, nexts []string) *sm.WorkflowModel {
	// get Hierarchical

	steps := d.FetchRecord(ds.DBWorkflowSchema.Name, map[string]interface{}{
		ds.WorkflowDBField: id,
	})
	if len(steps) == 0 {
		return workflow
	}

	workflow.Steps = make(map[string][]sm.WorkflowStepModel)
	if hierarchical, err := d.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBHierarchy.Name, map[string]interface{}{
		ds.UserDBField: d.Domain.GetUserID(),
		ds.EntityDBField: d.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(
			ds.DBEntityUser.Name,
			map[string]interface{}{
				ds.UserDBField: d.Domain.GetUserID(),
			}, true, ds.EntityDBField),
	}, false); err == nil && len(hierarchical) > 0 {
		workflow.Steps["0"] = []sm.WorkflowStepModel{}
		/*for _, hierarch := range hierarchical {
			h := sm.WorkflowStepModel{
				Name:      "Hierarchical Validation",
				Optionnal: false,
				IsSet:     true,
			}
			workflow.Steps["0"] = append(workflow.Steps["0"], )
		}*/
	}

	for _, step := range steps {
		index := utils.ToString(step["index"])
		newStep := sm.WorkflowStepModel{
			ID:        utils.GetInt(step, utils.SpecialIDParam),
			Name:      utils.ToString(step[sm.NAMEKEY]),
			Optionnal: utils.Compare(step["optionnal"], true),
			IsSet:     !utils.Compare(step["optionnal"], true) || slices.Contains(nexts, utils.ToString(step["wrapped_"+ds.RootID(ds.DBWorkflow.Name)])),
		}

		if workflow.Current != "" {
			d.populateTaskDetails(&newStep, step, requestID)
		}

		if wrapped, ok := step["wrapped_"+ds.RootID(ds.DBWorkflow.Name)]; ok {
			newStep.Workflow = d.EnrichWithWorkFlowView(utils.Record{utils.SpecialIDParam: wrapped}, ds.DBWorkflow.Name, true)
		}

		workflow.Steps[index] = append(workflow.Steps[index], newStep)
	}
	workflow.ID = id
	return workflow
}

func (d *ViewConvertor) populateTaskDetails(newStep *sm.WorkflowStepModel, step map[string]interface{}, requestID string) {
	tasks := d.FetchRecord(ds.DBTask.Name, map[string]interface{}{
		utils.SpecialIDParam: d.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
			ds.UserDBField: d.Domain.GetUserID(),
			ds.EntityDBField: d.Domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBEntityUser.Name,
				map[string]interface{}{
					ds.UserDBField: d.Domain.GetUserID(),
				}, false, ds.EntityDBField),
		}, false, utils.SpecialIDParam),
		ds.RequestDBField:        requestID,
		ds.WorkflowSchemaDBField: utils.GetInt(step, utils.SpecialIDParam),
	})
	if len(tasks) > 0 {
		newStep.IsClose = utils.Compare(tasks[0]["is_close"], true)
		newStep.IsCurrent = utils.Compare(tasks[0]["state"], "pending")
		newStep.IsDismiss = utils.Compare(tasks[0]["state"], "dismiss") || utils.Compare(tasks[0]["state"], "refused")
	}
}
