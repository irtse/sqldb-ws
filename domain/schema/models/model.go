package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"sqldb-ws/domain/utils"
	db "sqldb-ws/infrastructure/connector/db"
	"strconv"
	"strings"
)

var SchemaRegistry = map[string]SchemaModel{}

type SchemaModel struct { // lightest definition a db table
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Label          string       `json:"label"`
	IsEnum         bool         `json:"is_enum"`
	Category       string       `json:"category"`
	CanOwned       bool         `json:"can_owned"`
	Description    string       `json:"description"` // Special case for ownership, it's schema that can be owned by any user (like a request)
	Fields         []FieldModel `json:"fields,omitempty"`
	IsAssociated   bool         `json:"is_associated"`
	ViewIDOnDelete string       `json:"redirect_view_id_on_delete"`
}

func (t SchemaModel) Map(m map[string]interface{}) *SchemaModel {
	return &SchemaModel{
		ID:             utils.ToString(m["id"]),
		Name:           utils.ToString(m["name"]),
		Label:          utils.ToString(m["label"]),
		Category:       utils.ToString(m["category"]),
		IsAssociated:   utils.Compare(m["is_associated"], true),
		CanOwned:       utils.Compare(m["can_owned"], true),
		ViewIDOnDelete: utils.ToString(m["redirect_view_id_on_delete"]),
	}
}

func (t SchemaModel) GetID() int64 {
	i, err := strconv.Atoi(t.ID)
	if err != nil {
		return -1
	}
	return int64(i)
}

func (t SchemaModel) Deserialize(rec utils.Record) SchemaModel {
	b, _ := json.Marshal(rec)
	json.Unmarshal(b, &t)
	return t
}
func (t SchemaModel) GetName() string { return t.Name }

func (t SchemaModel) SetField(field map[string]interface{}) SchemaModel {
	newField := FieldModel{}.Map(field)
	if !t.HasField(newField.Name) {
		CacheMutex.Lock()
		defer CacheMutex.Unlock()
		t.Fields = append(t.Fields, *newField)
	} else {
		CacheMutex.Lock()
		defer CacheMutex.Unlock()
		for _, f := range t.Fields {
			if newField.Name != f.Name {
				f = *newField
			}
		}
	}
	SchemaRegistry[t.Name] = t
	return t
}

func (t SchemaModel) HasField(name string) bool {
	CacheMutex.Lock()
	defer CacheMutex.Unlock()
	for _, field := range t.Fields {
		if field.Name == name {
			return true
		}
	}
	return false
}

func GetSchemaByID(id int64) (SchemaModel, error) {
	CacheMutex.Lock()
	for _, schema := range SchemaRegistry {
		if schema.GetID() == id {
			CacheMutex.Unlock()
			return schema, nil
		}
	}
	CacheMutex.Unlock()
	return SchemaModel{}, errors.New("no schema corresponding to reference id")
}

func (t SchemaModel) GetTypeAndLinkForField(name string, search string, operator string, onUpload func(string, string)) (string, string, string, string, string, error) {
	field, err := t.GetField(strings.Split(name, ".")[0])
	if err != nil {
		return name, search, operator, "", "", err
	}
	if strings.Contains(field.Type, "upload") {
		if strings.Contains(field.Type, "upload_str") {
			onUpload(field.Name, search)
		}
		return name, search, operator, field.Type, "", errors.New("can't proceed a publication")
	}
	foreign, err := GetSchemaByID(field.GetLink())
	if err != nil {
		return name, search, operator, field.Type, "", nil
	}
	if strings.Contains(strings.ToUpper(field.Type), strings.ToUpper(MANYTOMANY.String())) {
		if sch, err := GetSchemaByID(field.GetLink()); err == nil {
			for _, f := range sch.Fields {
				if f.GetLink() > 0 && t.GetID() != f.GetLink() {
					_, _, _, sql := db.MakeSqlItem("", f.Type, "", f.Name, search, operator)
					return "id", "(SELECT db" + t.Name + "_id FROM " + sch.Name + " WHERE " + sql + " )", "IN", "manytomany", "", err
				}
			}
		}
		return name, search, operator, field.Type, foreign.Name, errors.New("can't filter many to many on this " + name + " field with value " + search)
	} else if strings.Contains(strings.ToUpper(field.Type), strings.ToUpper(ONETOMANY.String())) {
		if strings.Contains(name, ".") {
			subKey := strings.Join(strings.Split(name, ".")[1:], ".")
			if sch, err := GetSchemaByID(field.GetLink()); err == nil {
				var key = ""
				for _, f := range sch.Fields {
					if f.GetLink() > 0 && t.GetID() == f.GetLink() {
						key = f.Name
					}
				}
				if key != "" {
					if subKey, search, operator, typ, _, err := sch.GetTypeAndLinkForField(subKey, search, operator, onUpload); err == nil {
						_, _, _, sql := db.MakeSqlItem("", typ, "", strings.Split(subKey, ".")[0], search, operator)
						return "id", "(SELECT  " + key + " FROM " + sch.Name + " WHERE " + sql + ")", "IN", "onetomany", "", err
					}
				}
			}
		}
		return name, search, operator, field.Type, foreign.Name, errors.New("can't filter one to many on this " + name + " field with value " + search)
	}
	return name, search, operator, field.Type, foreign.Name, nil
}

func (t SchemaModel) GetField(name string) (FieldModel, error) {
	for _, field := range t.Fields {
		if field.Name == name {
			return field, nil
		}
	}
	return FieldModel{}, errors.New("no field corresponding to reference")
}
func (t SchemaModel) GetFieldByID(id int64) (FieldModel, error) {
	for _, field := range t.Fields {
		if field.GetID() == id {
			return field, nil
		}
	}
	return FieldModel{}, errors.New("no field corresponding to reference")
}

func (v SchemaModel) ToRecord() utils.Record {
	var r utils.Record
	b, _ := json.Marshal(v)
	json.Unmarshal(b, &r)
	return r
}

func (v SchemaModel) ToMapRecord() utils.Record {
	fields := map[string]FieldModel{}
	for _, field := range v.Fields {
		fields[field.Name] = field
	}
	var r utils.Record
	b, _ := json.Marshal(fields)
	json.Unmarshal(b, &r)
	return r
}

func (v SchemaModel) ToSchemaRecord() utils.Record {
	fields := []FieldModel{}
	for _, field := range v.Fields {
		if !strings.Contains(field.Type, "many") {
			fields = append(fields, field)
		}
	}
	var r utils.Record
	b, _ := json.Marshal(SchemaModel{
		ID:           v.ID,
		Name:         v.Name,
		Label:        v.Label,
		Category:     v.Category,
		IsAssociated: v.IsAssociated,
		Fields:       fields,
	})
	json.Unmarshal(b, &r)
	return r
}

type FieldModel struct { // definition a db table columns
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Label        string      `json:"label"`
	Desc         string      `json:"description"`
	Type         string      `json:"type"`
	Index        int64       `json:"index"`
	Placeholder  string      `json:"placeholder"`
	Default      interface{} `json:"default_value"`
	Level        string      `json:"read_level,omitempty"`
	Readonly     bool        `json:"readonly"`
	Dir          string      `json:"dir"`
	Link         string      `json:"link_id"`
	Subsection   string      `json:"subsection"`
	ForeignTable string      `json:"-"` // Special case for foreign key
	InResume     string      `json:"in_resume,omitempty"`
	Constraint   string      `json:"constraints"` // Special case for constraint on field
	Required     bool        `json:"required"`
	Hidden       bool        `json:"hidden"`
	Translatable bool        `json:"translatable,omitempty"`
	Transform    string      `json:"transform_function"`
	GroupBy      string      `json:"group_by"`
	SchemaID     string      `json:"dbschema_id"`
}

func (t FieldModel) Map(m map[string]interface{}) *FieldModel {
	return &FieldModel{
		ID:           utils.ToString(m["id"]),
		Name:         utils.ToString(m["name"]),
		Label:        utils.ToString(m["label"]),
		Desc:         utils.ToString(m["description"]),
		Type:         utils.ToString(m["type"]),
		Index:        utils.ToInt64(m["index"]),
		Placeholder:  utils.ToString(m["placeholder"]),
		Subsection:   utils.ToString(m["subsection"]),
		Default:      m["default_value"],
		InResume:     utils.ToString(m["in_resume"]),
		Level:        utils.ToString(m["read_level"]),
		Readonly:     utils.Compare(m["readonly"], true),
		Link:         utils.ToString(m["link_id"]),
		Constraint:   utils.ToString(m["constraints"]),
		Required:     utils.Compare(m["required"], true),
		Translatable: utils.Compare(m["translatable"], true),
		Hidden:       utils.Compare(m["hidden"], true),
		Transform:    utils.ToString(m["transform"]),
		SchemaID:     utils.ToString(m["dbschema_id"]),
	}
}

func (t FieldModel) GetID() int64 {
	i, err := strconv.Atoi(t.ID)
	if err != nil {
		return -1
	}
	return int64(i)
}

func (t FieldModel) GetLink() int64 {
	i, err := strconv.Atoi(t.Link)
	if err != nil {
		return -1
	}
	return int64(i)
}

func (v FieldModel) ToRecord() utils.Record {
	var r utils.Record
	b, _ := json.Marshal(v)
	json.Unmarshal(b, &r)
	return r
}

type ManualTriggerModel struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Type        string       `json:"type"`
	Schema      utils.Record `json:"schema"`
	Body        utils.Record `json:"body"`
	ActionPath  string       `json:"action_path"`
}

type ViewModel struct { // lightest struct based on SchemaModel dedicate to view
	ID            int64                    `json:"id"`
	Name          string                   `json:"name"`
	Label         string                   `json:"label"`
	SchemaID      int64                    `json:"schema_id"`
	SchemaName    string                   `json:"schema_name"`
	Description   string                   `json:"description"`
	Path          string                   `json:"link_path"`
	Order         []string                 `json:"order"`
	Schema        utils.Record             `json:"schema"`
	Items         []ViewItemModel          `json:"items"`
	Actions       []string                 `json:"actions"`
	ActionPath    string                   `json:"action_path"`
	ExportPath    string                   `json:"export_path"`
	Readonly      bool                     `json:"readonly"`
	Workflow      *WorkflowModel           `json:"workflow"`
	IsWrapper     bool                     `json:"is_wrapper"`
	Shortcuts     map[string]string        `json:"shortcuts"`
	Consents      []map[string]interface{} `json:"consents"`
	CommentBody   map[string]interface{}   `json:"comment_body"`
	Translatable  bool                     `json:"translatable,omitempty"`
	Redirection   string                   `json:"redirection,omitempty"`
	Triggers      []ManualTriggerModel     `json:"triggers,omitempty"`
	Max           int64                    `json:"max"`
	Rules         []map[string]interface{} `json:"rules"`
	MultiViewPath []string                 `json:"multi_view_path"`
}

func NewView(id int64, name string, label string, schema *SchemaModel, tableName string, max int64, triggers []ManualTriggerModel) ViewModel {
	return ViewModel{
		ID:            id,
		Name:          name,
		SchemaName:    schema.Name,
		SchemaID:      schema.GetID(),
		Description:   fmt.Sprintf("%s data", schema.Name),
		ActionPath:    utils.BuildPath(schema.Name, utils.ReservedParam),
		ExportPath:    utils.BuildPath(schema.Name, utils.ReservedParam),
		Path:          utils.BuildPath(schema.Name, utils.ReservedParam),
		IsWrapper:     tableName == "dbtask" || tableName == "dbrequest",
		Label:         label,
		Items:         []ViewItemModel{},
		Triggers:      triggers,
		Max:           max,
		MultiViewPath: []string{utils.BuildPath(schema.ID, utils.ReservedParam)},
	}
}

func (v ViewModel) ToRecord() utils.Record {
	var r utils.Record
	b, _ := json.Marshal(v)
	json.Unmarshal(b, &r)
	return r
}

type MetaData struct {
	CreatedUser string `json:"created_user"`
	UpdatedUser string `json:"updated_user"`
	CreatedDate string `json:"created_date"`
	UpdatedDate string `json:"updated_date"`
}

type ViewItemModel struct {
	IsEmpty       bool                     `json:"-"`
	Sort          int64                    `json:"-"`
	DataRef       string                   `json:"data_ref,omitempty"`
	Path          string                   `json:"link_path"`
	DataPaths     string                   `json:"data_path"`
	ValuePathMany map[string]string        `json:"values_path_many"`
	Values        map[string]interface{}   `json:"values"`
	ValueShallow  map[string]interface{}   `json:"values_shallow"`
	ValueMany     map[string]utils.Results `json:"values_many"`
	HistoryPath   string                   `json:"history_path"`
	CommentsPath  string                   `json:"comments_path"`
	Workflow      *WorkflowModel           `json:"workflow"`
	Readonly      bool                     `json:"readonly"`
	Sharing       SharingModel             `json:"sharing"`
	Draft         bool                     `json:"is_draft"`
	Synthesis     string                   `json:"synthesis_path"`
	New           bool                     `json:"new"`
	MetaData      *MetaData                `json:"metadata"`
}

type SharingModel struct {
	Body            map[string]interface{} `json:"body"`
	SharedWithPath  string                 `json:"shared_with_path"`
	Path            string                 `json:"share_path"`
	ShallowPath     map[string]string      `json:"shallow_path"`
	AdditionnalDate []string               `json:"additionnal_date"`
	AdditionnalBool []string               `json:"additionnal_bool"`
}

type ViewFieldModel struct { // lightest struct based on FieldModel dedicate to view
	Label        string                 `json:"label" validate:"required"`
	Type         string                 `json:"type" validate:"required"`
	Index        int64                  `json:"index"`
	Description  string                 `json:"description"`
	Placeholder  string                 `json:"placeholder"`
	Default      interface{}            `json:"default_value"`
	Required     bool                   `json:"required"`
	Readonly     bool                   `json:"readonly"`
	InResume     string                 `json:"in_resume,omitempty"`
	LinkPath     string                 `json:"values_path"`
	ActionPath   string                 `json:"action_path"`
	Actions      []string               `json:"actions"`
	DataSchemaID int64                  `json:"data_schema_id"`
	DataSchema   map[string]interface{} `json:"data_schema"`
	Subsection   string                 `json:"subsection"`
	Active       bool                   `json:"active"`
}

type WorkflowModel struct { // lightest struct based on SchemaModel dedicate to view
	ID             string                         `json:"id"`
	IsDismiss      bool                           `json:"is_dismiss"`
	IsDismissable  bool                           `json:"is_dismissable"`
	Current        string                         `json:"current"`
	Position       string                         `json:"position"`
	IsClose        bool                           `json:"is_close"`
	CurrentHub     bool                           `json:"current_hub"`
	CurrentDismiss bool                           `json:"current_dismiss"`
	CurrentClose   bool                           `json:"current_close"`
	Steps          map[string][]WorkflowStepModel `json:"steps"`
}

type WorkflowStepModel struct { // lightest struct based on SchemaModel dedicate to view
	ID        int64          `json:"id"`
	Name      string         `json:"name"`
	Optionnal bool           `json:"optionnal"`
	IsSet     bool           `json:"is_set"`
	IsDismiss bool           `json:"is_dismiss"`
	IsCurrent bool           `json:"is_current"`
	IsClose   bool           `json:"is_close"`
	Workflow  *WorkflowModel `json:"workflow"`
}

type FilterModel struct {
	ID        int64       `json:"id"`
	Name      string      `json:"name"`
	Label     string      `json:"label"`
	Type      string      `json:"type"`
	Index     float64     `json:"index"`
	Value     interface{} `json:"value"`
	Operator  string      `json:"operator"`
	Separator string      `json:"separator"`
	Dir       string      `json:"dir"`
	Width     float64     `json:"width"`
}
