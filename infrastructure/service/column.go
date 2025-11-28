package service

import (
	"errors"
	"fmt"
	connector "sqldb-ws/infrastructure/connector/db"
	"strings"
)

// Table is a table structure description
type TableColumnService struct {
	Views string
	InfraService
}

func NewTableColumnService(database connector.DB, admin bool, user string, name string, specializedService InfraSpecializedServiceItf, views string) *TableColumnService {
	col := &TableColumnService{
		Views:        views,
		InfraService: InfraService{DB: database, NoLog: false, SpecializedService: specializedService},
	}
	col.Fill(name, admin, user)
	if col.SpecializedService == nil {
		col.SpecializedService = &InfraSpecializedService{}
	}
	return col
}

func (t *TableColumnService) Template(restriction ...string) (interface{}, error) {
	return t.Get(restriction...)
}

func (t *TableColumnService) Math(algo string, restriction ...string) ([]map[string]interface{}, error) {
	restr, order, limit, _ := t.SpecializedService.GenerateQueryFilter(t.Name, strings.Join(restriction, " AND "))
	t.DB.ApplyQueryFilters(restr, order, limit, t.Views)
	res, err := t.DB.MathQuery(algo, t.Name)
	if err != nil || len(res) == 0 {
		return nil, err
	}
	t.Results = append(t.Results, map[string]interface{}{"result": res[0]["result"]})
	return t.Results, nil
}

func (t *TableColumnService) Get(restriction ...string) ([]map[string]interface{}, error) {
	var err error
	restr, order, limit, _ := t.SpecializedService.GenerateQueryFilter(t.Name, restriction...)
	t.DB.ApplyQueryFilters(restr, order, limit, t.Views)
	if t.Results, err = t.DB.SelectQueryWithRestriction(t.Name, map[string]interface{}{}, false); err != nil {
		return t.DBError(nil, err)
	}
	return t.Results, nil
}

func (t *TableColumnService) Verify(name string) (string, bool) {
	var typ string
	if cols, _, err := RetrieveTable(t.Name, t.DB); err == nil {
		if cols[name].Null {
			typ = cols[name].Type + ":nullable"
		} else {
			typ = cols[name].Type + ":required"
		}
	}
	return typ, typ != ""
}

func (t *TableColumnService) Create(record map[string]interface{}) ([]map[string]interface{}, error) {
	queries := t.DB.ClearQueryFilter().BuildCreateQueries(t.Name, "",
		fmt.Sprintf("%v", record["name"]), fmt.Sprintf("%v", record["type"]))
	for i, query := range queries {
		if query == "" {
			return nil, errors.New("missing values")
		}
		if err := t.DB.Query(query); err != nil && len(queries)-1 == i {
			return t.DBError(nil, err)
		}
	}
	t.update(record)
	if len(queries) > 0 {
		t.Views = fmt.Sprintf("%v", record["name"])
		return t.Get()
	}
	return nil, errors.New("no query to execute")
}

func (t *TableColumnService) Update(record map[string]interface{}, restr ...string) ([]map[string]interface{}, error) {
	t.DB.ClearQueryFilter()
	typ := fmt.Sprintf("%v", record["type"])
	name := fmt.Sprintf("%v", record["name"])
	if typ == "" || typ == "<nil>" || name == "" || name == "<nil>" {
		return nil, errors.New("missing one of the needed value type & name")
	}
	if err := t.update(record); err != nil {
		return t.DBError(nil, err)
	}
	if strings.TrimSpace(name) != "" && !strings.Contains(t.Name, "db") {
		col := strings.Split(t.Views, ",")[0]
		query := "ALTER TABLE " + t.Name + " RENAME COLUMN " + col + " TO " + name + ";" // TODO
		err := t.DB.Query(query)
		if err != nil {
			return t.DBError(nil, err)
		}
	}
	return t.Get()
}

func (t *TableColumnService) Delete(restriction ...string) ([]map[string]interface{}, error) {
	if strings.Contains(t.Name, "db") { // protect root db columns
		return nil, errors.New("can't delete protected root db columns")
	}
	for _, col := range strings.Split(t.Views, ",") {
		if err := t.DB.ClearQueryFilter().DeleteQuery(t.Name, col); err != nil {
			return t.DBError(nil, err)
		}
		t.Results = append(t.Results, map[string]interface{}{"name": col})
	}
	return t.Results, nil
}

func (t *TableColumnService) update(record map[string]interface{}) error {
	queries, err := t.DB.BuildUpdateColumnQueries(t.Name, record, nil)
	for _, query := range queries {
		if err2 := t.DB.Query(query); err2 != nil && !strings.Contains(query, "DROP") {
			fmt.Println(query, t.DB.Query(query))
		}
	}
	return err
}
