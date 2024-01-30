package schema_service

import (
	"fmt"
	tool "sqldb-ws/lib"
	"sqldb-ws/lib/entities"
)

type SchemaFields struct { tool.AbstractSpecializedService }

func (s *SchemaFields) Entity() tool.SpecializedServiceInfo {return entities.DBSchemaField }
func (s *SchemaFields) VerifyRowAutomation(record tool.Record, create bool) (tool.Record, bool) {
	schemas, err := s.Domain.Schema(record)
	newRecord := tool.Record{}
	if !create {
		for k, v := range record {
			if k == "name" { newRecord["label"] = v 
			} else if k != "type" { newRecord[k] = v }
		}
	}
	return newRecord, err == nil && schemas != nil && len(schemas) > 0
}
func (s *SchemaFields) WriteRowAutomation(record tool.Record, tableName string) { 
	res, err := s.Domain.SuperCall(
		tool.Params{ tool.RootTableParam : entities.DBSchema.Name, 
			         tool.RootRowsParam: fmt.Sprintf("%v", record[entities.RootID(entities.DBSchema.Name)]) }, 
		tool.Record{}, 
		tool.SELECT, 
		"Get",
	)
	if err != nil { return }
	data := tool.Record{ 
		entities.NAMEATTR : record[entities.NAMEATTR],
		entities.TYPEATTR : record[entities.TYPEATTR],
	}
	if _, ok := record["default_value"]; ok { data["default_value"] = record["default_value"] }
	if _, ok := record["description"]; ok { data["comment"] = record["description"] }
	if len(res) > 0 {
		s.Domain.SuperCall(
			tool.Params{ tool.RootTableParam : res[0][entities.NAMEATTR].(string), 
				         tool.RootColumnsParam: tool.ReservedParam }, 
			data, 
			tool.CREATE, 
			"CreateOrUpdate")
	}
}
func (s *SchemaFields) UpdateRowAutomation(results tool.Results, record tool.Record) {
	for _, r := range results {
		res, err := s.Domain.SuperCall(
			tool.Params{ tool.RootTableParam : entities.DBSchema.Name, 
				    tool.RootRowsParam: fmt.Sprintf("%s", r[entities.RootID(entities.DBSchema.Name)]) }, 
			tool.Record{}, 
			tool.SELECT, 
			"Get",
		)
		if err != nil || res == nil || len(res) == 0 { return }
		newRecord := tool.Record{}
		for k, v := range record {
			if k == "default_value" { newRecord[k] = v 
			} else if k == "description"{ newRecord["comment"] = v }
		}
		newRecord[entities.TYPEATTR] = r[entities.TYPEATTR]
		newRecord[entities.NAMEATTR] = r[entities.NAMEATTR]
		_, err = s.Domain.SuperCall(
			tool.Params{ 
				tool.RootTableParam : res[0][entities.NAMEATTR].(string), 
				tool.RootColumnsParam: r[entities.NAMEATTR].(string) }, 
			newRecord, 
			tool.UPDATE, 
			"CreateOrUpdate",
		)
	}
}
func (s *SchemaFields) DeleteRowAutomation(results tool.Results, tableName string) { 
	for _, record := range results { 
		res, err := s.Domain.SuperCall(
			tool.Params{ tool.RootTableParam : entities.DBSchema.Name, 
				    tool.RootRowsParam: fmt.Sprintf("%v", record[entities.RootID(entities.DBSchema.Name)]) }, 
			tool.Record{}, 
			tool.SELECT, 
			"Get",
		)
		if err != nil || res == nil || len(res) == 0 { continue }
	    s.Domain.SuperCall(
			tool.Params{ tool.RootTableParam : res[0][entities.NAMEATTR].(string), 
				    tool.RootColumnsParam: record[entities.NAMEATTR].(string) }, 
			tool.Record{}, 
			tool.DELETE, 
			"Delete",
		)
	}
}
func (s *SchemaFields) PostTreatment(results tool.Results, tableName string) tool.Results { 	
	return s.Domain.PostTreat( results, tableName, false) 
}
func (s *SchemaFields) ConfigureFilter(tableName string, params tool.Params) (string, string) {
	return s.Domain.ViewDefinition(tableName, params)
}	