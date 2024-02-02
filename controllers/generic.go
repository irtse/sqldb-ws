package controllers

import (
	"fmt"
	tool "sqldb-ws/lib"
	domain "sqldb-ws/lib/domain"
	"sqldb-ws/lib/entities"
)

type MainController struct { AbstractController }
// Operations about table
type GenericController struct { AbstractController }

// @Title /
// @Description Main call
// @Param	body		body 	Credential	true		"Credentials"
// @Success 200 {string} success !
// @Failure 403 user does not exist
// @router / [get]
func (l *MainController) Main() {
	// Main is the default root of the API, it gives back all your allowed shallowed view
	user_id, _, err := l.authorized() // check up if allowed to communicate with API
	if err == nil { // if authorized then ask for shallow view authorized for the user
		params := l.paramsOver(map[string]string{ tool.RootTableParam : entities.DBView.Name, 
												  tool.RootRowsParam : tool.ReservedParam,
												  "indexable" : fmt.Sprintf("%v", true),
												  tool.RootShallow : "enable" })
		d := domain.Domain(false, user_id, false) // load a domain with user perms
		response, err := d.Call(params, tool.Record{}, tool.SELECT, true, "Get") // then call with auth activated
		if err != nil {
			l.response(response, err); return
		}
		l.response(response, nil)
	}
	l.response(nil, err) // TODO can be replace by a custom view (login view ???)
}
// @Title Post data in table
// @Description post data in table
// @Param	table		path 	string	true		"Name of the table"
// @Param	data		body 	json	true		"body for data content (Json format)"
// @Success 200 {string} success
// @Failure 403 :table post issue
// @router /:table [post]
func (t *GenericController) Post() { t.SafeCall(tool.CREATE, "CreateOrUpdate") }
// @Title Put data in table
// @Description put data in table
// @Param	table		path 	string	true		"Name of the table"
// @Param	data		body 	json	true		"body for data content (Json format)"
// @Success 200 {string} success
// @Failure 403 :table put issue
// @router /:table [put]
func (t *GenericController) Put() { t.SafeCall(tool.UPDATE, "CreateOrUpdate") }
// web.InsertFilter("/*", web.BeforeRouter, FilterUserPost)
// }

// @Title Delete
// @Description delete the data in table
// @Param	table		path 	string	true		"Name of the table"
// @Param	body		body 			true		"body for data content (Json format)"
// @Success 200 {string} delete success!
// @Failure 403 delete issue
// @router /:table [delete]
func (t *GenericController) Delete() { t.SafeCall(tool.DELETE, "Delete") }

// @Title Get
// @Description get Datas
// @Param	table			path 	string	true		"Name of the table"
// @Success 200 {string} success !
// @Failure 403 no table
// @router /:table [get]
func (t *GenericController) Get() { t.SafeCall(tool.SELECT, "Get") }

func (t *GenericController) Link() { t.SafeCall(tool.CREATE, "Link") }

func (t *GenericController) UnLink() { t.SafeCall(tool.DELETE, "UnLink") }
// @Title Import
// @Description post Import
// @Param	table			path 	string	true		"Name of the table"
// @Success 200 {string} success !
// @Failure 403 no table
// @router /:table/import/:filename [post]
func (t *GenericController) Importated() { t.SafeCall(tool.CREATE, "Import", t.GetString(":filename")) }
// @Title Delete by Import
// @Description delete Import
// @Param	table path 	string true "Name of the table"
// @Success 200 {string} success !
// @Failure 403 no table
// @router /:table/import/:filename [delete]
func (t *GenericController) NotImportated() { t.SafeCall(tool.DELETE, "Import", t.GetString(":filename")) }