package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sqldb-ws/domain"
	"sqldb-ws/domain/utils"
	"strings"
	"sync"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
	"github.com/gorilla/websocket"
	"github.com/matthewhartstonge/argon2"
	"github.com/rs/zerolog/log"
)

/*
AbstractController defines main procedure that a generic Handler would get.
*/
var JSON = "json"
var DATA = "data"
var ERROR = "error"

// Operations about table
type AbstractController struct {
	ParamsOverload map[string]string
	beego.Controller
}

// SafeCaller will ask for a authenticated procedure
func (t *AbstractController) SafeCall(method utils.Method, args ...interface{}) {
	t.Call(true, method, args...)
}

// SafeCaller will ask for a free of authentication procedure
func (t *AbstractController) UnSafeCall(method utils.Method, args ...interface{}) {
	t.Call(false, method, args...)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // allow all origins
}

// user/tableName/row
var clients = map[string]map[string]map[string]map[*websocket.Conn]bool{}
var clientsLock = sync.Mutex{}

func (t *AbstractController) WebSocketController(w *context.Response, r *http.Request, params utils.Params, user string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		t.Response(utils.Results{}, err, "", "")
		return
	}
	defer conn.Close()
	ok := false
	var tableName, rowName string = "", ""
	if tableName, ok = params.Get(utils.RootRowsParam); !ok {
		t.Response(utils.Results{}, errors.New("can't found table name"), "", "")
		return
	}
	if rowName, ok = params.Get(utils.RootRowsParam); !ok {
		t.Response(utils.Results{}, errors.New("can't found row name"), "", "")
		return
	}
	// register client
	clientsLock.Lock()
	if clients[user] == nil {
		clients[user] = map[string]map[string]map[*websocket.Conn]bool{}
	}
	if clients[user][tableName] == nil {
		clients[user][tableName] = map[string]map[*websocket.Conn]bool{}
	}
	if clients[user][tableName][rowName] == nil {
		clients[user][tableName][rowName] = map[*websocket.Conn]bool{}
	}
	clients[user][tableName][rowName][conn] = true
	clientsLock.Unlock()
	for {
		// Read message from client
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			t.Response(utils.Results{}, err, "", "")
			break
		}
		// Echo the message back
		if err := conn.WriteMessage(msgType, msg); err != nil {
			t.Response(utils.Results{}, err, "", "")
			break
		}
	}
	clientsLock.Lock()
	delete(clients[user][tableName][rowName], conn)
	clientsLock.Unlock()
	t.Response(utils.Results{}, nil, "", "")
}

func (t *AbstractController) WebsocketTrigger(user string, params utils.Params, domain utils.DomainITF, args ...interface{}) {
	// serialize the map to JSON
	resp, err := domain.Call(params, utils.Record{}, utils.SELECT, args...)
	if err != nil {
		fmt.Println("Callerror:", err)
		return
	}
	ok := false
	var tableName, rowName string = "", ""
	if tableName, ok = params.Get(utils.RootRowsParam); !ok {
		t.Response(utils.Results{}, errors.New("can't found table name"), "", "")
		return
	}
	if rowName, ok = params.Get(utils.RootRowsParam); !ok {
		t.Response(utils.Results{}, errors.New("can't found row name"), "", "")
		return
	}
	msgBytes, err := json.Marshal(map[string]interface{}{
		"data":  resp,
		"event": domain.GetMethod().String(),
	})
	if err != nil {
		fmt.Println("JSON marshal error:", err)
		return
	}

	clientsLock.Lock()
	defer clientsLock.Unlock()
	if clients[user] == nil || clients[user][tableName] == nil || clients[user][tableName][rowName] == nil {
		return
	}

	for conn := range clients[user][tableName][rowName] {
		if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
			fmt.Println("Write error, removing client:", err)
			conn.Close()
			delete(clients[user][tableName][rowName], conn)
		}
	}
}

// Call function invoke Domain service and ask for the proper function by function name & method
func (t *AbstractController) Call(auth bool, method utils.Method, args ...interface{}) {
	superAdmin := false
	var user string
	var err error
	if auth { // we will verify authentication and status if auth is expected
		if user, superAdmin, err = t.IsAuthorized(); err != nil {
			t.Response(utils.Results{}, err, "", "")
			return
		}
	} // then proceed to exec by calling domain
	d := domain.Domain(superAdmin, user, nil)
	p, asLabel := t.Params()
	if method == utils.WEBSOCKET {
		t.WebSocketController(t.Ctx.ResponseWriter, t.Ctx.Request, utils.NewParams(p), user)
	} else if method == utils.IMPORT {
		file, header, err := t.Ctx.Request.FormFile("file")
		if err != nil {
			t.Response(utils.Results{}, err, "", d.GetUniqueRedirection())
			return
		}
		d.SetFile(file, header)
		t.Respond(p, asLabel, method, d, args...)
	} else if files, err := t.FormFile(asLabel); err == nil && len(files) > 0 {
		resp := utils.Results{}
		var error error
		for _, file := range files {
			response, err := d.Call(utils.NewParams(p), file, method, args...)
			if method != utils.SELECT {
				t.WebsocketTrigger(user, utils.NewParams(p), d, args...)
			}
			if err != nil {
				error = fmt.Errorf("%v|%v", error, err)
			} else {
				resp = append(resp, response...)
			}
		}
		t.Response(resp, error, "", d.GetUniqueRedirection()) // send back response
	} else {
		t.Respond(p, asLabel, method, d, args...)
	}
}

// params will produce a Params struct compose of url & query parameters
func (t *AbstractController) Params() (map[string]string, map[string]string) {
	paramsAsLabel := map[string]string{}
	if t.ParamsOverload != nil {
		return t.ParamsOverload, paramsAsLabel
	}
	params := map[string]string{}
	rawParams := t.Ctx.Input.Params() // extract all params from url and fill params
	for key, val := range rawParams {
		if strings.Contains(key, ":") && !strings.Contains(key, "splat") {
			params[key[1:]] = val
		}
	}
	path := strings.Split(t.Ctx.Input.URI(), "?")
	if len(path) >= 2 {
		uri := strings.Split(path[1], "&")
		for _, val := range uri {
			kv := strings.Split(val, "=")
			if strings.Contains(kv[0], "_aslabel") && len(kv) > 1 {
				paramsAsLabel[kv[0]] = kv[1]
			} else if len(kv) > 1 {
				params[kv[0]] = kv[1]
			}
		}
	}
	if pass, ok := params["password"]; ok { // if any password founded hash it
		argon := argon2.DefaultConfig()
		hash, err := argon.HashEncoded([]byte(pass))
		if err != nil {
			log.Error().Msg(err.Error())
		}
		params["password"] = string(hash)
	}
	if _, ok := params[utils.RootExport]; ok {
		params[utils.RootRawView] = ""
		if _, ok := params[utils.RootFilename]; !ok {
			params[utils.RootFilename] = params[utils.RootTableParam]
		}
	}
	return params, paramsAsLabel
}

// body is the main body extracter from the controller
func (t *AbstractController) Body(hashed bool) utils.Record {
	var res utils.Record
	json.Unmarshal(t.Ctx.Input.RequestBody, &res)
	if pass, ok := res["password"]; ok { // if any password founded hash it
		argon := argon2.DefaultConfig()
		hash, err := argon.HashEncoded([]byte(utils.ToString(pass)))
		if err != nil {
			log.Error().Msg(err.Error())
		}
		if hashed {
			res["password"] = string(hash)
		}
	}
	return res
}
