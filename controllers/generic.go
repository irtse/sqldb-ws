package controllers

import (
	"fmt"
	"net/http"
	"os"
	"sqldb-ws/controllers/controller"
	ds "sqldb-ws/domain/schema/database_resources"
	"sqldb-ws/domain/utils"
	connector "sqldb-ws/infrastructure/connector/db"
	"strings"
)

type MainController struct{ controller.AbstractController }

// Operations about table
type GenericController struct{ controller.AbstractController }

// @Title /
// @Description Main call
// @Param	body		body 	Credential	true		"Credentials"
// @Success 200 {string} success !
// @Failure 403 user does not exist
// @router / [get]
func (l *MainController) Main() {
	connector.COUNTREQUEST = 0
	// Main is the default root of the API, it gives back all your allowed shallowed view
	l.ParamsOverload = utils.AllParams(ds.DBView.Name).RootShallow().Enrich(
		map[string]interface{}{
			"indexable": true,
		}).Values
	l.SafeCall(utils.SELECT)
}

// @Title /
// @Description Download call
// @Param	path		path 	string	true		"Name of the table"
// @Success 200 {string} success !
// @Failure 403 user does not exist
// @router /download/:path [get]
func (l *MainController) Download() {
	// Get the filename from the query string
	filePath := l.GetString(":path")
	if !strings.Contains(filePath, "/mnt/files/") {
		filePath = "/mnt/files/" + filePath
	}
	fmt.Println(fmt.Sprintf("%v.gz", strings.Trim(filePath, " ")))
	if _, err := os.Stat(filePath); err == nil {
		fmt.Printf("File exists\n")
	} else {
		fmt.Printf("File does not exist\n")
	}
	uncompressedP, err := l.UncompressGzip(filePath)
	fmt.Println(uncompressedP)
	if err != nil {
		l.Ctx.Output.SetStatus(http.StatusNoContent)
		l.Ctx.Output.Body([]byte(err.Error()))
		return
	}
	// Open the file
	file, err := os.Open(uncompressedP)

	if err != nil {
		l.Ctx.Output.SetStatus(http.StatusNotFound)
		l.Ctx.Output.Body([]byte("File not found"))
		return
	}
	defer file.Close()

	// Get file info
	stat, err := file.Stat()
	if err != nil {
		l.Ctx.Output.SetStatus(http.StatusInternalServerError)
		l.Ctx.Output.Body([]byte("Unable to get file info"))
		return
	}

	// Set headers for file download
	l.Ctx.Output.Header("Content-Disposition", "attachment; filename="+stat.Name())
	l.Ctx.Output.Header("Content-Type", "application/octet-stream")
	l.Ctx.Output.Header("Content-Length", string(rune(stat.Size())))
	l.DeleteUncompressed(uncompressedP)
	// Serve the file
	http.ServeFile(l.Ctx.ResponseWriter, l.Ctx.Request, filePath)
}

// @Title Post data in table
// @Description post data in table
// @Param	table		path 	string	true		"Name of the table"
// @Param	data		body 	json	true		"body for data content (Json format)"
// @Success 200 {string} success
// @Failure 403 :table post issue
// @router /:table [post]
func (t *GenericController) Post() { t.SafeCall(utils.CREATE) }

// @Title Put data in table
// @Description put data in table
// @Param	table		path 	string	true		"Name of the table"
// @Param	data		body 	json	true		"body for data content (Json format)"
// @Success 200 {string} success
// @Failure 403 :table put issue
// @router /:table [put]
func (t *GenericController) Put() { t.SafeCall(utils.UPDATE) }

// web.InsertFilter("/*", web.BeforeRouter, FilterUserPost)
// }

// @Title Delete
// @Description delete the data in table
// @Param	table		path 	string	true		"Name of the table"
// @Param	body		body 			true		"body for data content (Json format)"
// @Success 200 {string} delete success!
// @Failure 403 delete issue
// @router /:table [delete]
func (t *GenericController) Delete() { t.SafeCall(utils.DELETE) }

// @Title Get
// @Description get Datas
// @Param	table			path 	string	true		"Name of the table"
// @Success 200 {string} success !
// @Failure 403 no table
// @router /:table [get]
func (t *GenericController) Get() { t.SafeCall(utils.SELECT) }

// @Title Math
// @Description math on Datas
// @Param	table			path 	string	true		"Name of the table"
// @Success 200 {string} success !
// @Failure 403 no table
// @router /:table/:function [get]
func (t *GenericController) Math() {
	function := t.Ctx.Input.Params()[":function"]
	t.SafeCall(utils.Found(function))
}

// @Title Import
// @Description import Datas
// @Param	table			path 	string	true		"Import in columnName"
// @Success 200 {string} success !
// @Failure 403 no table
// @router /:table/import [post]
func (t *GenericController) Import() { t.SafeCall(utils.IMPORT) }
