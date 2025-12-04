package schema_service

import (
	"errors"
	"fmt"
	"slices"
	sch "sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	sm "sqldb-ws/domain/schema/models"
	servutils "sqldb-ws/domain/specialized_service/utils"
	"sqldb-ws/domain/utils"
	"strings"
)

var MissingField = map[string][]utils.Record{}

// DONE - UNDER 100 LINES - NOT TESTED
type SchemaFields struct{ servutils.SpecializedService }

func NewSchemaFieldsService() utils.SpecializedServiceITF {
	return &SchemaFields{}
}

func (s *SchemaFields) Entity() utils.SpecializedServiceInfo { return ds.DBSchemaField }

func (s *SchemaFields) VerifyDataIntegrity(record map[string]interface{}, tablename string) (map[string]interface{}, error, bool) {
	if s.Domain.GetMethod() == utils.DELETE { // delete root schema field
		if s.Domain.IsSuperAdmin() {
			return record, nil, false
		}
		return record, fmt.Errorf("cannot delete root schema field"), false
	}
	if s, err := sch.GetSchemaByID(utils.GetInt(record, ds.SchemaDBField)); err == nil {
		if _, err := s.GetFieldByID(utils.GetInt(record, utils.SpecialIDParam)); err == nil {
			return record, errors.New("already exists"), false
		}
	}
	utils.Add(record, sm.TYPEKEY, record[sm.TYPEKEY],
		func(i interface{}) bool { return i != nil && i != "" },
		func(i interface{}) interface{} { return utils.PrepareEnum(utils.ToString(i)) })
	if !strings.Contains(sm.DataTypeToEnum(), utils.ToString(record[sm.TYPEKEY])) {
		return record, fmt.Errorf("invalid type"), false
	}
	utils.Add(record, sm.LABELKEY, record[sm.LABELKEY],
		func(i interface{}) bool { return true },
		func(i interface{}) interface{} {
			if i == nil || i == "" {
				i = utils.ToString(record[sm.NAMEKEY])
			}
			return strings.Replace(utils.ToString(i), "_", " ", -1)
		})
	if strings.Contains(utils.GetString(record, "type"), "many") && utils.GetString(record, "link_id") == "" {
		if utils.CREATE == s.Domain.GetMethod() {
			if MissingField[utils.GetString(record, "foreign_table")] == nil {
				MissingField[utils.GetString(record, "foreign_table")] = []utils.Record{}
			}
			MissingField[utils.GetString(record, "foreign_table")] = append(MissingField[utils.GetString(record, "foreign_table")], record)
			return nil, errors.New("later implementation"), false
		}
	}
	delete(record, "foreign_table")
	if !slices.Contains(ds.NOAUTOLOADROOTTABLESSTR, tablename) {
		if rec, err := sch.ValidateBySchema(record, tablename, s.Domain.GetMethod(), s.Domain, s.Domain.VerifyAuth); err != nil {

			return s.SpecializedService.VerifyDataIntegrity(rec, tablename)
		}
	}
	return s.SpecializedService.VerifyDataIntegrity(record, tablename)
}

func (s *SchemaFields) SpecializedCreateRow(record map[string]interface{}, tableName string) { // THERE
	s.Write(record, record, false)
	s.AbstractSpecializedService.SpecializedCreateRow(record, tableName)
}

func (s *SchemaFields) SpecializedUpdateRow(results []map[string]interface{}, record map[string]interface{}) {
	for _, r := range results {
		s.Write(r, record, true)
	}
	s.AbstractSpecializedService.SpecializedUpdateRow(results, record)
}

func (s *SchemaFields) Write(r map[string]interface{}, record map[string]interface{}, isUpdate bool) (*sm.SchemaModel, error) {
	schema, err := sch.GetSchemaByID(utils.ToInt64(r[ds.SchemaDBField]))
	if err != nil {
		return nil, err
	}
	if typ, ok := record[sm.TYPEKEY]; !ok || strings.Contains(utils.ToString(typ), "many") || schema.HasField(utils.ToString(record[sm.NAMEKEY])) {
		return nil, fmt.Errorf("field already exists")
	} else if utils.ToString(typ) == "url" || strings.Contains(utils.ToString(typ), "upload") {
		record[sm.TYPEKEY] = "varchar"
	} else if utils.ToString(typ) == "html" {
		record[sm.TYPEKEY] = "text"
	} else if utils.ToString(typ) == "link_add" {
		record[sm.TYPEKEY] = "integer"
	}
	readLevels := []string{sm.LEVELNORMAL}
	if level, ok := record["read_level"]; ok && level != "" && level != sm.LEVELOWN && slices.Contains(sm.READLEVELACCESS, utils.ToString(level)) {
		readLevels = append(readLevels, strings.Replace(utils.ToString(level), "'", "", -1))
	}
	UpdatePermissions(record, schema.Name, readLevels, s.Domain)
	if !slices.Contains(ds.NOAUTOLOADROOTTABLESSTR, utils.ToString(record[sm.NAMEKEY])) {
		if isUpdate {
			newRecord := utils.ToRecord(record, map[string]interface{}{
				sm.TYPEKEY: r[sm.TYPEKEY],
				sm.NAMEKEY: r[sm.NAMEKEY],
			})
			s.Domain.UpdateSuperCall(utils.GetColumnTargetParameters(schema.Name, r[sm.NAMEKEY]).RootRaw(), newRecord, false)
		} else {
			s.Domain.CreateSuperCall(utils.GetColumnTargetParameters(schema.Name, r[sm.NAMEKEY]).RootRaw(), record, false)
		}
	}
	schema = schema.SetField(r)
	return &schema, nil
}

func (s *SchemaFields) SpecializedDeleteRow(results []map[string]interface{}, tableName string) {
	for _, record := range results { // delete all columns
		schema, err := sch.GetSchemaByID(utils.ToInt64(record[ds.SchemaDBField]))
		if err != nil { // schema not found
			s.Domain.DeleteSuperCall(utils.GetColumnTargetParameters(schema.Name, record[sm.NAMEKEY]).RootRaw(), false)
			s.Domain.DeleteSuperCall(
				utils.AllParams(ds.DBPermission.Name).Enrich(map[string]interface{}{
					sm.NAMEKEY: "%" + schema.Name + ":" + utils.ToString(record[sm.NAMEKEY]) + "%",
				}).RootRaw(), false,
			)
			sch.DeleteSchemaField(schema.Name, utils.ToString(record[sm.NAMEKEY]))
		}
	}
	s.AbstractSpecializedService.SpecializedDeleteRow(results, tableName)
}
