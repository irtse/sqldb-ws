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
		ImportVisibility()
		time.Sleep(24 * time.Hour)
	}
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
		9:  "conference_start_date",
		10: "conference_end_date",
		11: "conference_city",
		12: "conference_country",
		13: "conference_link",
		14: "media_name",
		15: "publishing_date",
		23: "effective_publishing_date",
		25: "authors",                                 // special OK
		26: "affiliation",                             // special OK
		27: "IRT_manager" + ds.RootID(ds.DBUser.Name), // special OK
		28: "i_start_date",
		29: "i_end_date",
		30: "is_awarded", // special OK
		31: "defense_date",
		34: "director_" + ds.RootID(ds.DBUser.Name), // special
		35: "t_start_date",
		36: "t_end_date",
		37: "meeting_name",
		38: "meeting_date",
		39: "manager_" + ds.RootID(ds.DBUser.Name), // special OK
		41: "state",                                // special OK
		42: "active",                               // special OK -> VERIFY is ok
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
		if data[4] != "these" {
			dt = []int{5, 3, 30, 23, 25, 26, 34, 39, 35, 363}
			dbName = models.ThesisFR.Name
			affDbName = models.ThesisAffiliationAuthorsFR.Name
			authorsDbName = models.ThesisAffiliationAuthorsFR.Name
		} else if data[4] != "stage" {
			dt = []int{5, 3, 30, 23, 25, 26, 39, 27, 28, 29}
			dbName = models.InternshipFR.Name
			affDbName = models.InternshipAffiliationAuthorsFR.Name
			authorsDbName = models.InternshipAuthorsFR.Name
		} else if data[4] != "poster" {
			dt = []int{5, 3, 30, 23, 25, 26, 39, 8, 9, 11, 12, 13}
			dbName = models.PosterFR.Name
			affDbName = models.PosterAffiliationAuthorsFR.Name
			authorsDbName = models.PosterAuthorsFR.Name
		} else if data[4] != "demo" {
			dt = []int{5, 3, 30, 23, 25, 26, 39, 37, 38}
			dbName = models.DemoFR.Name
			affDbName = models.DemoAffiliationAuthorsFR.Name
			authorsDbName = models.DemoAuthorsFR.Name
		} else if data[4] != "article_journal" {
			dt = []int{5, 3, 30, 23, 25, 26, 39, 14, 15}
			dbName = models.ArticleFR.Name
			affDbName = models.ArticleAffiliationAuthorsFR.Name
			authorsDbName = models.ArticleAuthorsFR.Name
		} else if data[4] != "communication" {
			// TODO DEFINE IF CONFERENCE OR PRESENTATION
		}
		// TODO FILE RETRIEVAL +
		for _, i := range dt {
			if i == 42 {
				if data[i] == "0" {
					model[mapped[i]] = false
				} else {
					model[mapped[i]] = true
				}
			} else if i == 41 {
				if model["state"] == nil {
					if data[i] == "brouillon" || data[i] == "initie" {
						if st, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.PublicationStatusFR.Name, []interface{}{
							"name::text LIKE '%init%'",
						}, false); err == nil && len(st) > 0 {
							model["state"] = st[0][utils.SpecialIDParam]
						}
					} else if strings.Contains(data[i], "annule") || strings.Contains(data[i], "refuse") {
						if st, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.PublicationStatusFR.Name, []interface{}{
							"name::text LIKE '%ban%'",
						}, false); err == nil && len(st) > 0 {
							model["state"] = st[0][utils.SpecialIDParam]
						}
					} else if strings.Contains(data[i], "final_valide") {
						if st, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.PublicationStatusFR.Name, []interface{}{
							"name::text LIKE '%pub%'",
						}, false); err == nil && len(st) > 0 {
							model["state"] = st[0][utils.SpecialIDParam]
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
				if usr, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
					"code": connector.Quote(data[i]),
				}, false); err == nil && len(usr) > 0 {
					model[mapped[i]] = usr[0][utils.SpecialIDParam]
				}
			} else if i == 27 {
				restr := []interface{}{}
				data[i] = strings.ReplaceAll(data[i], ";", ",")
				for _, auth := range strings.Split(data[i], ",") {
					for _, n := range strings.Split(auth, " ") {
						restr = append(restr, "name::text LIKE '%"+strings.Trim(n, " ")+"%'")
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
						restr = append(restr, "name::text LIKE '%"+strings.Trim(n, " ")+"%'")
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
		if res, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(dbName, map[string]interface{}{
			"name": connector.Quote(utils.GetString(model, "name")),
		}, false); err == nil && len(res) == 0 {
			delete(model, "authors")
			if id, err := d.GetDb().ClearQueryFilter().CreateQuery(dbName, model, func(s string) (string, bool) { return s, true }); err == nil {
				for _, auth := range model["authors"].([]map[string]interface{}) {
					authorss := auth["authors"]
					delete(auth, "authors")
					auth[ds.RootID(dbName)] = id
					if id, err := d.GetDb().ClearQueryFilter().CreateQuery(affDbName, auth, func(s string) (string, bool) { return s, true }); err == nil {
						authorss.(map[string]interface{})[ds.RootID(dbName)] = id
						d.GetDb().ClearQueryFilter().CreateQuery(authorsDbName, authorss.(map[string]interface{}), func(s string) (string, bool) { return s, true })
					}
				}
			}

			// authors
			// aff

		}
	}
}

func ImportVisibility() {
	d := domain.Domain(true, os.Getenv("SUPERADMIN_NAME"), nil)
	filepath := os.Getenv("VISIBILITY_FILE_PATH")
	if filepath == "" {
		filepath = "./visibility_test.csv"
	} else {
		filepath = "/mnt/plugin_files/" + filepath
	}

	_, datas := importFile(filepath)
	for _, data := range datas {
		if res, err := d.GetDb().ClearQueryFilter().ClearQueryFilter().SelectQueryWithRestriction(ds.DBEntityUser.Name, map[string]interface{}{
			ds.EntityDBField: d.GetDb().ClearQueryFilter().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBEntity.Name, map[string]interface{}{
				"name": d.GetDb().ClearQueryFilter().ClearQueryFilter().BuildSelectQueryWithRestriction(models.Project.Name, map[string]interface{}{
					"code": connector.Quote(utils.ToString(data[2])),
				}, false, "name"),
			}, false, utils.SpecialIDParam),
			ds.UserDBField: d.GetDb().ClearQueryFilter().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
				"name": connector.Quote(data[1]),
			}, false, utils.SpecialIDParam),
		}, false); err == nil && len(res) == 0 {
			if res, err := d.GetDb().ClearQueryFilter().ClearQueryFilter().SelectQueryWithRestriction(ds.DBEntity.Name, map[string]interface{}{
				"name": d.GetDb().ClearQueryFilter().ClearQueryFilter().BuildSelectQueryWithRestriction(models.Project.Name, map[string]interface{}{
					"code": connector.Quote(utils.ToString(data[2])),
				}, false, "name"),
			}, false); err == nil && len(res) > 0 {
				if usr, err := d.GetDb().ClearQueryFilter().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
					"name": connector.Quote(data[1]),
				}, false); err == nil && len(usr) > 0 {
					d.GetDb().ClearQueryFilter().CreateQuery(ds.DBEntityUser.Name, map[string]interface{}{
						ds.UserDBField:   usr[0][utils.SpecialIDParam],
						ds.EntityDBField: res[0][utils.SpecialIDParam],
					}, func(s string) (string, bool) { return "", true })
				}
			}
		}
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
			if res, err := d.GetDb().ClearQueryFilter().ClearQueryFilter().SelectQueryWithRestriction(models.Axis.Name, map[string]interface{}{
				"code": connector.Quote(utils.GetString(record, "code")),
			}, false); err == nil && len(res) > 0 {
				record[utils.SpecialIDParam] = res[0][utils.SpecialIDParam]
				d.GetDb().ClearQueryFilter().UpdateQuery(models.Axis.Name, record, map[string]interface{}{
					utils.SpecialIDParam: res[0][utils.SpecialIDParam],
				}, false)
				continue
			}

			res, err := d.GetDb().ClearQueryFilter().CreateQuery(ds.DBEntity.Name, map[string]interface{}{
				"name": record["name"],
			}, func(s string) (string, bool) { return "", true })
			if err == nil {
				record[ds.EntityDBField] = res
				d.GetDb().ClearQueryFilter().CreateQuery(models.Axis.Name, record, func(s string) (string, bool) { return "", true })
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
					record["prj_start_date"] = t.Format("2006-01-02 15:04:05")
				}
			}
			if i == 9 && data[i] != "" {
				t, err := time.Parse("02/01/2006", fmt.Sprintf("%v", data[i]))
				if err == nil {
					record["prj_end_date"] = t.Format("2006-01-02 15:04:05")
				}
			}
			if i == 4 && data[i] != "" {
				if res, err := d.GetDb().ClearQueryFilter().ClearQueryFilter().SelectQueryWithRestriction(models.Axis.Name, map[string]interface{}{
					"code": connector.Quote(data[i]),
				}, false); err == nil && len(res) > 0 {
					record[ds.RootID(models.Axis.Name)] = res[0][utils.SpecialIDParam]
				}
			}
			if i == 12 && data[i] != "" {
				if res, err := d.GetDb().ClearQueryFilter().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
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
				if res, err := d.GetDb().ClearQueryFilter().ClearQueryFilter().SelectQueryWithRestriction(ds.DBEntity.Name, map[string]interface{}{
					"name": connector.Quote(axisName),
				}, false); err == nil && len(res) > 0 {
					i := utils.GetInt(res[0], utils.SpecialIDParam)
					parentID = &i
				}
			}
			if res, err := d.GetDb().ClearQueryFilter().ClearQueryFilter().SelectQueryWithRestriction(models.Project.Name, map[string]interface{}{
				"code": connector.Quote(utils.GetString(record, "code")),
			}, false); err == nil && len(res) > 0 {

				record[utils.SpecialIDParam] = res[0][utils.SpecialIDParam]
				d.GetDb().ClearQueryFilter().UpdateQuery(models.Project.Name, record, map[string]interface{}{
					utils.SpecialIDParam: res[0][utils.SpecialIDParam],
				}, false)
				m := map[string]interface{}{
					ds.UserDBField: respPrj,
				}
				m2 := map[string]interface{}{
					ds.UserDBField: respPrj,
				}
				if respPrj >= 0 { // add a CDP to a project
					if res, err := d.GetDb().ClearQueryFilter().ClearQueryFilter().SelectQueryWithRestriction(ds.DBEntity.Name, map[string]interface{}{
						"name": connector.Quote(utils.ToString(res[0]["name"])),
					}, false); err == nil && len(res) > 0 {
						m[ds.EntityDBField] = res[0][utils.SpecialIDParam]
					}
					d.GetDb().ClearQueryFilter().DeleteQueryWithRestriction(ds.DBEntityUser.Name, m, false)
					d.GetDb().ClearQueryFilter().CreateQuery(ds.DBEntityUser.Name, m, func(s string) (string, bool) { return "", true })
					if res, err := d.GetDb().ClearQueryFilter().ClearQueryFilter().SelectQueryWithRestriction(ds.DBEntity.Name, map[string]interface{}{
						"name": "'CDP'",
					}, false); err == nil && len(res) > 0 {
						m2[ds.EntityDBField] = res[0][utils.SpecialIDParam]
					}
					d.GetDb().ClearQueryFilter().DeleteQueryWithRestriction(ds.DBEntityUser.Name, m2, false)
					d.GetDb().ClearQueryFilter().CreateQuery(ds.DBEntityUser.Name, m2, func(s string) (string, bool) { return "", true })
				}
				continue
			}
			if parentID != nil {
				res, err := d.GetDb().ClearQueryFilter().CreateQuery(ds.DBEntity.Name, map[string]interface{}{
					"name":      record["name"],
					"parent_id": parentID,
				}, func(s string) (string, bool) { return "", true })
				if err == nil {
					record[ds.EntityDBField] = res

					d.GetDb().ClearQueryFilter().CreateQuery(models.Project.Name, record, func(s string) (string, bool) { return "", true })
				}
			} else {
				res, err := d.GetDb().ClearQueryFilter().CreateQuery(ds.DBEntity.Name, map[string]interface{}{
					"name": record["name"],
				}, func(s string) (string, bool) { return "", true })
				if err == nil {
					record[ds.EntityDBField] = res

					d.GetDb().ClearQueryFilter().CreateQuery(models.Project.Name, record, func(s string) (string, bool) { return "", true })
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
	d.GetDb().ClearQueryFilter().CreateQuery(ds.DBEntity.Name, map[string]interface{}{
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

						res, err := d.GetDb().ClearQueryFilter().CreateQuery(ds.DBEntity.Name, map[string]interface{}{
							"name": cocName,
						}, func(s string) (string, bool) { return "", true })
						if err == nil {
							d.GetDb().ClearQueryFilter().CreateQuery(models.CoCFR.Name, map[string]interface{}{
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
				ds.UserDBField: d.GetDb().ClearQueryFilter().ClearQueryFilter().BuildSelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
					"name": connector.Quote(utils.ToString(record["name"])),
				}, false, "id"),
				ds.EntityDBField: d.GetDb().ClearQueryFilter().ClearQueryFilter().BuildSelectQueryWithRestriction(
					ds.DBEntity.Name, map[string]interface{}{
						"name": cocName,
					}, false, "id"),
			}
			if cocName != "" { // add a CDP to a project
				if res, err := d.GetDb().ClearQueryFilter().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
					"name": connector.Quote(utils.ToString(record["name"])),
				}, false); err == nil && len(res) > 0 {
					m[ds.UserDBField] = res[0][utils.SpecialIDParam]
				}
				if res, err := d.GetDb().ClearQueryFilter().ClearQueryFilter().SelectQueryWithRestriction(ds.DBEntity.Name, map[string]interface{}{
					"name": connector.Quote(cocName),
				}, false); err == nil && len(res) > 0 {
					m[ds.EntityDBField] = res[0][utils.SpecialIDParam]
					d.GetDb().ClearQueryFilter().DeleteQueryWithRestriction(ds.DBEntityUser.Name, m, false)
					d.GetDb().ClearQueryFilter().CreateQuery(ds.DBEntityUser.Name, m, func(s string) (string, bool) { return "", true })
				}
			}
			if !slices.Contains(inside, utils.GetString(record, "name")) {
				inside = append(inside, utils.GetString(record, "name"))
				if utils.GetBool(record, "active") {
					if res, err := d.GetDb().ClearQueryFilter().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
						"name": connector.Quote(utils.GetString(record, "name")),
					}, false); err == nil && len(res) > 0 {
						record[utils.SpecialIDParam] = res[0][utils.SpecialIDParam]
						d.GetDb().ClearQueryFilter().UpdateQuery(ds.DBUser.Name, record, map[string]interface{}{
							utils.SpecialIDParam: res[0][utils.SpecialIDParam],
						}, false)
					} else {
						d.GetDb().ClearQueryFilter().CreateQuery(ds.DBUser.Name, record, func(s string) (string, bool) { return "", true })
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
			if i == 11 && data[i] != "" {
				if res, err := d.Db.ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
					"code": connector.Quote(data[i]),
				}, false); err == nil && len(res) > 0 {
					hierarchyID = utils.GetString(res[0], utils.SpecialIDParam)
				}
			}
		}
		if userID != "" && hierarchyID != "" && hierarchyID != userID {
			d.GetDb().ClearQueryFilter().DeleteQueryWithRestriction(ds.DBHierarchy.Name, map[string]interface{}{
				ds.UserDBField: userID,
			}, false)
			d.GetDb().ClearQueryFilter().CreateQuery(ds.DBHierarchy.Name, map[string]interface{}{
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
