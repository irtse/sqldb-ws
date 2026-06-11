// compile with: go build -buildmode=plugin -o plugin.so plugin.go

// plugin.go
package main

import (
	"encoding/csv"
	ds "sqldb-ws/domain/schema/database_resources"
	"sqldb-ws/domain/specialized_service/task_service"
	connector "sqldb-ws/infrastructure/connector/db"
	"strings"
	"time"

	"fmt"
	"os"
	"sqldb-ws/domain"
	"sqldb-ws/domain/utils"

	models "sqldb-ws/plugins/datas"
)

func Run() {
	ImportPublication()
}

func ImportPublication() {
	d := domain.Domain(true, os.Getenv("SUPERADMIN_NAME"), nil)
	filepath := os.Getenv("PUBLICATION_FILE_PATH")
	if filepath == "" {
		filepath = "./publication_test.csv"
	} else {
		filepath = "/mnt/plugin_files/" + filepath
	}

	mapped := map[int]string{
		5:  "name",
		8:  "conference_name",
		9:  "conference_start_date", // date for date
		10: "conference_end_date",
		11: "conference_city",
		12: "conference_country",
		13: "conference_link",
		14: "media_name",
		15: "publishing_date",                         // date for date
		23: "effective_publishing_date",               // date for date
		25: "authors",                                 // special OK
		26: "affiliation",                             // special OK
		27: "IRT_manager" + ds.RootID(ds.DBUser.Name), // special OK
		28: "i_start_date",                            // date for date
		29: "i_end_date",
		30: "is_awarded",                            // special OK
		31: "defense_date",                          // date for date
		34: "director_" + ds.RootID(ds.DBUser.Name), // special
		35: "t_start_date",                          // date for date
		36: "t_end_date",
		37: "meeting_name",
		38: "meeting_date",                         // date for date
		39: "manager_" + ds.RootID(ds.DBUser.Name), // special OK
		41: "state",                                // special OK
		42: "active",                               // special OK
		3:  "project_accronym",                     // special OK
	}
	// TODO finalized_publication failed
	_, datas := importFile(filepath)
	for _, data := range datas {
		model := map[string]interface{}{}

		dbName := models.OtherPublicationFR.Name
		affDbName := models.OtherPublicationAffiliationAuthorsFR.Name
		authorsDbName := models.OtherPublicationAuthorsFR.Name
		dt := []int{5, 3, 30, 23, 25, 26, 39, 8, 15}
		if strings.ToLower(data[4]) != "these" {
			dt = []int{5, 3, 30, 23, 25, 26, 34, 39, 35, 363}
			dbName = models.ThesisFR.Name
			affDbName = models.ThesisAffiliationAuthorsFR.Name
			authorsDbName = models.ThesisAffiliationAuthorsFR.Name
		} else if strings.ToLower(data[4]) != "stage" {
			dt = []int{5, 3, 30, 23, 25, 26, 39, 27, 28, 29}
			dbName = models.InternshipFR.Name
			affDbName = models.InternshipAffiliationAuthorsFR.Name
			authorsDbName = models.InternshipAuthorsFR.Name
		} else if strings.ToLower(data[4]) != "poster" {
			dt = []int{5, 3, 30, 23, 25, 26, 39, 8, 9, 11, 12, 13}
			dbName = models.PosterFR.Name
			affDbName = models.PosterAffiliationAuthorsFR.Name
			authorsDbName = models.PosterAuthorsFR.Name
		} else if strings.ToLower(data[4]) != "demo" {
			dt = []int{5, 3, 30, 23, 25, 26, 39, 37, 38}
			dbName = models.DemoFR.Name
			affDbName = models.DemoAffiliationAuthorsFR.Name
			authorsDbName = models.DemoAuthorsFR.Name
		} else if strings.ToLower(data[4]) != "article" {
			dt = []int{5, 3, 30, 23, 25, 26, 39, 14, 15}
			dbName = models.ArticleFR.Name
			affDbName = models.ArticleAffiliationAuthorsFR.Name
			authorsDbName = models.ArticleAuthorsFR.Name
		} else if strings.ToLower(data[4]) != "communication" {
			// TODO DEFINE IF CONFERENCE OR PRESENTATION
		} else if strings.ToLower(data[4]) != "article_bdd" {
			// TODO DEFINE IF CONFERENCE OR PRESENTATION
			dt = []int{5, 3, 8, 30, 23, 25, 26, 39, 37, 38}
			dbName = models.OtherPublicationFR.Name
			affDbName = models.OtherPublicationAffiliationAuthorsFR.Name
			authorsDbName = models.OtherPublicationAuthorsFR.Name
		}
		date := ""
		// TODO FILE RETRIEVAL +
		for _, i := range dt {
			if (i == 21 || i == 9 || i == 15 || i == 23 || i == 28 || i == 31 || i == 35 || i == 38) && date == "" { // format d/m/y
				date = data[i]
			}

			if i == 42 {
				if data[i] == "0" {
					model[mapped[i]] = false
					continue
				} else {
					model[mapped[i]] = true
				}
			} else if i == 41 {
				if model["state"] == nil {
					if strings.ToLower(data[i]) == "init" {
						if st, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.PublicationStatusFR.Name, []interface{}{
							"name::text LIKE '%init%'",
						}, false); err == nil && len(st) > 0 {
							model["state"] = st[0][utils.SpecialIDParam]
						}
					} else if strings.Contains(strings.ToLower(data[i]), "annul") || strings.Contains(strings.ToLower(data[i]), "refu") {
						if st, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.PublicationStatusFR.Name, []interface{}{
							"name::text LIKE '%ban%'",
						}, false); err == nil && len(st) > 0 {
							model["state"] = st[0][utils.SpecialIDParam]
						}
					} else if strings.Contains(strings.ToLower(data[i]), "réalis") {
						if model["is_awarded"] == true {
							if st, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.PublicationStatusFR.Name, []interface{}{
								"name::text LIKE '%prim%'",
							}, false); err == nil && len(st) > 0 {
								model["state"] = st[0][utils.SpecialIDParam]
							}
						} else {
							if st, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.PublicationStatusFR.Name, []interface{}{
								"name::text LIKE '%pub%'",
							}, false); err == nil && len(st) > 0 {
								model["state"] = st[0][utils.SpecialIDParam]
							}
						}

					} else {
						if st, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.PublicationStatusFR.Name, []interface{}{
							"name::text LIKE '%aut%'",
						}, false); err == nil && len(st) > 0 {
							model["state"] = st[0][utils.SpecialIDParam]
						}
					}
				}

			} else if i == 30 {
				if strings.Trim(data[i], " ") == "oui" {
					model[mapped[i]] = true
					if st, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.PublicationStatusFR.Name, []interface{}{
						"name::text LIKE '%prim%'",
					}, false); err == nil && len(st) > 0 {
						model["state"] = st[0][utils.SpecialIDParam]
					}
				} else {
					model[mapped[i]] = false
				}
			} else if i == 39 {
				c := data[i]
				for {
					if len(c) < 10 {
						c = "0" + c
					} else {
						break
					}
				}

				if usr, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
					"code": connector.Quote(c),
				}, false); err == nil && len(usr) > 0 {
					model[mapped[i]] = usr[0][utils.SpecialIDParam]
					coc, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.CoCFR.Name, map[string]interface{}{
						"name": d.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBEntity.Name, map[string]interface{}{
							utils.SpecialIDParam: d.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBEntityUser.Name, map[string]interface{}{
								ds.UserDBField: usr[0][utils.SpecialIDParam],
							}, false, ds.EntityDBField),
						}, false, "name"),
					}, false)
					if err == nil && len(coc) > 0 {
						model["competence_center"] = coc[0][utils.SpecialIDParam]
					}
				}
			} else if i == 27 {
				restr := []interface{}{}
				data[i] = strings.ReplaceAll(data[i], ";", ",")
				for _, auth := range strings.Split(data[i], ",") {
					for _, n := range strings.Split(auth, " ") {
						restr = append(restr, "name::text LIKE '%"+strings.Trim(strings.ReplaceAll(n, "'", "''"), " ")+"%'")
					}
				}
				if usr, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, restr, false); err == nil && len(usr) > 0 {
					model[mapped[i]] = usr[0][utils.SpecialIDParam]
					break
				}
			} else if i == 3 {
				if prj, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.Project.Name, map[string]interface{}{
					"code": connector.Quote(data[i]),
				}, false); err == nil && len(prj) > 0 {
					model[mapped[i]] = prj[0][utils.SpecialIDParam]
					if res, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.Axis.Name, map[string]interface{}{
						utils.SpecialIDParam: prj[0][ds.RootID(models.Axis.Name)],
					}, false); err == nil && len(res) > 0 {
						model["axis"] = res[0][utils.SpecialIDParam]
					} else if res, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.Axis.Name, map[string]interface{}{
						"code": "NDE",
					}, false); err == nil && len(res) > 0 {
						model["axis"] = res[0][utils.SpecialIDParam]
					}
				}
			} else if i == 25 {
				if model["authors"] == nil {
					model["authors"] = []map[string]interface{}{}
				}
				data[i] = strings.ReplaceAll(data[i], ";", ",")
				for y, authors := range strings.Split(data[i], ",") {
					if len(model["authors"].([]map[string]interface{})) < y {
						model["authors"] = append(model["authors"].([]map[string]interface{}), map[string]interface{}{})
					}
					restr := []interface{}{}
					for _, n := range strings.Split(authors, " ") {
						restr = append(restr, "name::text LIKE '%"+strings.Trim(strings.ReplaceAll(n, "'", "''"), " ")+"%'")
					}
					if usr, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, restr, false); err == nil && len(usr) > 0 {
						model["authors"].([]map[string]interface{})[y]["authors"] = map[string]interface{}{
							ds.UserDBField: usr[0][utils.SpecialIDParam],
						}
					} else {
						model["authors"].([]map[string]interface{})[y]["authors"] = map[string]interface{}{
							"name": strings.Trim(authors, " "),
						}
					}
				}
			} else if i == 26 {
				if model["authors"] == nil {
					model["authors"] = []map[string]interface{}{}
				}
				for y, aff := range strings.Split(data[i], ",") {
					if len(model["authors"].([]map[string]interface{})) < y {
						model["authors"] = append(model["authors"].([]map[string]interface{}), map[string]interface{}{})
					}
					model["authors"].([]map[string]interface{})[y]["affiliation"] = aff
				}
			} else {
				model[mapped[i]] = data[i]
			}
			// TODO check special field like project, authors, affiliation... etc.
		}
		m2 := map[string]interface{}{}
		for k, v := range model {
			m2[k] = v
		}
		if res, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(dbName, map[string]interface{}{
			"name": connector.Quote(utils.GetString(model, "name")),
		}, false); err == nil && len(res) == 0 {
			delete(m2, "authors")
			if id, err := d.GetDb().ClearQueryFilter().CreateQuery(dbName, m2, func(s string) (string, bool) { return s, true }); err == nil {
				createDate, err := time.Parse("02/01/2006", date)
				if err != nil {
					createDate = time.Time{}
				}
				_, err = d.GetDb().ClearQueryFilter().CreateQuery(ds.DBDataAccess.Name, map[string]interface{}{
					"write":             true,
					"access_date":       createDate,
					ds.DestTableDBField: id,
					ds.SchemaDBField: d.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBSchema.Name, map[string]interface{}{
						"name": dbName,
					}, false, utils.SpecialIDParam),
					ds.UserDBField: model["manager_"+ds.RootID(ds.DBUser.Name)],
				}, func(s string) (string, bool) { return s, true })
				for _, auth := range model["authors"].([]map[string]interface{}) {
					authorss := auth["authors"]
					delete(auth, "authors")
					auth[ds.RootID(dbName)] = id
					if id, err := d.GetDb().ClearQueryFilter().CreateQuery(affDbName, auth, func(s string) (string, bool) { return s, true }); err == nil {
						authorss.(map[string]interface{})[ds.RootID(dbName)] = id
						d.GetDb().ClearQueryFilter().CreateQuery(authorsDbName, authorss.(map[string]interface{}), func(s string) (string, bool) { return s, true })
					}
				}
				_, err = d.GetDb().ClearQueryFilter().CreateQuery(models.PublicationHistoryStatusFR.Name, map[string]interface{}{
					ds.RootID(models.PublicationStatusFR.Name): model["state"],
					"update_date":       createDate,
					ds.DestTableDBField: id,
					ds.SchemaDBField: d.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBSchema.Name, map[string]interface{}{
						"name": dbName,
					}, false, utils.SpecialIDParam),
					ds.UserDBField: model["manager_"+ds.RootID(ds.DBUser.Name)],
				}, func(s string) (string, bool) { return s, true })
				if (model["state"] == 3 || model["state"] == 5) && model["manager_"+ds.RootID(ds.DBUser.Name)] != nil {
					if wfs, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBWorkflow.Name, map[string]interface{}{
						ds.SchemaDBField: d.Db.BuildSelectQueryWithRestriction(ds.DBSchema.Name, map[string]interface{}{
							"name": dbName,
						}, false, utils.SpecialIDParam),
					}, false); err == nil && len(wfs) > 0 {
						m := map[string]interface{}{
							"name":              "APU retrieval " + utils.ToString(model["name"]),
							"state":             "pending",
							"is_close":          false,
							"current_index":     1,
							ds.DestTableDBField: id,
							ds.SchemaDBField: d.Db.BuildSelectQueryWithRestriction(ds.DBSchema.Name, map[string]interface{}{
								"name": dbName,
							}, false, utils.SpecialIDParam),
							ds.WorkflowDBField: res[0][utils.SpecialIDParam],
							ds.UserDBField:     model["manager_"+ds.RootID(ds.DBUser.Name)],
						}

						if i, err := d.GetDb().ClearQueryFilter().CreateQuery(ds.DBRequest.Name, m, func(s string) (string, bool) { return "", true }); err != nil {
							if wfss, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBWorkflowSchema.Name, map[string]interface{}{
								"index":            1,
								ds.WorkflowDBField: wfs[0][utils.SpecialIDParam],
							}, false); err == nil && len(wfss) > 0 {
								m["id"] = i
								task_service.PrepareAndCreateTask(wfss[0], m, m, d, false)
							}
						}
					}
				}
			}
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
