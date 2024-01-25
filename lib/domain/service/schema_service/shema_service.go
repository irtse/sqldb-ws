package schema_service

import (
	"fmt"
	"errors"
	tool "sqldb-ws/lib"
	"sqldb-ws/lib/infrastructure/entities"
)

type SchemaService struct { tool.AbstractSpecializedService }

func (s *SchemaService) Entity() tool.SpecializedServiceInfo { return entities.DBSchema }
func (s *SchemaService) VerifyRowAutomation(record tool.Record, create bool) (tool.Record, bool) { return record, true }
func (s *SchemaService) DeleteRowAutomation(results tool.Results) { 
	for _, record := range results { 
		s.Domain.SetIsCustom(true)
		s.Domain.SuperCall( 
		                	tool.Params{ tool.RootTableParam : entities.DBSchemaField.Name, 
			                                tool.RootRowsParam: tool.ReservedParam,
			 	                            entities.RootID(entities.DBSchema.Name) : fmt.Sprintf("%v", record["id"]) }, 
							tool.Record{ }, 
							tool.DELETE, 
							"Delete",
						)
	}
}
func (s *SchemaService) UpdateRowAutomation(results tool.Results, record tool.Record) {}
func (s *SchemaService) WriteRowAutomation(record tool.Record) { 
	s.Domain.SuperCall(
		tool.Params{ tool.RootTableParam : record[entities.NAMEATTR].(string), }, 
		tool.Record{ entities.NAMEATTR : record[entities.NAMEATTR], 
			    "columns": map[string]interface{}{} }, 
				tool.CREATE, "CreateOrUpdate",)
}
func (s *SchemaService) PostTreatment(results tool.Results) tool.Results { 
	fmt.Printf("POSTTREATMENT \n")
	res := tool.Results{}
	for _, record := range results{
		schemas, err := Schema(s.Domain, tool.Record{entities.RootID(entities.DBSchema.Name) : record[tool.SpecialIDParam].(int64)})
		if err != nil || len(schemas) == 0 { continue }
		res = append(res, record)
	}
	return res 
}

func (s *SchemaService) ConfigureFilter(tableName string, params tool.Params) (string, string) {
	return "", ""
}	

func Schema(domain tool.DomainITF, record tool.Record) (tool.Results, error) {
	if schemaID, ok := record[entities.RootID(entities.DBSchema.Name)]; ok {
		params := tool.Params{ tool.RootTableParam : entities.DBSchema.Name, 
			tool.RootRowsParam : tool.ReservedParam, 
			tool.SpecialIDParam : fmt.Sprintf("%v", schemaID),
		}
		schemas, err := domain.SuperCall( params, tool.Record{}, tool.SELECT, "Get")
		if err != nil || len(schemas) == 0 { return nil, err }
		if _, ok := domain.GetPermission().Verify(schemas[0][entities.NAMEATTR].(string)); !ok { 
			return nil, errors.New("not authorized ") 
		}
		return schemas, nil
	}
	return nil, errors.New("no schemaID refered...")
}