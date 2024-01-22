package service

import (
	"os"
	"fmt"
	"strings"
	"errors"
	"encoding/json"
	tool "sqldb-ws/lib"
	"github.com/rs/zerolog/log"
	_ "github.com/go-sql-driver/mysql"
	"sqldb-ws/lib/infrastructure/entities"
	conn "sqldb-ws/lib/infrastructure/connector"
)

type TableRowInfo struct {
	SpecializedService  tool.SpecializedService `json:"-"`
	Table				*TableInfo
	EmptyCol            *TableColumnInfo
	Verified  	        bool
	InfraService
}

func (t *TableRowInfo) Template() (interface{}, error) { return t.Get() }

func (t *TableRowInfo) Verify(name string) (string, bool) {
	t.db.SQLRestriction = "id=" + name
	res, err := t.Get()
	return name, err == nil && len(res) > 0
}

func (t *TableRowInfo) Get() (tool.Results, error) {
	t.db = ToFilter(t.Table.Name, t.Params, t.db)
	d, err := t.db.SelectResults(t.Table.Name)
	t.Results = d
	if err != nil { return DBError(nil, err) }
	return t.Results, nil
}

func (t *TableRowInfo) Create() (tool.Results, error) {
	var id int64
	var err error
	var result tool.Results
	columns := ""
	values := ""
	if _, ok := t.SpecializedService.VerifyRowWorkflow(t.Record, true); !ok { return nil, errors.New("verification failed.") }
	v := Validator[map[string]interface{}]()
	_, err = v.ValidateSchema(t.Record, t.Table, false)
	if err != nil { return nil, errors.New("Not a proper struct to create a row " + err.Error()) }
	for key, element := range t.Record {
		columns += key + ","
		typ := ""
		if t.Verified { 
			typ, _ = t.EmptyCol.Verify(key) 
			values += conn.FormatForSQL(typ, element) + ","
		} else {
			values += fmt.Sprintf("%v", element) + ","
		}
	}
	query := "INSERT INTO " + t.Table.Name + "(" + conn.RemoveLastChar(columns) + ") VALUES (" + conn.RemoveLastChar(values) + ")"
	if t.db.Driver == conn.PostgresDriver { 
		id, err = t.db.QueryRow(query)
		if err != nil { return DBError(nil, err) }
		t.db.SQLRestriction = fmt.Sprintf("id=%d", id)
		if err != nil { return DBError(nil, err) }
	}
	if t.db.Driver == conn.MySQLDriver {
		stmt, err := t.db.Prepare(query)
		if err != nil { return DBError(nil, err) }
		res, err := stmt.Exec()
		if err != nil { return DBError(nil, err) }
		id, err = res.LastInsertId()
		t.db.SQLRestriction = fmt.Sprintf("id=%d", id)
		if err != nil { return DBError(nil, err) }
		if err != nil { return DBError(nil, err) }
	}
	result, err = t.db.SelectResults(t.Table.Name)
	t.Results = result
	t.SpecializedService.WriteRowWorkflow(t.Record)
	return t.Results, nil
}

func (t *TableRowInfo) Update() (tool.Results, error) {
	v := Validator[map[string]interface{}]()
	_, err := v.ValidateSchema(t.Record, t.Table, true)
	if err != nil { return nil, errors.New("Not a proper struct to update a row") }
	r, _ := t.SpecializedService.VerifyRowWorkflow(t.Record, false) 
	t.Record = r
	t.db = ToFilter(t.Table.Name, t.Params, t.db)
	stack := ""
	filter := ""
	for key, element := range t.Record {
		if key != "id" { 
			if len(t.PermService.WarningUpdateField) > 0 {
				found := false
				for _, w := range t.PermService.WarningUpdateField { 
					if w == key { found = true; break }
				}
				if found {
					t.Params[key]="NULL"
					resp, _ := t.db.SelectResults(t.Table.Name)
					if len(resp) == 0 { continue }
				}					
			} 
			if t.Verified {
				typ, ok := t.EmptyCol.Verify(key)
				if ok { 
					stack = stack + " " + key + "=" + conn.FormatForSQL(typ, element) + "," 
					filter += key + "=" + conn.FormatForSQL(typ, element) + " and " 
				}
			} else { 
				stack = stack + " " + key + "=" + fmt.Sprintf("%v", element) + "," 
				filter += key + "=" + fmt.Sprintf("%v", element) + " and " 
			}
		} else if !strings.Contains(t.db.SQLRestriction, "id=") { t.db.SQLRestriction += "id=" + fmt.Sprintf("%d", int64(element.(float64))) + " " }
	}
	stack = conn.RemoveLastChar(stack)
	query := ("UPDATE " + t.Table.Name + " SET " + stack) // REMEMBER id is a restriction !
	if t.db.SQLRestriction != "" { query += " WHERE " + t.db.SQLRestriction }
	rows, err := t.db.Query(query)
	if err != nil { return DBError(nil, err) }
	defer rows.Close()
	if len(t.db.SQLRestriction) > 0 { 
		if (len(filter) > 0) {
			t.db.SQLRestriction += "and " + filter[:len(filter) - 4]
		}
    } else { if (len(filter) > 0) { t.db.SQLRestriction = filter[:len(filter) - 4] }  }
	
	res, err := t.db.SelectResults(t.Table.Name)
	if err != nil { return DBError(nil, err) }
	t.SpecializedService.UpdateRowWorkflow(res, t.Record) 
	t.Results = res
	return t.Results, nil
}

func (t *TableRowInfo) CreateOrUpdate() (tool.Results, error) {
	_, ok := t.Params[tool.SpecialIDParam]
	if ok == false && t.Method != tool.UPDATE { return t.Create() 
	} else { return t.Update() }
}

func (t *TableRowInfo) Delete() (tool.Results, error) {
	t.db = ToFilter(t.Table.Name, t.Params, t.db)
	res, err := t.db.SelectResults(t.Table.Name)
	if err != nil { return DBError(nil, err) }
	t.Results = res
	query := ("DELETE FROM " + t.Table.Name)
	if t.db.SQLRestriction != "" { query += " WHERE " + t.db.SQLRestriction }
	rows, err := t.db.Query(query)
	if err != nil { return DBError(nil, err) }
	defer rows.Close()
	t.SpecializedService.DeleteRowWorkflow(t.Results)
	return t.Results, nil
}

func (t *TableRowInfo) Add() (tool.Results, error) { 
	return nil, errors.New("not implemented...")
}

func (t *TableRowInfo) Remove() (tool.Results, error) { 
	return nil, errors.New("not implemented...")
}

func (t *TableRowInfo) Import(filename string) (tool.Results, error)  {
	var jsonSource []TableRowInfo
	byteValue, _ := os.ReadFile(filename)
	err := json.Unmarshal([]byte(byteValue), &jsonSource)
	if err != nil { return DBError(nil, err) }
	for _, row := range jsonSource {
		row.db = t.db
		if t.Method == tool.DELETE { _, err = row.Delete() 
		} else { _, err = row.Create() }
		if err != nil { log.Error().Msg(err.Error()) }
	}
	return t.Results, nil
}

func (t *TableRowInfo) Link() (tool.Results, error) {
	if _, ok := t.Params[tool.RootToTableParam]; !ok { return nil, errors.New("no destination table") }
	otherName := t.Params[tool.RootToTableParam]
	v := Validator[entities.LinkEntity]()
	v.data = entities.LinkEntity{}
	te, err := v.ValidateStruct(t.Record)
	if err != nil { return nil, errors.New("Not a proper struct to create a table - expect <LinkEntity> Scheme " + err.Error()) }
	if _, ok := t.EmptyCol.Verify(otherName + "_id"); ok && te.Anchor == "" {
		// should verify record from_id to_id
		res, err := t.link(te, otherName, false)
		if err != nil { t.Results = append(t.Results, res...) }
	} else {
		// here FIND LINK TABLE
		schemas, err := t.Table.schema(tool.ReservedParam)
		if err != nil { return nil, errors.New("problem on schema")}
		for _, scheme := range schemas {
			_, findRoot := scheme.AssColumns[t.Name + "_id"] 
			_, findOther := scheme.AssColumns[otherName  + "_id"] 
			if findRoot && findOther && strings.Contains(scheme.Name, te.Anchor) {
				t.EmptyCol.Name = scheme.Name
				t.Table.Name = t.EmptyCol.Name
				res, err := t.link(te, otherName, false)
				if err == nil { t.Results = append(t.Results, res...) }
			}
		}
	}
	return t.Results, nil
}
func (t *TableRowInfo) link(te *entities.LinkEntity, otherName string, nullable bool) (tool.Results, error)  {
	t.Record = tool.Record{ otherName + "_id" : te.To, t.Name + "_id" : te.From }
	if te.Columns != nil && !nullable {
		for col, val := range te.Columns {
			if _, ok := t.EmptyCol.Verify(col); ok && val != "" { t.Record[col]=val }
		}
	}
	if len(t.Record) == 0 { return nil, errors.New("no data to set or create")}
	if !nullable { return t.CreateOrUpdate() 
	} else { return t.Delete()  }
}

func (t *TableRowInfo) UnLink() (tool.Results, error) {
	if _, ok := t.Params[tool.RootToTableParam]; !ok { return nil, errors.New("no destination table") }
	otherName := t.Params[tool.RootToTableParam]
	v := Validator[entities.LinkEntity]()
	v.data = entities.LinkEntity{}
	te, err := v.ValidateStruct(t.Record)
	if err != nil { return nil, errors.New("Not a proper struct to create a table - expect <LinkEntity> Scheme " + err.Error()) }
	if _, ok := t.EmptyCol.Verify(otherName + "_id"); ok {
		res, err := t.link(te, otherName, true)
		if err != nil { t.Results = append(t.Results, res...) }
	} else { 
		schema, err := t.Table.schema(tool.ReservedParam)
		if err != nil { return nil, errors.New("problem on schema")}
		for _, scheme := range schema {
			_, findRoot := scheme.AssColumns[t.Name + "_id"] 
			_, findOther := scheme.AssColumns[otherName + "_id"] 
			if findRoot && findOther && strings.Contains(scheme.Name, te.Anchor) {
				t.EmptyCol.Name = scheme.Name
				t.Table.Name = t.EmptyCol.Name
				res, err := t.link(te, otherName, true)
				if err != nil { t.Results = append(t.Results, res...) }
			}
		}
	}
	return t.Results, nil
}