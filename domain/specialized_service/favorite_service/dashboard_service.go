package favorite_service

import (
	"errors"
	"fmt"
	"sqldb-ws/domain/domain_service/filter"
	"sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	"sqldb-ws/domain/schema/models"
	servutils "sqldb-ws/domain/specialized_service/utils"
	utils "sqldb-ws/domain/utils"
	connector "sqldb-ws/infrastructure/connector/db"
	"strconv"
	"strings"
)

// DONE - ~ 200 LINES - PARTIALLY TESTED
type DashboardService struct {
	servutils.AbstractSpecializedService
	Elements       []map[string]interface{}
	UpdateElements bool
}

// mainly should deserialize the data from the database
// into a format that can be used by the front-end to display the data
func (s *DashboardService) Entity() utils.SpecializedServiceInfo                                    { return ds.DBDashboard }
func (s *DashboardService) SpecializedDeleteRow(results []map[string]interface{}, tableName string) {}
func (s *DashboardService) SpecializedUpdateRow(results []map[string]interface{}, record map[string]interface{}) {
	s.Write(record, ds.DBFilter.Name)
	s.AbstractSpecializedService.SpecializedUpdateRow(results, record)
}

func (s *DashboardService) CreateDashboardMathOperation(elementID string, record map[string]interface{}) error {
	if elementID == "" {
		return errors.New("element id is required")
	}
	record[ds.DashboardElementDBField] = elementID
	_, err := s.Domain.CreateSuperCall(utils.AllParams(ds.DBDashboardMathField.Name).RootRaw(), record)
	return err
}

func (s *DashboardService) CreateDashboardElement(dashboardID string, record map[string]interface{}) error {
	if dashboardID == "" {
		return errors.New("dashboard id is required")
	}
	record[ds.DashboardDBField] = dashboardID
	_, err := s.Domain.CreateSuperCall(utils.AllParams(ds.DBDashboardElement.Name).RootRaw(), record)
	return err
}

func (s *DashboardService) SpecializedCreateRow(record map[string]interface{}, tableName string) {
	s.Write(record, tableName)
	s.AbstractSpecializedService.SpecializedCreateRow(record, tableName)
}

func (s *DashboardService) Write(record map[string]interface{}, tableName string) {
	for _, element := range s.Elements {
		err := s.CreateDashboardElement(utils.ToString(record[utils.SpecialIDParam]), element)
		if fields, ok := record["fields"]; ok && err == nil {
			for _, field := range utils.ToList(fields) {
				err = s.CreateDashboardMathOperation(utils.ToString(record[utils.SpecialIDParam]), utils.ToMap(field))
				if err != nil {
					break
				}
			}
		}
	}
}
func (s *DashboardService) GenerateQueryFilter(tableName string, innerestr ...string) (string, string, string, string) {
	return filter.NewFilterService(s.Domain).GetQueryFilter(tableName, s.Domain.GetParams().Copy(), false, innerestr...)
}

func (s *DashboardService) VerifyDataIntegrity(record map[string]interface{}, tablename string) (map[string]interface{}, error, bool) {
	s.UpdateElements = false
	method := s.Domain.GetMethod()

	if method != utils.DELETE {
		s.ProcessName(record)
		s.ProcessElements(record)

		if method == utils.UPDATE && s.UpdateElements {
			s.HandleUpdate(record)
		}
	} else {
		s.HandleDelete(record)
	}

	s.ProcessSelection(record)
	return s.AbstractSpecializedService.VerifyDataIntegrity(record, tablename)
}

func (s *DashboardService) ProcessName(record map[string]interface{}) {
	if name, ok := record["name"]; ok {
		if result, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBDashboard.Name, map[string]interface{}{
			"name": name,
		}, false); err == nil && len(result) > 0 {
			record[utils.SpecialIDParam] = result[0][utils.SpecialIDParam]
		}
	}
}

func (s *DashboardService) ProcessElements(record map[string]interface{}) {
	if els, ok := record["elements"]; ok {
		s.UpdateElements = true
		s.Elements = make([]map[string]interface{}, 0)
		for _, el := range utils.ToList(els) {
			s.Elements = append(s.Elements, utils.ToMap(el))
		}
	}
}

func (s *DashboardService) HandleUpdate(record map[string]interface{}) {
	s.HandleDelete(record)
}

func (d *DashboardService) HandleDelete(record map[string]interface{}) {
	elements, err := d.getDashBoardElement(utils.ToString(record[utils.SpecialIDParam]))
	if err != nil {
		return
	}
	for _, element := range elements {
		// delete the operator element
		if element[utils.SpecialIDParam] != nil && element[utils.SpecialIDParam] != "" {
			params := utils.AllParams(ds.DBDashboardElement.Name).RootRaw()
			params.Set(ds.DashboardElementDBField, utils.ToString(element[utils.SpecialIDParam]))
			d.Domain.DeleteSuperCall(params)
			params = utils.AllParams(ds.DBDashboardElement.Name)
			params.Set(ds.DashboardDBField, utils.ToString(record[utils.SpecialIDParam]))
			d.Domain.DeleteSuperCall(params)
		}
	}
}

func (d *DashboardService) getDashBoardElement(dashboardID string) ([]map[string]interface{}, error) {
	return d.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBDashboardElement.Name, map[string]interface{}{
		ds.DashboardDBField: dashboardID,
	}, false)
}

func (d *DashboardService) GetDashboardElementView(dashboardID string) utils.Results {
	results := utils.Results{}

	elements, err := d.getDashBoardElement(dashboardID)
	if err != nil || len(elements) == 0 {
		return results
	}

	for _, element := range elements {
		res, err := d.ProcessDashboardElement(element)
		if err != nil {
			continue
		}
		results = append(results, res)
	}

	return results
}

func (d *DashboardService) ProcessDashboardElement(element map[string]interface{}) (utils.Record, error) {
	filt, ok := element[ds.FilterDBField]
	if !ok {
		return nil, errors.New("missing filter field")
	}

	restriction, orderBy, err := d.GetFilterRestrictionAndOrder(filt, element)
	if err != nil {
		return nil, err
	}

	res, isMultiple, err := d.getDashBoardMathFieldView(
		utils.ToString(element[utils.SpecialIDParam]),
		restriction, orderBy,
	)
	if err != nil {
		return nil, err
	}

	return utils.Record{
		"name":        utils.ToString(element[models.NAMEKEY]),
		"description": utils.ToString(element["description"]),
		"is_multiple": isMultiple,
		"results":     res,
	}, nil
}

func (d *DashboardService) GetFilterRestrictionAndOrder(filt interface{}, element map[string]interface{}) (string, string, error) {
	var restriction, orderBy string
	f := filter.NewFilterService(d.Domain)

	res, err := d.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBFilter.Name, map[string]interface{}{utils.SpecialIDParam: filt}, false)
	if err != nil || len(res) == 0 {
		return "", "", fmt.Errorf("failed to fetch filter: %v", err)
	}

	sch, err := schema.GetSchema(utils.ToString(res[0][ds.SchemaDBField]))
	if err != nil {
		return "", "", fmt.Errorf("failed to get schema: %v", err)
	}

	_, restriction = f.ProcessFilterRestriction(utils.ToString(filt), sch)

	if orderID, exists := element["order_by_"+ds.SchemaDBField]; exists {
		if i, err := strconv.Atoi(utils.ToString(orderID)); err == nil {
			if field, err := sch.GetFieldByID(int64(i)); err == nil {
				orderBy = field.Name
			}
		}
	}

	return restriction, orderBy, nil
}
func (d *DashboardService) getDashBoardMathFieldView(elementID, restriction, orderBy string) (utils.Results, bool, error) {
	if elementID == "" {
		return nil, false, errors.New("element id is required")
	}
	db := d.Domain.GetDb()
	fields, err := db.SelectQueryWithRestriction(ds.DBDashboardMathField.Name, map[string]interface{}{
		ds.DashboardElementDBField: elementID,
	}, false)
	if err != nil {
		return nil, false, err
	}

	names, views, err := d.processMathFields(fields)
	if err != nil {
		return nil, false, err
	}

	db.ClearQueryFilter()
	db.SQLRestriction = restriction
	db.SQLOrder = orderBy
	db.SQLView = strings.Join(views, ",")

	r, err := db.SelectQueryWithRestriction("", map[string]interface{}{}, false)
	if err != nil {
		return nil, false, err
	}

	return d.ProcessMathResults(r, names)
}

func (d *DashboardService) processMathFields(fields []map[string]interface{}) ([]string, []string, error) {
	names := []string{}
	views := []string{}

	for _, field := range fields {
		name, rowAlgo, colAlgo, err := d.ExtractMathFieldData(field)
		if err != nil {
			return nil, nil, err
		}

		names = append(names, name)
		views = append(views, connector.FormatMathViewQuery(colAlgo, rowAlgo, name))
	}

	return names, views, nil
}

func (d *DashboardService) ExtractMathFieldData(field map[string]interface{}) (string, string, string, error) {
	name := utils.ToString(field[models.NAMEKEY])
	if name == "" {
		return "", "", "", errors.New("name is required")
	}

	rowAlgo := utils.ToString(field["row_math_func"])
	if rowAlgo == "" {
		return "", "", "", errors.New("row math func is required")
	}
	return name, rowAlgo, utils.ToString(field["column_math_func"]), nil
}

func (d *DashboardService) ProcessMathResults(r []map[string]interface{}, names []string) (utils.Results, bool, error) {
	results := utils.Results{}
	isMultiple := len(r) > 1

	for i, rec := range r {
		for _, n := range names {
			f, err := strconv.ParseFloat(utils.ToString(rec[n]), 64)
			if err != nil {
				continue
			}

			name := n
			if isMultiple {
				name += utils.ToString(i)
			}

			results = append(results, utils.Record{
				"name":  name,
				"value": f,
			})
		}
	}

	return results, isMultiple, nil
}

func (s *DashboardService) ProcessSelection(record map[string]interface{}) {
	if sel, ok := record["is_selected"]; ok && utils.Compare(sel, true) { // TODO
		s.Domain.GetDb().UpdateQuery(ds.DBDashboard.Name, utils.Record{
			"is_selected": false,
		}, map[string]interface{}{
			ds.DashboardDBField: record[ds.DashboardDBField],
		}, true)
	}
}
