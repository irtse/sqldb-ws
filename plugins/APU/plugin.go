// compile with: go build -buildmode=plugin -o plugin.so plugin.go

// plugin.go
package main

import (
	"encoding/csv"
	"os/exec"
	"sqldb-ws/domain/schema"
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
	fmt.Println("RUN APU")
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
		47: "finalized_publication",
		3:  "project_accronym", // special OK // TODO ajouté dans un csv.
	}
	_, datas := importFile(filepath)
	for _, data := range datas {
		model := map[string]interface{}{}

		dbName := models.OtherPublicationFR.Name
		affDbName := models.OtherPublicationAffiliationAuthorsFR.Name
		authorsDbName := models.OtherPublicationAuthorsFR.Name
		dt := []int{5, 41, 42, 3, 30, 23, 25, 26, 39, 8, 15, 47}
		if strings.Contains(strings.ToLower(data[4]), "these") {
			dt = []int{5, 41, 42, 3, 30, 23, 25, 26, 34, 39, 35, 363, 47}
			dbName = models.ThesisFR.Name
			affDbName = models.ThesisAffiliationAuthorsFR.Name
			authorsDbName = models.ThesisAffiliationAuthorsFR.Name
		} else if strings.Contains(strings.ToLower(data[4]), "stage") {
			dt = []int{5, 41, 42, 3, 30, 23, 25, 26, 39, 27, 28, 29, 47}
			dbName = models.InternshipFR.Name
			affDbName = models.InternshipAffiliationAuthorsFR.Name
			authorsDbName = models.InternshipAuthorsFR.Name
		} else if strings.Contains(strings.ToLower(data[4]), "poster") {
			dt = []int{5, 41, 42, 3, 30, 23, 25, 26, 39, 8, 9, 11, 12, 13, 47}
			dbName = models.PosterFR.Name
			affDbName = models.PosterAffiliationAuthorsFR.Name
			authorsDbName = models.PosterAuthorsFR.Name
		} else if strings.Contains(strings.ToLower(data[4]), "demo") {
			dt = []int{5, 41, 42, 3, 30, 23, 25, 26, 39, 37, 38, 47}
			dbName = models.DemoFR.Name
			affDbName = models.DemoAffiliationAuthorsFR.Name
			authorsDbName = models.DemoAuthorsFR.Name
		} else if strings.Contains(strings.ToLower(data[4]), "article_bdd") {
			// TODO DEFINE IF CONFERENCE OR PRESENTATION
			dt = []int{5, 41, 42, 3, 8, 30, 23, 25, 26, 39, 37, 38, 47}
			dbName = models.OtherPublicationFR.Name
			affDbName = models.OtherPublicationAffiliationAuthorsFR.Name
			authorsDbName = models.OtherPublicationAuthorsFR.Name
		} else if strings.Contains(strings.ToLower(data[4]), "article") {
			dt = []int{5, 41, 42, 3, 30, 23, 25, 26, 39, 14, 15, 47}
			dbName = models.ArticleFR.Name
			affDbName = models.ArticleAffiliationAuthorsFR.Name
			authorsDbName = models.ArticleAuthorsFR.Name
		} else if strings.Contains(strings.ToLower(data[4]), "communication") {
			// TODO DEFINE IF CONFERENCE OR PRESENTATION
		}
		missing_project := []string{}
		date := ""
		no_model := false
		// TODO FILE RETRIEVAL +
		for _, i := range dt {
			if (i == 21 || i == 9 || i == 15 || i == 23 || i == 28 || i == 31 || i == 35 || i == 38) && date == "" { // format d/m/y
				date = data[i]
			}
			if i == 47 && data[45] != "" {
				if strings.Contains(data[45], "abstract") {
					model["abstract_publication"] = "/apu_files/" + data[i]
				} else {
					if strings.Contains(data[45], "fin") {
						model[mapped[i]] = "/apu_files/" + data[i]
					} else if model[mapped[i]] == nil || model[mapped[i]] == "" {
						model[mapped[i]] = "/apu_files/" + data[i]
					}
				}
			} else if i == 42 {
				if data[i] == "0" {
					model[mapped[i]] = false
					no_model = true
					break
				} else {
					model[mapped[i]] = true
				}
			} else if i == 41 {
				if strings.Contains(strings.ToLower(data[i]), "init") {
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
						"name::text LIKE '%cour%'",
					}, false); err == nil && len(st) > 0 {
						model["state"] = st[0][utils.SpecialIDParam]
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
				} else {
					// TODO

				}
			} else if i == 27 || i == 34 {
				restr := []interface{}{}
				data[i] = strings.ReplaceAll(data[i], ";", ",")
				for _, auth := range strings.Split(data[i], ",") {
					for _, n := range strings.Split(auth, " ") {
						restr = append(restr, "name::text LIKE '%"+strings.Trim(strings.ReplaceAll(n, "'", "''"), " ")+"%'")
					}
				}
				if usr, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, restr, false); err == nil && len(usr) > 0 {
					model[mapped[i]] = usr[0][utils.SpecialIDParam]
				} else if len(strings.Split(data[i], ",")) > 0 {
					iu, err := d.GetDb().ClearQueryFilter().CreateQuery(ds.DBUser.Name, map[string]interface{}{
						"name": strings.Split(data[i], ",")[0],
					}, func(s string) (string, bool) { return "", true })
					if err == nil {
						model[mapped[i]] = iu
					} else {
						model[mapped[i]] = 1 // root
					}
				} else {
					model[mapped[i]] = 1 // root
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
				} else {
					missing_project = append(missing_project, data[i])
					no_model = true
					break
				}
			} else if i == 25 {
				a := []map[string]interface{}{}
				data[i] = strings.ReplaceAll(data[i], ";", ",")
				aa := []map[string]interface{}{}
				for y, authors := range strings.Split(data[i], ",") {
					restr := []interface{}{}
					fmt.Println("FOUND AUTHOR", authors)
					for _, n := range strings.Split(authors, " ") {
						restr = append(restr, "name::text LIKE '%"+strings.Trim(strings.ReplaceAll(strings.ToLower(n), "'", "''"), " ")+"%'")
					}
					if usr, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, restr, false); err == nil && len(usr) > 0 && len(model["authors"].([]map[string]interface{})) > y {
						aa = append(aa, map[string]interface{}{
							ds.UserDBField: usr[0][utils.SpecialIDParam],
						})
					} else {
						aa = append(aa, map[string]interface{}{
							"name": strings.Trim(authors, " "),
						})
					}
				}
				a = append(a, map[string]interface{}{
					"authors": aa,
				})
				model["authors"] = a
			} else if i == 26 {
				if model["authors"] == nil {
					model["authors"] = []map[string]interface{}{}
				}
				for y, aff := range strings.Split(data[i], ",") {
					if len(model["authors"].([]map[string]interface{})) < y {
						model["authors"] = append(model["authors"].([]map[string]interface{}), map[string]interface{}{})
					}
					if len(model["authors"].([]map[string]interface{})) > y {
						model["authors"].([]map[string]interface{})[y]["affiliation"] = aff
					}

				}
			} else if len(data) > i {
				fmt.Println(model, mapped[i], i)
				if createDate, err := time.Parse("02/01/2006", data[i]); err != nil {
					model[mapped[i]] = data[i]
				} else {
					model[mapped[i]] = createDate
				}
			}
			// TODO check special field like project, authors, affiliation... etc.
		}
		if no_model {
			continue // do not create anything
		}
		cmd := exec.Command("./id_script.sh", missing_project...) // generate csv with missing project

		_, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println("Erreur :", err)
		}
		fmt.Println("MODEL", dbName, model)
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
				if sch, err := schema.GetSchema(dbName); err == nil {
					_, err = d.GetDb().ClearQueryFilter().CreateQuery(ds.DBDataAccess.Name, map[string]interface{}{
						"write":             true,
						"access_date":       createDate,
						ds.DestTableDBField: id,
						ds.SchemaDBField:    sch.ID,
						ds.UserDBField:      m2["manager_"+ds.RootID(ds.DBUser.Name)],
					}, func(s string) (string, bool) { return s, true })
					fmt.Println("DATAACCESS", err)

					if model["authors"] != nil {
						for _, auth := range model["authors"].([]map[string]interface{}) {
							fmt.Println("AUTH", auth)
							if auth["authors"] == nil {
								continue
							}
							m := []map[string]interface{}{}
							for _, v := range auth["authors"].([]map[string]interface{}) {
								m = append(m, v)
							}
							delete(auth, "authors")
							auth[ds.RootID(dbName)] = id

							fmt.Println(affDbName, auth, m)
							if id, err := d.GetDb().ClearQueryFilter().CreateQuery(affDbName, auth, func(s string) (string, bool) { return s, true }); err == nil {
								for _, mm := range m {
									mm[ds.RootID(affDbName)] = id
									_, err := d.GetDb().ClearQueryFilter().CreateQuery(authorsDbName, mm, func(s string) (string, bool) { return s, true })
									fmt.Println("AUTHORS", mm, err)
								}
							} else {
								fmt.Println("AFF", err)
							}
						}
					}

					if r, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(dbName, map[string]interface{}{
						"name": connector.Quote(utils.GetString(model, "name")),
					}, false); err == nil && len(r) > 0 {

						_, err = d.GetDb().ClearQueryFilter().CreateQuery(models.PublicationHistoryStatusFR.Name, map[string]interface{}{
							ds.RootID(models.PublicationStatusFR.Name): r[0]["state"],
							"update_date":       createDate,
							ds.DestTableDBField: id,
							ds.SchemaDBField:    sch.ID,
						}, func(s string) (string, bool) { return s, true })
						if (utils.ToString(r[0]["state"]) == "1" || utils.ToString(r[0]["state"]) == "6") && r[0]["manager_"+ds.RootID(ds.DBUser.Name)] != nil {
							if wfs, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBWorkflow.Name, map[string]interface{}{
								ds.SchemaDBField: sch.ID,
							}, false); err == nil && len(wfs) > 0 {
								m := map[string]interface{}{
									"name":              "APU retrieval " + utils.ToString(r[0]["name"]),
									"state":             "pending",
									"is_close":          false,
									"current_index":     1,
									ds.DestTableDBField: id,
									ds.SchemaDBField:    sch.ID,
									ds.WorkflowDBField:  wfs[0][utils.SpecialIDParam],
									ds.UserDBField:      r[0]["manager_"+ds.RootID(ds.DBUser.Name)],
								}

								if i, err := d.GetDb().ClearQueryFilter().CreateQuery(ds.DBRequest.Name, m, func(s string) (string, bool) { return "", true }); err == nil {
									if wfss, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBWorkflowSchema.Name, map[string]interface{}{
										"index":            1,
										ds.WorkflowDBField: wfs[0][utils.SpecialIDParam],
									}, false); err == nil && len(wfss) > 0 {
										m["id"] = i
										newTask := task_service.ConstructNotificationTask(wfss[0], m, d)
										newTask[ds.UserDBField] = r[0]["manager_"+ds.RootID(ds.DBUser.Name)]
										newTask[ds.SchemaDBField] = sch.ID
										newTask[ds.DestTableDBField] = id
										_, err := d.GetDb().ClearQueryFilter().CreateQuery(ds.DBTask.Name, newTask, func(s string) (string, bool) {
											return "", true
										})
										if err != nil {
											fmt.Println("zouin", err)
											return
										}
									} else {
										fmt.Println("zzzzz", err)
									}
								} else {
									fmt.Println("lkqsdqsdk", err)
								}
							} else {
								fmt.Println("WF ERR", err)
							}
						}
					} else {
						fmt.Println("qsdsq", r, err)
					}
				}

			} else {
				fmt.Println("Err create", err)
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
	reader.Comma = ';'

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
