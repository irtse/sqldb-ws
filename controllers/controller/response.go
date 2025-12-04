package controller

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"net/url"
	"sqldb-ws/domain/utils"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/tealeg/xlsx"
)

func (t *AbstractController) Respond(user string, params map[string]string, asLabel map[string]string, method utils.Method, domain utils.DomainITF, args ...interface{}) {
	if _, ok := params[utils.RootExport]; ok {
		params[utils.RootRawView] = "disable"
	}
	response, err := domain.Call(utils.NewParams(params), t.Body(true), method, args...)
	if method != utils.SELECT {
		t.WebsocketTrigger(user, utils.NewParams(params), domain, args...)
	}
	if format, ok := params[utils.RootExport]; ok {
		var cols, cmd, cmdCols string = "", "", ""
		if pp, ok := params[utils.RootColumnsParam]; ok {
			cols = pp
		}
		if pp, ok := params[utils.RootCommandRow]; ok {
			cmd = pp
		}
		if pp, ok := params[utils.RootCommandCols]; ok {
			cmdCols = pp
		}
		t.download(domain, cols, cmdCols, cmd, format, params[utils.RootFilename], asLabel, response, err)
		return
	}
	t.Response(response, err, "", domain.GetUniqueRedirection()) // send back response
}

// response rules every http response
func (t *AbstractController) Response(resp utils.Results, err error, format string, redirection string) {
	t.Ctx.Output.SetStatus(http.StatusOK) // defaulting on absolute success
	if err != nil {                       // Check nature of error if there is one
		//if strings.Contains(err.Error(), "AUTH") { t.Ctx.Output.SetStatus(http.StatusUnauthorized) }
		if strings.Contains(err.Error(), "partial") {
			t.Ctx.Output.SetStatus(http.StatusPartialContent)
			t.Data[JSON] = map[string]interface{}{DATA: resp, ERROR: err.Error()}
		} else {
			t.Data[JSON] = map[string]interface{}{DATA: utils.Results{}, ERROR: err.Error()}
		}
		if format == "json" {
			t.Data[JSON] = map[string]interface{}{}
		}
	} else { // if success precise an error if no datas is founded
		t.Data[JSON] = map[string]interface{}{DATA: resp, ERROR: nil}
		for _, json := range utils.ToMap(t.Data[JSON])[DATA].(utils.Results) {
			delete(json, "password") // never send back a password in any manner
		}
		if format == "json" {
			t.Data[JSON] = resp
		}
	}
	if redirection != "" {
		t.Ctx.Output.SetStatus(302)
		t.Ctx.Output.Header("Location", redirection)
	}
	t.ServeJSON() // then serve response by beego
}

func (t *AbstractController) download(d utils.DomainITF, col string, colsCmd string, cmd string, format string, name string, mapping map[string]string, resp utils.Results, error error) {
	cols, lastLine, results := t.mapping(col, colsCmd, cmd, mapping, resp) // mapping
	t.Ctx.ResponseWriter.Header().Set("Content-Type", "text/"+format)
	t.Ctx.ResponseWriter.Header().Set("Content-Disposition", "attachment; filename="+name+"_"+strings.Replace(time.Now().Format(time.RFC3339), " ", "_", -1)+"."+format)
	switch format {
	case "csv":
		t.Ctx.ResponseWriter.Write([]byte{0xEF, 0xBB, 0xBF})
		w := csv.NewWriter(t.Ctx.ResponseWriter)
		w.Comma = ';'
		w.WriteAll(t.csv(d, lastLine, mapping, cols, results))
	case "json":
		t.json(d, lastLine, mapping, cols, results)
	case "pdf":
		t.pdf(d, lastLine, mapping, cols, results)
	case "xlsx":
		t.xlsx(d, lastLine, mapping, cols, results)
	default:
		t.Response(results, error, format, d.GetUniqueRedirection())
	}
}

func (t *AbstractController) json(d utils.DomainITF, colsFunc map[string]string, mapping map[string]string, cols []string, results utils.Results) {
	for _, r := range results {
		for _, c := range cols {
			if v, ok := mapping[c+"_aslabel"]; ok && v != "" {
				r[v] = r[c+"_aslabel"]
				delete(r, c)
			}
			if v, ok := colsFunc[c]; ok && v != "" {
				res, err := d.GetDb().QueryAssociativeArray("SELECT " + v + " as result FROM " + d.GetTable() + " WHERE " + d.GetDb().ClearQueryFilter().SQLRestriction)
				if err == nil && len(res) > 0 {
					splitted := strings.Split(v, "(")
					r["results"] = splitted[0] + ": " + utils.GetString(res[0], "result")
				}
			} else {
				r["results"] = ""
			}
		}
	}
	t.Response(results, nil, "json", d.GetUniqueRedirection())
}

func (t *AbstractController) csv(d utils.DomainITF, colsFunc map[string]string, mapping map[string]string, cols []string, results utils.Results) [][]string {
	var data [][]string
	lastLine, labs := []string{}, []string{}
	for _, c := range cols {
		if v, ok := mapping[c+"_aslabel"]; ok && v != "" {
			decoded, err := url.QueryUnescape(v)
			if err == nil {
				labs = append(labs, decoded)
			} else {
				labs = append(labs, c)
			}
		} else {
			labs = append(labs, c)
		}
	}
	data = append(data, labs)
	for _, c := range cols {
		if v, ok := colsFunc[c]; ok && v != "" {
			r, err := d.GetDb().QueryAssociativeArray("SELECT " + v + " as result FROM " + d.GetTable() + " WHERE " + d.GetDb().ClearQueryFilter().SQLRestriction)
			if err == nil && len(r) > 0 {
				splitted := strings.Split(v, "(")
				lastLine = append(lastLine, splitted[0]+": "+utils.GetString(r[0], "result"))
			}
		} else {
			lastLine = append(lastLine, "")
		}
	}
	for _, r := range results {
		var row []string
		for _, c := range cols {
			if v, ok := r[c]; !ok || v == nil {
				row = append(row, "")
				continue
			}
			v := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(
				utils.ToString(r[c]), "(completed)", ""), "(pending)", ""), "(refused)", ""), "(dismiss)", ""), "(running)", "")
			row = append(row, v)
			if v == "true" {
				v = "yes"
			} else if v == "false" {
				v = "no"
			}
		}
		data = append(data, row)
	}
	data = append(data, lastLine)
	return data
}

func (t *AbstractController) pdf(d utils.DomainITF, colsFunc, mapping map[string]string, cols []string, results utils.Results) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 12)

	// Header
	for _, c := range cols {
		if v, ok := mapping[c+"_aslabel"]; ok && v != "" {
			decoded, err := url.QueryUnescape(v)
			if err == nil {
				pdf.CellFormat(40, 10, decoded, "1", 0, "C", false, 0, "")
			} else {
				pdf.CellFormat(40, 10, c, "1", 0, "C", false, 0, "")
			}
		} else {
			pdf.CellFormat(40, 10, c, "1", 0, "C", false, 0, "")
		}
	}
	pdf.Ln(-1)

	// Data
	for _, r := range results {
		for _, c := range cols {
			v := utils.ToString(r[c])
			if v == "true" {
				v = "yes"
			} else if v == "false" {
				v = "no"
			}
			pdf.CellFormat(40, 10, v, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}

	t.Ctx.ResponseWriter.Header().Set("Content-Type", "application/pdf")
	t.Ctx.ResponseWriter.Header().Set("Content-Disposition",
		"attachment; filename="+time.Now().Format("2006-01-02_15-04-05")+".pdf")
	err := pdf.Output(t.Ctx.ResponseWriter)
	if err != nil {
		fmt.Println(err)
	}
}

func (t *AbstractController) xlsx(d utils.DomainITF, colsFunc, mapping map[string]string, cols []string, results utils.Results) {
	file := xlsx.NewFile()
	sheet, _ := file.AddSheet("Sheet1")

	// Header
	headerRow := sheet.AddRow()
	for _, c := range cols {
		var label string
		if v, ok := mapping[c+"_aslabel"]; ok && v != "" {
			decoded, err := url.QueryUnescape(v)
			if err == nil {
				label = decoded
			} else {
				label = c
			}
		} else {
			label = c
		}
		headerRow.AddCell().Value = label
	}

	// Data
	for _, r := range results {
		row := sheet.AddRow()
		for _, c := range cols {
			v := utils.ToString(r[c])
			if v == "true" {
				v = "yes"
			} else if v == "false" {
				v = "no"
			}
			row.AddCell().Value = v
		}
	}

	t.Ctx.ResponseWriter.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	t.Ctx.ResponseWriter.Header().Set("Content-Disposition",
		"attachment; filename="+time.Now().Format("2006-01-02_15-04-05")+".xlsx")

	err := file.Write(t.Ctx.ResponseWriter)
	if err != nil {
		fmt.Println(err)
	}
}

func (t *AbstractController) mapping(col string, colsCmd string, cmd string, mapping map[string]string, resp utils.Results) ([]string, map[string]string, utils.Results) {
	cols, colsFunc := []string{}, map[string]string{}
	results := utils.Results{}
	if len(resp) == 0 {
		return cols, colsFunc, results
	}
	r := resp[0]
	additionnalCol := ""
	order := []interface{}{"id"}
	order = append(order, utils.ToList(r["order"])...)
	if cmd != "" {
		decodedLine, _ := url.QueryUnescape(cmd)
		re := strings.Split(decodedLine, " as ")
		if len(re) > 1 {
			additionnalCol = re[len(re)-1]
			order = append(order, additionnalCol)
			colsFunc[additionnalCol] = re[0]
		}
	}

	for _, c := range strings.Split(colsCmd, ",") {
		re := strings.Split(c, ":")
		if len(re) > 1 {
			if v, ok := colsFunc[re[0]]; ok && v != "" {
				colsFunc[re[0]] = strings.ToUpper(re[1]) + "((" + v + "))"
			} else {
				colsFunc[re[0]] = re[1]
			}

		}
	}
	schema := utils.ToMap(r["schema"])
	for _, o := range order {
		key := utils.ToString(o)
		if col != "" && !strings.Contains(col, key) && !(additionnalCol == "" || strings.Contains(additionnalCol, key)) {
			continue
		}
		if scheme, ok := schema[key]; ok && strings.Contains(utils.ToString(utils.ToMap(scheme)["type"]), "many") {
			continue
		}
		label := key
		/*if scheme, ok := schema[key]; ok {
			label = strings.Replace(utils.ToString(utils.ToMap(scheme)["label"]), "_", " ", -1)
		}*/
		if mapKey, ok := mapping[key]; ok && mapKey != "" {
			label = mapKey
		}
		cols = append(cols, label)
	}
	for _, item := range utils.ToList(r["items"]) {
		record := utils.Record{}
		for _, o := range order {
			key := utils.ToString(o)
			it := utils.ToMap(item)
			if scheme, ok := schema[key]; ok && key != "id" && strings.Contains(
				utils.ToString(utils.ToMap(scheme)["type"]), "many") {
				continue
			}
			label := key
			/*if scheme, ok := schema[key]; ok {
				label = strings.Replace(utils.ToString(utils.ToMap(scheme)["label"]), "_", " ", -1)
			}*/
			if mapKey, ok := mapping[key]; ok && mapKey != "" {
				label = mapKey
			}
			if v, ok := utils.ToMap(it["values_shallow"])[key]; ok {
				record[label] = utils.ToString(utils.ToMap(v)["name"])
			} else if v, ok := utils.ToMap(it["values"])[key]; ok && v != nil {
				record[label] = utils.ToString(v)
			} else {
				record[label] = ""
			}
			colsFunc[label] = colsFunc[key]
		}
		results = append(results, record)
	}
	return cols, colsFunc, results
}
