package controller

import (
	"encoding/json"
	"fmt"
	"sqldb-ws/domain"
	"sqldb-ws/domain/utils"
	"strings"

	beego "github.com/beego/beego/v2/server/web"
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
	if method == utils.IMPORT {
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
