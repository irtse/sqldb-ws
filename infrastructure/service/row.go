package service

import (
	"errors"
	"fmt"
	conn "sqldb-ws/infrastructure/connector/db"
	"strings"
)

type TableRowService struct {
	Table    *TableService
	EmptyCol *TableColumnService
	InfraService
}

func NewTableRowService(database conn.DB, admin bool, user string, name string, specializedService InfraSpecializedServiceItf) *TableRowService {
	row := &TableRowService{
		Table: NewTableService(database, admin, user, name),
		EmptyCol: &TableColumnService{InfraService: InfraService{
			DB:                 database,
			SpecializedService: &InfraSpecializedService{},
		}},
		InfraService: InfraService{
			DB:                 database,
			NoLog:              false,
			SpecializedService: specializedService,
		},
	}
	row.Fill(name, admin, user)
	if row.SpecializedService == nil {
		row.SpecializedService = &InfraSpecializedService{}
	}
	return row
}

func (t *TableRowService) Template(restriction ...string) (interface{}, error) {
	return t.Get(restriction...)
}

func (t *TableRowService) Verify(name string) (string, bool) {
	res, err := t.Get("id=" + name)
	return name, err == nil && len(res) > 0
}

func (t *TableRowService) Math(algo string, restriction ...string) ([]map[string]interface{}, error) {
	if _, err := t.setupFilter(map[string]interface{}{}, false, false, restriction...); err != nil {
		return nil, err
	}
	res, err := t.DB.MathQuery(algo, t.Table.Name)
	if err != nil || len(res) == 0 {
		return nil, err
	}
	t.Results = append(t.Results, map[string]interface{}{"result": res[0]["result"]})
	return t.Results, nil
}

func (t *TableRowService) Get(restriction ...string) ([]map[string]interface{}, error) {
	var err error
	if _, err = t.setupFilter(map[string]interface{}{}, false, false, restriction...); err != nil {
		return nil, err
	}
	if t.Results, err = t.DB.SelectQueryWithRestriction(
		t.Table.Name, map[string]interface{}{}, false); err != nil {
		return t.DBError(nil, err)
	}
	return t.Results, nil
}

func (t *TableRowService) Create(record map[string]interface{}) ([]map[string]interface{}, error) {
	if len(record) == 0 {
		return nil, errors.New("no data to insert")
	}
	t.DB.ClearQueryFilter()
	var id int64
	var err error
	if r, err, forceChange := t.SpecializedService.VerifyDataIntegrity(record, t.Name); err != nil {
		return nil, err
	} else if forceChange {
		if record["id"] == nil && r["id"] != nil {
			return t.Update(record, "id="+fmt.Sprintf("%v", r["id"]))
		}
		record = r
	} else if record["id"] == nil && r["id"] != nil {
		return t.Update(record, "id="+fmt.Sprintf("%v", r["id"]))
	}
	t.EmptyCol.Name = t.Name
	verify := t.EmptyCol.Verify
	if id, err = t.DB.CreateQuery(t.Name, record, verify); err != nil {
		return t.DBError(nil, err)
	}
	r, err := t.DB.ClearQueryFilter().SelectQueryWithRestriction(t.Table.Name, map[string]interface{}{
		"id": id,
	}, false)
	if len(r) > 0 {
		t.Results = r
		t.SpecializedService.SpecializedCreateRow(t.Results[0], t.Table.Name)
	}
	return t.Results, err
}

func (t *TableRowService) Update(record map[string]interface{}, restriction ...string) ([]map[string]interface{}, error) {
	var err error
	if strings.Contains(t.Name, "request") {
		fmt.Println("REQ", record)
	}
	if record, err = t.setupFilter(record, true, true, restriction...); err != nil {
		return nil, err
	}
	if strings.Contains(t.Name, "request") {
		fmt.Println("REQ1", record)
	}
	if strings.Contains(t.DB.GetSQLRestriction(), "id=null") {
		t.DB.ClearQueryFilter()
		return t.Create(record)
	}
	if strings.Contains(t.Name, "request") {
		fmt.Println("REQ2", record)
	}
	t.EmptyCol.Name = t.Name
	if query, err := t.DB.BuildUpdateRowQuery(t.Table.Name, record, t.EmptyCol.Verify); err == nil {
		if strings.Contains(t.Name, "request") {
			fmt.Println("REQ3", query)
		}
		if err := t.DB.Query(query); err != nil {
			return t.DBError(nil, err)
		}
	} else {
		return t.DBError(nil, err)
	}
	r, err := t.DB.SelectQueryWithRestriction(t.Table.Name, map[string]interface{}{}, false)
	if err != nil {
		return t.DBError(nil, err)
	} else {
		t.Results = r
	}
	t.SpecializedService.SpecializedUpdateRow(t.Results, record)
	return t.Results, nil
}
func (t *TableRowService) Delete(restriction ...string) ([]map[string]interface{}, error) {
	var err error
	if _, err = t.setupFilter(map[string]interface{}{}, true, true, restriction...); err != nil {
		return nil, err
	}
	if t.Results, err = t.Get(restriction...); err == nil {
		if t.DB.GetSQLRestriction() == "" {
			return t.DBError(nil, errors.New("no restriction can't delete all"))
		} else if err = t.DB.DeleteQuery(t.Table.Name, ""); err != nil {
			return t.DBError(nil, err)
		}
		t.SpecializedService.SpecializedDeleteRow(t.Results, t.Table.Name)
	}
	return t.Results, err
}

func (t *TableRowService) setupFilter(record map[string]interface{}, verify bool, write bool, restriction ...string) (map[string]interface{}, error) {
	if verify {
		if r, err, forceChange := t.SpecializedService.VerifyDataIntegrity(record, t.Name); err != nil {
			return record, err
		} else if forceChange {
			record = r
		}
	}
	restr, order, limit, view := t.SpecializedService.GenerateQueryFilter(t.Table.Name, restriction...)
	if write {
		t.DB.ClearQueryFilter().ApplyQueryFilters(restr, "", "", view)
	} else {
		t.DB.ClearQueryFilter().ApplyQueryFilters(restr, order, limit, view)
	}
	return record, nil
}
