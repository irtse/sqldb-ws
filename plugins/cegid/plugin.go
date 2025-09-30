// compile with: go build -buildmode=plugin -o plugin.so plugin.go

// plugin.go
package main

import (
	"encoding/csv"
	"slices"
	ds "sqldb-ws/domain/schema/database_resources"
	connector "sqldb-ws/infrastructure/connector/db"
	"strings"

	"fmt"
	"os"
	"sqldb-ws/domain"
	"sqldb-ws/domain/utils"
	"time"

	models "sqldb-ws/plugins/datas"
)

func Run() {
	for {
		ImportUserHierachy()
		ImportProjectAxis()
		time.Sleep(24 * time.Hour)
	}
}

func ImportProjectAxis() {
	mapped := map[int]string{
		4: "code",
		5: "name",
		6: "domain_code",
	}
	d := domain.Domain(true, os.Getenv("SUPERADMIN_NAME"), nil)
	filepath := os.Getenv("PROJECT_FILE_PATH")
	if filepath == "" {
		filepath = "./project_test.csv"
	} else {
		filepath = "/mnt/plugin_files/" + filepath
	}
	headers, datas := importFile(filepath)
	inside := []string{}
	for _, data := range datas {
		record := map[string]interface{}{}
		for i, _ := range headers {
			if realLabel, ok := mapped[i]; ok && realLabel != "" && data[i] != "" {
				record[realLabel] = data[i]
			}
		}
		if len(record) == 3 && !slices.Contains(inside, utils.GetString(record, "name")) {
			inside = append(inside, utils.GetString(record, "name"))
			// TODO Axe : entity binded to user
			if res, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.Axis.Name, map[string]interface{}{
				"code": connector.Quote(utils.GetString(record, "code")),
			}, false); err == nil && len(res) > 0 {
				record[utils.SpecialIDParam] = res[0][utils.SpecialIDParam]
				d.GetDb().UpdateQuery(models.Axis.Name, record, map[string]interface{}{
					utils.SpecialIDParam: res[0][utils.SpecialIDParam],
				}, false)
				continue
			}

			res, err := d.GetDb().CreateQuery(ds.DBEntity.Name, map[string]interface{}{
				"name": record["name"],
			}, func(s string) (string, bool) { return "", true })
			if err == nil {
				record[ds.EntityDBField] = res
				d.GetDb().CreateQuery(models.Axis.Name, record, func(s string) (string, bool) { return "", true })
			}
		}
	}
	mapped = map[int]string{
		0:  "code",
		2:  "project_task",
		3:  "name",
		10: "state",
	}
	for _, data := range datas {
		axisName := ""
		respPrj := int64(-1)
		record := map[string]interface{}{}
		for i, _ := range headers {
			if realLabel, ok := mapped[i]; ok && realLabel != "" && data[i] != "" {
				if strings.ToLower(data[i]) == "non" {
					record[realLabel] = false
				} else if strings.ToLower(data[i]) == "oui" {
					record[realLabel] = true
				} else {
					record[realLabel] = data[i]
				}
			}
			if i == 0 {
				record["code"] = data[i]
			}
			if i == 5 && data[i] != "" {
				axisName = data[i]
			}
			if i == 8 && data[i] != "" {
				t, err := time.Parse("02/01/2006", fmt.Sprintf("%v", data[i]))
				if err == nil {
					record["start_date"] = t.Format("2006-01-02")
				}
			}
			if i == 9 && data[i] != "" {
				t, err := time.Parse("02/01/2006", fmt.Sprintf("%v", data[i]))
				if err == nil {
					record["end_date"] = t.Format("2006-01-02")
				}
			}
			if i == 4 && data[i] != "" {
				if res, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.Axis.Name, map[string]interface{}{
					"code": connector.Quote(data[i]),
				}, false); err == nil && len(res) > 0 {
					record[ds.RootID(models.Axis.Name)] = res[0][utils.SpecialIDParam]
				}
			}
			if i == 12 && data[i] != "" {
				if res, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
					"email": connector.Quote(data[i]),
				}, false); err == nil && len(res) > 0 {
					// TODO Missing add to user entity of project
					respPrj = utils.ToInt64(res[0][utils.SpecialIDParam])
					record[ds.UserDBField] = res[0][utils.SpecialIDParam]
				}
			}
		}
		if len(record) > 0 {
			record["name"] = utils.ToString(record["name"]) + " (" + utils.ToString(record["code"]) + ")"
			// depend to
			var parentID *int64
			if axisName != "" {
				if res, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBEntity.Name, map[string]interface{}{
					"name": connector.Quote(axisName),
				}, false); err == nil && len(res) > 0 {
					i := utils.GetInt(res[0], utils.SpecialIDParam)
					parentID = &i
				}
			}
			if res, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.Project.Name, map[string]interface{}{
				"code": connector.Quote(utils.GetString(record, "code")),
			}, false); err == nil && len(res) > 0 {
				record[utils.SpecialIDParam] = res[0][utils.SpecialIDParam]
				d.GetDb().UpdateQuery(models.Project.Name, record, map[string]interface{}{
					utils.SpecialIDParam: res[0][utils.SpecialIDParam],
				}, false)
				m := map[string]interface{}{
					ds.UserDBField: respPrj,
				}
				m2 := map[string]interface{}{
					ds.UserDBField: respPrj,
				}
				if respPrj >= 0 { // add a CDP to a project
					if res, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBEntity.Name, map[string]interface{}{
						"name": connector.Quote(utils.ToString(res[0]["name"])),
					}, false); err == nil && len(res) > 0 {
						m[ds.EntityDBField] = res[0][utils.SpecialIDParam]
					}
					d.GetDb().DeleteQueryWithRestriction(ds.DBEntityUser.Name, m, false)
					d.GetDb().CreateQuery(ds.DBEntityUser.Name, m, func(s string) (string, bool) { return "", true })
					if res, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBEntity.Name, map[string]interface{}{
						"name": "'CDP'",
					}, false); err == nil && len(res) > 0 {
						m2[ds.EntityDBField] = res[0][utils.SpecialIDParam]
					}
					d.GetDb().DeleteQueryWithRestriction(ds.DBEntityUser.Name, m2, false)
					d.GetDb().CreateQuery(ds.DBEntityUser.Name, m2, func(s string) (string, bool) { return "", true })
				}
				continue
			}
			if parentID != nil {
				res, err := d.GetDb().CreateQuery(ds.DBEntity.Name, map[string]interface{}{
					"name":      record["name"],
					"parent_id": parentID,
				}, func(s string) (string, bool) { return "", true })
				if err == nil {
					record[ds.EntityDBField] = res
					d.GetDb().CreateQuery(models.Project.Name, record, func(s string) (string, bool) { return "", true })
				}
			} else {
				res, err := d.GetDb().CreateQuery(ds.DBEntity.Name, map[string]interface{}{
					"name": record["name"],
				}, func(s string) (string, bool) { return "", true })
				if err == nil {
					record[ds.EntityDBField] = res
					d.GetDb().CreateQuery(models.Project.Name, record, func(s string) (string, bool) { return "", true })
				}
			}
		}
	}
}

func ImportUserHierachy() {
	mapped := map[int]string{
		10: "active",
		0:  "name",
		5:  "email",
		1:  "code",
	}
	d := domain.Domain(true, os.Getenv("SUPERADMIN_NAME"), nil)
	filepath := os.Getenv("USER_FILE_PATH")
	if filepath == "" {
		filepath = "./user_test.csv"
	} else {
		filepath = "/mnt/plugin_files/" + filepath
	}

	headers, datas := importFile(filepath)
	inside := []string{}
	insideCoc := []string{}
	d.GetDb().CreateQuery(ds.DBEntity.Name, map[string]interface{}{
		"name": "CDP",
	}, func(s string) (string, bool) { return "", true })
	for _, data := range datas {
		cocName := ""
		record := map[string]interface{}{}
		for i, _ := range headers {
			if realLabel, ok := mapped[i]; ok && realLabel != "" && data[i] != "" {
				if strings.ToLower(data[i]) == "non" {
					record[realLabel] = false
				} else if strings.ToLower(data[i]) == "oui" {
					record[realLabel] = true
				} else if realLabel == "name" {
					record[realLabel] = strings.ToLower(data[i])
				} else {
					record[realLabel] = data[i]
				}
			} else {
				if i == 6 && data[i] != "" {
					cocName = data[i]
					if !slices.Contains([]string{"CIAC", "CIAA", "CIAS", "CSEC", "CSOM", "CSIS", "CMCP", "CMMP", "CMSA", "CEHT", "CEHF"}, strings.ToUpper(cocName)) {
						cocName = "autre centre de compétence"
					}
					if !slices.Contains(insideCoc, cocName) {
						insideCoc = append(insideCoc, cocName)

						res, err := d.GetDb().CreateQuery(ds.DBEntity.Name, map[string]interface{}{
							"name": cocName,
						}, func(s string) (string, bool) { return "", true })
						if err == nil {
							d.GetDb().CreateQuery(models.CoCFR.Name, map[string]interface{}{
								"name":           cocName,
								ds.EntityDBField: res,
							}, func(s string) (string, bool) { return "", true })
						}
					}
				}
			}
		}
		if len(record) > 0 {
			m := map[string]interface{}{
				ds.UserDBField: d.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
					"name": connector.Quote(utils.ToString(record["name"])),
				}, false, "id"),
				ds.EntityDBField: d.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(
					ds.DBEntity.Name, map[string]interface{}{
						"name": cocName,
					}, false, "id"),
			}
			if cocName != "" { // add a CDP to a project
				if res, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
					"name": connector.Quote(utils.ToString(record["name"])),
				}, false); err == nil && len(res) > 0 {
					m[ds.UserDBField] = res[0][utils.SpecialIDParam]
				}
				if res, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBEntity.Name, map[string]interface{}{
					"name": connector.Quote(cocName),
				}, false); err == nil && len(res) > 0 {
					m[ds.EntityDBField] = res[0][utils.SpecialIDParam]
					d.GetDb().DeleteQueryWithRestriction(ds.DBEntityUser.Name, m, false)
					d.GetDb().CreateQuery(ds.DBEntityUser.Name, m, func(s string) (string, bool) { return "", true })
				}
			}
			if !slices.Contains(inside, utils.GetString(record, "name")) {
				inside = append(inside, utils.GetString(record, "name"))
				if utils.GetBool(record, "active") {
					if res, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
						"name": connector.Quote(utils.GetString(record, "name")),
					}, false); err == nil && len(res) > 0 {
						record[utils.SpecialIDParam] = res[0][utils.SpecialIDParam]
						d.GetDb().UpdateQuery(ds.DBUser.Name, record, map[string]interface{}{
							utils.SpecialIDParam: res[0][utils.SpecialIDParam],
						}, false)
					} else {
						d.GetDb().CreateQuery(ds.DBUser.Name, record, func(s string) (string, bool) { return "", true })
					}
				}
			}
		}
	}
	for _, data := range datas {
		userID := ""
		hierarchyID := ""
		for i, _ := range headers {
			if i == 5 && data[i] != "" {
				if res, err := d.Db.ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
					"email": connector.Quote(data[i]),
				}, false); err == nil && len(res) > 0 {
					userID = utils.GetString(res[0], utils.SpecialIDParam)
				}
			}
			if i == 1 && data[i] != "" {
				if res, err := d.Db.ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
					"code": connector.Quote(data[i]),
				}, false); err == nil && len(res) > 0 {
					hierarchyID = utils.GetString(res[0], utils.SpecialIDParam)
				}
			}
		}
		if userID != "" && hierarchyID != "" {
			d.GetDb().DeleteQueryWithRestriction(ds.DBHierarchy.Name, map[string]interface{}{
				ds.UserDBField:             userID,
				"parent_" + ds.UserDBField: hierarchyID,
			}, false)
			d.GetDb().CreateQuery(ds.DBHierarchy.Name, map[string]interface{}{
				"parent_" + ds.UserDBField: hierarchyID,
				ds.UserDBField:             userID,
			}, func(s string) (string, bool) { return "", true })
		}
	}
}

func importFile(filePath string) ([]string, [][]string) {
	// Open CSV file
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Failed to open file:", err)
		return []string{}, [][]string{}
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Failed to read CSV:", err)
		return []string{}, [][]string{}
	}

	if len(records) < 2 {
		fmt.Println("Not enough rows to sort")
		return []string{}, [][]string{}
	}

	headers := records[0]
	datas := records[1:]
	return headers, datas
}
