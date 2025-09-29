package domain

import (
	"errors"
	"fmt"
	"net/url"
	"slices"

	"sqldb-ws/domain/domain_service"
	permissions "sqldb-ws/domain/domain_service/permission"
	schserv "sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	domain "sqldb-ws/domain/specialized_service"
	"sqldb-ws/domain/utils"
	conn "sqldb-ws/infrastructure/connector/db"
	infrastructure "sqldb-ws/infrastructure/service"
	"strings"

	"github.com/google/uuid"
)

/*
		Domain is defined as the DDD patterns will suggest it.
		It's the specialized part of the API, it concive particular behavior on datas (in our cases, particular Root DB declares in entity)
		Main Service at a Domain level, it follows the DOMAIN ITF from schserv.
		Domain interact at a "Model" level with generic and abstract infra services.
		Mai	"fmt"

	  service give the main process to interact with Infra.
*/
var EXCEPTION_FUNC = []string{"Count"}

type SpecializedDomain struct {
	utils.AbstractDomain
	PermsService *permissions.PermDomainService
	Db           *conn.Database
}

// generate a new domain controller
func Domain(superAdmin bool, user string, permsService *permissions.PermDomainService) *SpecializedDomain {
	if permsService == nil {
		permsService = permissions.NewPermDomainService(conn.Open(nil), user, superAdmin, false)
	}
	return &SpecializedDomain{
		AbstractDomain: utils.AbstractDomain{
			SearchInFiles:   map[string]string{},
			DomainRequestID: uuid.New().String(),
			SuperAdmin:      superAdmin, // carry the security level of the "User" or an inner action
			User:            user,       // current user...
		},
		PermsService: permsService, // carry the permissions service
	}
}

func (d *SpecializedDomain) VerifyAuth(tableName string, colName string, level string, method utils.Method, args ...string) bool {
	if len(args) > 0 {
		return d.PermsService.LocalPermsCheck(tableName, colName, level, args[0], d.Own, method, d)
	} else {
		return d.PermsService.PermsCheck(tableName, colName, level, d.Own, method, d)
	}
}

func (d *SpecializedDomain) GetSpecialized(override string) infrastructure.InfraSpecializedServiceItf {
	if override != "" {
		spe := domain.SpecializedService(override)
		spe.SetDomain(d)
		return spe
	}
	return d.SpecializedService
}

func (s *SpecializedDomain) HandleRecordAttributes(record utils.Record) {
	s.Empty = utils.Compare(record["is_empty"], true)
	s.LowerRes = utils.Compare(record["is_list"], true)
	s.Own = utils.Compare(record["own_view"], true)
}
func (d *SpecializedDomain) IsOwn(checkPerm bool, force bool, method utils.Method) bool {
	if checkPerm {
		return d.PermsService.IsOwnPermission(d.TableName, force, d.Own, d) && d.Own
	}
	return d.Own
}

func (d *SpecializedDomain) GetDb() *conn.Database { return d.Db }

func (d *SpecializedDomain) CreateSuperCall(params utils.Params, record utils.Record, args ...interface{}) (utils.Results, error) {
	return d.SuperCall(params, record, utils.CREATE, false, args...) // how to...
}

func (d *SpecializedDomain) UpdateSuperCall(params utils.Params, record utils.Record, args ...interface{}) (utils.Results, error) {
	return d.SuperCall(params, record, utils.UPDATE, false, args...) // how to...
}

func (d *SpecializedDomain) DeleteSuperCall(params utils.Params, args ...interface{}) (utils.Results, error) {
	return d.SuperCall(params, utils.Record{}, utils.DELETE, false, args...) // how to...
}

// Infra func caller with admin view && superadmin right (not a structured view made around data for view reason)
func (d *SpecializedDomain) SuperCall(params utils.Params, record utils.Record, method utils.Method, isOwn bool, args ...interface{}) (utils.Results, error) {
	params.Set(utils.RootRawView, "enable")
	d2 := Domain(true, d.User, d.PermsService)
	d2.DomainRequestID = d.DomainRequestID
	d2.SetAutoload(d.GetAutoload())
	if isOwn {
		d2.Own = d.IsOwn(false, false, method)
	}
	return d2.call(params, record, method, args...)
}

// Infra func caller with current option view and user rights.
func (d *SpecializedDomain) Call(params utils.Params, record utils.Record, method utils.Method, args ...interface{}) (utils.Results, error) {
	return d.call(params, record, method, args...)
}

func (d *SpecializedDomain) onBooleanValue(key string, sup func(bool)) {
	if t, ok := d.Params.Get(key); ok && t == "enable" {
		sup(ok)
	}
}

// Main process to call an Infra function
func (d *SpecializedDomain) call(params utils.Params, record utils.Record, method utils.Method, args ...interface{}) (utils.Results, error) {
	d.Method = method
	d.Params = params
	d.onBooleanValue(utils.RootSuperCall, func(b bool) { d.Super = b })
	d.onBooleanValue(utils.RootShallow, func(b bool) { d.Shallowed = b })
	if tablename, ok := params.Get(utils.RootTableParam); ok { // retrieve tableName in query (not optionnal)
		d.TableName = strings.ToLower(schserv.GetTablename(tablename))
		specializedService := domain.SpecializedService(d.TableName)
		d.SpecializedService = specializedService.SetDomain(d)
		if d.Db == nil || d.Db.Conn == nil {
			d.Db = conn.Open(d.Db)
		}
		defer d.Db.Close()
		if d.GetUserID() == "" && !d.AutoLoad {
			if res, err := d.Db.SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
				"name":  conn.Quote(d.User),
				"email": conn.Quote(d.User),
			}, true); err == nil && len(res) > 0 {
				d.UserID = utils.GetString(res[0], utils.SpecialIDParam)
				fmt.Println("by User: ", d.UserID)
			} else if !d.SuperAdmin {
				return utils.Results{}, errors.New("not authorized : unknown user attempt to reach api")
			}
		}
		if d.Method.IsMath() {
			d.Method = utils.SELECT
		}
		if !d.SuperAdmin && !d.PermsService.PermsCheck(d.TableName, "", "", d.Own, d.Method, d) && !d.AutoLoad && method != utils.DELETE {
			return utils.Results{}, errors.New("not authorized to " + method.String() + " " + d.TableName + " data")
		}
		// load the highest entity avaiable Table level.
		d.Service = infrastructure.NewTableService(d.Db, d.SuperAdmin, d.User, strings.ToLower(d.TableName))
		d.Params.SimpleDelete(utils.RootTableParam)
		if rowName, ok := params.Get(utils.RootRowsParam); ok { // rows override columns
			return d.GetRowResults(rowName, record, specializedService, args...)
		}
		if !d.SuperAdmin || method == utils.DELETE || method == utils.IMPORT {
			return utils.Results{}, errors.New("not authorized to " + method.String() + " " + d.Service.GetName() + " data")
		}
		if col, ok := params.Get(utils.RootColumnsParam); ok && d.TableName != utils.ReservedParam {
			d.Service = infrastructure.NewTableColumnService(d.Db, d.SuperAdmin, d.User, strings.ToLower(d.TableName), d.SpecializedService, strings.ToLower(col))
		} else if d.TableName == utils.ReservedParam {
			return utils.Results{}, errors.New("can't load table as " + utils.ReservedParam)
		}
		return d.Invoke(record, method, args...)
	}
	return utils.Results{}, errors.New("no service available")
}

func (d *SpecializedDomain) GetRowResults(
	rowName string,
	record utils.Record,
	specializedService utils.SpecializedServiceITF,
	args ...interface{},
) (utils.Results, error) {

	rowName, _ = url.QueryUnescape(rowName)
	ids := strings.Split(rowName, ",")
	all_results := utils.Results{}
	for _, id := range ids {
		if id == "" {
			continue
		}
		if record["is_draft"] != nil && !utils.GetBool(record, "is_draft") && d.Method == utils.UPDATE {
			if rr, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(d.TableName, map[string]interface{}{
				utils.SpecialIDParam: id,
				"is_draft":           true,
			}, false); err == nil && len(rr) > 0 {
				d.IsDraftToPublished = true
			}
		}
		d.Params.Add(utils.SpecialIDParam, strings.ToLower(id), func(_ string) bool {
			return strings.ToLower(id) != utils.ReservedParam
		})
		d.Params.SimpleDelete(utils.RootRowsParam)
		if p, _ := d.Params.Get(utils.SpecialIDParam); p == "" || p == utils.ReservedParam {
			d.Params.SimpleDelete(utils.SpecialIDParam)
		} else if record != nil {
			record[utils.SpecialIDParam], _ = d.Params.Get(utils.SpecialIDParam)
		}
		if id, _ := d.Params.Get(utils.SpecialIDParam); d.Method == utils.DELETE && (!d.PermsService.CanDelete(d.Params.Values, record, d) || (id == "" && !d.IsSuperAdmin())) {
			fmt.Println("can't delete datas", id, d.PermsService.CanDelete(d.Params.Values, record, d))
			continue
		}
		d.Service = infrastructure.NewTableRowService(d.Db, d.SuperAdmin, d.User, strings.ToLower(d.TableName), specializedService)
		if d.Method == utils.IMPORT {
			path, err := domain_service.NewUploader(d).ApplyUpload(d.File, d.FileHandler)
			return utils.Results{{"path": path}}, err // only apply on first ID
		} else {
			p, _ := d.Params.Get(utils.RootShallow)
			if p == "enable" {
				if _, ok := d.Params.Get(utils.RootOffset); !ok {
					d.Params.Set(utils.RootLimit, "10")
					d.Params.Set(utils.RootOffset, "0")
				}
			}
			res, err := d.Invoke(record, d.Method, args...)
			if err != nil {
				return all_results, err
			}
			p, _ = d.Params.Get(utils.RootRawView)
			if p != "enable" && err == nil && !d.IsSuperCall() && !slices.Contains(EXCEPTION_FUNC, d.Method.Calling()) {
				res = specializedService.TransformToGenericView(res, d.TableName, d.Params.GetAsArgs(utils.RootDestIDParam)...)
				d.Redirections = append(d.Redirections, d.GetRedirections(res)...)
			}
			all_results = append(all_results, res...)
		}
	}
	return all_results, nil
}

func (d *SpecializedDomain) Invoke(record utils.Record, method utils.Method, args ...interface{}) (utils.Results, error) {
	var err error
	res := []map[string]interface{}{}
	if d.Service == nil {
		return utils.ToResult(res), errors.New("no service available")
	}
	switch method {
	case utils.CREATE:
		res, err = d.Service.Create(record)
	case utils.UPDATE:
		res, err = d.Service.Update(record, utils.ToListStr(args)...)
	case utils.DELETE:
		res, err = d.Service.Delete(utils.ToListStr(args)...)
	case utils.SELECT:
		res, err = d.Service.Get(utils.ToListStr(args)...)
	default:
		if method.IsMath() {
			res, err = d.Service.Math(method.String(), utils.ToListStr(args)...)
		} else {
			err = errors.New("unknow method " + method.Calling())
		}
	}
	return utils.ToResult(res), err
}

func (d *SpecializedDomain) GetRedirections(results utils.Results) []string {
	reds := []string{}
	for _, res := range results {
		if red, ok := res["redirection"]; ok && red != "" {
			reds = append(reds, utils.GetString(res, "redirection"))
		}
	}
	return reds
}
