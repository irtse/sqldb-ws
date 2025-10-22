package schema

import (
	"errors"
	"slices"
	ds "sqldb-ws/domain/schema/database_resources"
	"sqldb-ws/domain/schema/models"
	"sqldb-ws/domain/utils"
	conn "sqldb-ws/infrastructure/connector/db"
	"strconv"
	"strings"
)

func GetTablename(supposedTableName string) string {
	i, err := strconv.Atoi(supposedTableName)
	if err != nil {
		return supposedTableName
	}
	tablename, err := GetSchemaByID(int64(i))
	if err != nil {
		return ""
	}
	return tablename.Name
}

func GetSchemaByFieldID(id int64) (models.SchemaModel, error) {
	models.CacheMutex.Lock()
	defer models.CacheMutex.Unlock()
	for _, t := range models.SchemaRegistry {
		for _, field := range t.Fields {
			if field.GetID() == id {
				return t, nil
			}
		}
	}
	return models.SchemaModel{}, errors.New("no field corresponding to reference")
}

func GetFieldByID(id int64) (models.FieldModel, error) {
	models.CacheMutex.Lock()
	defer models.CacheMutex.Unlock()
	for _, t := range models.SchemaRegistry {
		for _, field := range t.Fields {
			if field.GetID() == id {
				return field, nil
			}
		}
	}
	return models.FieldModel{}, errors.New("no field corresponding to reference")
}

func DeleteSchema(tableName string) {
	models.CacheMutex.Lock()
	defer models.CacheMutex.Unlock()
	delete(models.SchemaRegistry, tableName)
}

func DeleteSchemaField(tableName string, fieldName string) {
	models.CacheMutex.Lock()
	defer models.CacheMutex.Unlock()
	if schema, ok := models.SchemaRegistry[tableName]; ok {
		fields := []models.FieldModel{}
		for i, field := range schema.Fields {
			if field.Name != fieldName {
				fields = append(fields, schema.Fields[i])
			}
		}
		schema.Fields = fields
	}
	delete(models.SchemaRegistry, tableName)
}

func SetSchema(schema map[string]interface{}) (models.SchemaModel, error) {
	models.CacheMutex.Lock()
	defer models.CacheMutex.Unlock()

	newSchema := models.SchemaModel{}.Map(schema)
	if s, ok := models.SchemaRegistry[newSchema.Name]; ok && newSchema != nil {
		models.SchemaRegistry[newSchema.Name] = models.SchemaModel{
			ID:       s.ID,
			Name:     newSchema.Name,
			Label:    newSchema.Label,
			Category: newSchema.Category,
			Fields:   s.Fields,
		}
	} else if newSchema != nil {
		models.SchemaRegistry[newSchema.Name] = *newSchema
	}
	return models.SchemaRegistry[newSchema.Name], nil
}

func LoadCache(name string, db *conn.Database) {
	db.ClearQueryFilter()
	t := map[string]interface{}{}
	if name != utils.ReservedParam {
		t["name"] = conn.Quote(name)
	} // Filter out system tables

	schemas, err := db.SelectQueryWithRestriction(ds.DBSchema.Name, t, false) // Load schemas from base
	if err != nil || len(schemas) == 0 {
		return
	}
	for _, schema := range schemas {
		s, err := SetSchema(schema) // Add schema to cache
		if err != nil {
			continue
		}
		db.ClearQueryFilter()
		fields, err := db.SelectQueryWithRestriction(
			ds.DBSchemaField.Name, map[string]interface{}{
				ds.SchemaDBField: utils.ToString(s.ID),
			}, false) // Get fields
		db.SetSQLRestriction("") // Reset restriction
		if err == nil && len(fields) > 0 {
			for _, field := range fields {
				s = s.SetField(field) // Add field to schema
			}
		}

	}
}
func HasSchema(tableName string) bool {
	models.CacheMutex.Lock()
	if _, ok := models.SchemaRegistry[tableName]; !ok {
		models.CacheMutex.Unlock()
		return false
	} else {
		models.CacheMutex.Unlock()
		return true
	}
}

func HasField(tableName string, name string) bool {
	if schema, ok := models.SchemaRegistry[tableName]; !ok {
		return false
	} else {
		return schema.HasField(name)
	}
}

func GetSchema(tableName string) (models.SchemaModel, error) {
	models.CacheMutex.Lock()
	if schema, ok := models.SchemaRegistry[tableName]; !ok {
		models.CacheMutex.Unlock()
		return models.SchemaModel{}, errors.New("no schema corresponding to reference name")
	} else {
		models.CacheMutex.Unlock()
		return schema, nil
	}
}

func GetSchemaByID(id int64) (models.SchemaModel, error) {
	return models.GetSchemaByID(id)
}

func ValidateBySchema(data utils.Record, tableName string, method utils.Method, domain utils.DomainITF,
	check func(string, string, string, utils.Method, ...string) bool) (utils.Record, error) {
	if method == utils.DELETE || method == utils.SELECT {
		return data, nil
	}
	schema, err := GetSchema(tableName)
	if err != nil {
		return data, errors.New("no schema corresponding to reference")
	}
	order := []string{}
	if method == utils.CREATE {
		// test wf
		if fields, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBFilterField.Name, map[string]interface{}{
			ds.FilterDBField: domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBWorkflow.Name, map[string]interface{}{
				ds.SchemaDBField:            schema.ID,
				"!view_" + ds.FilterDBField: nil,
			}, false, "view_"+ds.FilterDBField),
		}, false); err == nil && len(fields) > 0 {
			for _, f := range fields {
				if field, err := schema.GetFieldByID(utils.GetInt(f, ds.SchemaFieldDBField)); err == nil {
					order = append(order, field.Name)
				}
			}
		}
	} else if method == utils.UPDATE {
		// test : wfsch
		if fields, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBFilterField.Name, map[string]interface{}{
			ds.FilterDBField: domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBWorkflowSchema.Name, map[string]interface{}{
				utils.SpecialIDParam: domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBTask.Name, map[string]interface{}{
					"is_close":          false,
					ds.DestTableDBField: data[utils.SpecialIDParam],
					ds.SchemaDBField:    schema.ID,
				}, false, ds.WorkflowSchemaDBField),
				"!view_" + ds.FilterDBField: nil,
			}, false, "view_"+ds.FilterDBField),
		}, false); err == nil && len(fields) > 0 {
			for _, f := range fields {
				if field, err := schema.GetFieldByID(utils.GetInt(f, ds.SchemaFieldDBField)); err == nil {
					order = append(order, field.Name)
				}
			}
		}
	}

	newData := utils.Record{}
	for _, field := range schema.Fields {
		if len(order) > 0 && !slices.Contains(order, field.Name) {
			continue
		}
		if field.Required && field.Default == nil && method != utils.UPDATE {
			if strings.Contains(field.Type, "many") || strings.Contains(field.Type, "upload") {
				continue
			}
			if val, ok := data[field.Name]; !((ok && val != nil && val != "") || field.Name == utils.SpecialIDParam || !check(tableName, field.Name, field.Level, utils.SELECT)) {
				if field.Label != "" {
					return data, errors.New("Missing a required field " + field.Label + " (can't see it ? you probably missing permissions)")
				} else {
					return data, errors.New("Missing a required field " + field.Name + " (can't see it ? you probably missing permissions)")
				}
			}
		}
		if field.Readonly && method == utils.UPDATE && !domain.IsSuperAdmin() {
			continue
		}
		if v, ok := data[field.Name]; ok {
			newData[field.Name] = v
			if field.Name == models.FOREIGNTABLEKEY {
				schema, err := GetSchema(utils.ToString(v))
				if err != nil {
					newData[models.LINKKEY] = schema.ID
				}
			}
		}
	}
	return newData, nil
}
