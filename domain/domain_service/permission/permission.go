package permission

import (
	"encoding/json"
	"slices"
	schserv "sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	sm "sqldb-ws/domain/schema/models"
	"sqldb-ws/domain/utils"
	conn "sqldb-ws/infrastructure/connector/db"
	"strings"
	"sync"
)

// DONE - ~ 240 LINES - PARTIALLY TESTED
type Perms struct {
	Read   string `json:"read"`
	Create bool   `json:"write"`
	Update bool   `json:"update"`
	Delete bool   `json:"delete"`
}

type PermDomainService struct {
	mutexPerms   sync.RWMutex
	Perms        map[string]map[string]Perms
	IsSuperAdmin bool
	Empty        bool
	User         string
	db           *conn.Database
}

func NewPermDomainService(db *conn.Database, user string, isSuperAdmin bool, empty bool) *PermDomainService {
	return &PermDomainService{
		mutexPerms:   sync.RWMutex{},
		Perms:        map[string]map[string]Perms{},
		IsSuperAdmin: isSuperAdmin,
		Empty:        empty,
		db:           db,
		User:         user,
	}
}

var cachePerms = map[string]map[string]map[string]Perms{}

func (p *PermDomainService) PermsBuilder(domain utils.DomainITF) {
	if domain.GetUserID() == "" {
		return
	}
	if cachePerms[domain.GetUserID()] != nil {
		p.Perms = cachePerms[domain.GetUserID()]
		return
	}
	datas, _ := p.db.SelectQueryWithRestriction(ds.DBPermission.Name, []interface{}{
		conn.FormatSQLRestrictionWhereByMap("", map[string]interface{}{
			utils.SpecialIDParam: p.db.BuildSelectQueryWithRestriction(
				ds.DBRolePermission.Name,
				map[string]interface{}{
					ds.DBRole.Name + "_id": p.db.BuildSelectQueryWithRestriction(
						ds.DBRoleAttribution.Name,
						map[string]interface{}{
							ds.DBUser.Name + "_id": domain.GetUserID(),
							ds.DBEntity.Name + "_id": p.db.BuildSelectQueryWithRestriction(
								ds.DBEntityUser.Name,
								map[string]interface{}{
									ds.DBUser.Name + "_id": domain.GetUserID(),
								}, true, ds.DBEntity.Name+"_id",
							),
						}, true, ds.DBRole.Name+"_id"),
				}, false, ds.DBPermission.Name+"_id"),
		}, false),
	}, false)
	if len(datas) == 0 {
		return
	}
	p.mutexPerms.Lock()
	defer p.mutexPerms.Unlock()
	for _, record := range datas {
		p.ProcessPermissionRecord(record)
	}
	cachePerms[domain.GetUserID()] = p.Perms
}

func (p *PermDomainService) ProcessPermissionRecord(record map[string]interface{}) {
	names := strings.Split(utils.ToString(record[sm.NAMEKEY]), ":")
	if len(names) < 2 {
		return
	}
	tName, n := names[0], names[1]
	if len(names) < 4 {
		n = names[0]
	}
	var perms Perms
	b, _ := json.Marshal(record)
	json.Unmarshal(b, &perms)
	if p.Perms[tName] == nil {
		p.Perms[tName] = make(map[string]Perms)
	}

	perm := p.Perms[tName][n]
	if slices.Index(sm.READLEVELACCESS, perms.Read) > slices.Index(sm.READLEVELACCESS, perm.Read) {
		perm.Read = perms.Read
	}
	perm = p.MapPerm(perm, perms)

	p.Perms[tName][n] = perm
}

func (p *PermDomainService) MapPerm(perm Perms, perms Perms) Perms {
	perm.Create = perms.Create
	perm.Update = perms.Update
	perm.Delete = perms.Delete
	return perm
}

func (p *PermDomainService) exception(tableName string, force bool, method utils.Method, isOwn bool) bool {
	if !force {
		return false
	}
	return slices.Contains(ds.OWNPERMISSIONEXCEPTION, tableName) && isOwn ||
		slices.Contains(ds.AllPERMISSIONEXCEPTION, tableName) ||
		(slices.Contains(ds.PERMISSIONEXCEPTION, tableName) && method == utils.SELECT) ||
		(slices.Contains(ds.PUPERMISSIONEXCEPTION, tableName) && method == utils.UPDATE) ||
		(slices.Contains(ds.POSTPERMISSIONEXCEPTION, tableName) && method == utils.CREATE)
}

func (p *PermDomainService) IsOwnPermission(tableName string, force bool, isOwn bool, domain utils.DomainITF) bool {
	if p.exception(tableName, !force, domain.GetMethod(), isOwn) || domain.GetMethod() != utils.SELECT {
		return slices.Contains(ds.OWNPERMISSIONEXCEPTION, tableName)
	}
	if len(p.Perms) == 0 {
		p.PermsBuilder(domain)
	}
	p.mutexPerms.Lock()
	defer p.mutexPerms.Unlock()
	if tPerms, ok := p.Perms[tableName]; ok {
		return tPerms[tableName].Read == sm.LEVELOWN
	}
	return false
}

// can redact a view based on perms.
func (p *PermDomainService) PermsCheck(tableName string, colName string, level string, isOwn bool, method utils.Method, domain utils.DomainITF) bool {
	return p.LocalPermsCheck(tableName, colName, level, "", isOwn, method, domain)
}
func (p *PermDomainService) LocalPermsCheck(tableName string, colName string, level string, destID string, isOwn bool, method utils.Method, domain utils.DomainITF) bool {
	// Super admin override or exception handling
	if p.IsSuperAdmin || p.exception(tableName, level == "" || level == "<nil>" || level == sm.LEVELNORMAL, method, isOwn) {
		return true
	}

	// Build permissions if empty
	if len(p.Perms) == 0 {
		p.PermsBuilder(domain)
	}
	// Retrieve permissions
	p.mutexPerms.Lock()
	perms := p.getPermissions(tableName, colName)
	p.mutexPerms.Unlock()
	// Handle SELECT method permissions
	schema, err := schserv.GetSchema(tableName)
	if err != nil {
		return false
	}
	accesGranted := true
	if method == utils.SELECT && !p.hasReadAccess(level, perms.Read) {
		accesGranted = p.IsShared(schema, destID, "read_access", true)
	}
	// Handle UPDATE and CREATE permissions
	if method == utils.CREATE && !perms.Create {
		accesGranted = false
	}
	if method == utils.UPDATE && !perms.Update {
		if !p.checkUpdateCreatePermissions(tableName, destID, domain) {
			accesGranted = p.IsShared(schema, destID, "update_access", true)
		}
	}
	if method == utils.DELETE && !perms.Delete {
		accesGranted = p.IsShared(schema, destID, "delete_access", true)
	}
	// Handle DELETE permissions
	return accesGranted
}

func (p *PermDomainService) getPermissions(tableName, colName string) Perms {
	if tPerms, ok := p.Perms[tableName]; ok {
		if cPerms, ok2 := tPerms[colName]; ok2 && colName != "" {
			return cPerms
		}
		return p.aggregatePermissions(tPerms, tableName)
	}
	return Perms{}
}

func (p *PermDomainService) aggregatePermissions(tPerms map[string]Perms, tableName string) Perms {
	perms := p.Perms[tableName][tableName]
	for _, perm := range tPerms {
		p.MapPerm(perm, perms)
	}
	return perms
}

func (p *PermDomainService) hasReadAccess(level, readPerm string) bool {
	if slices.Contains(sm.READLEVELACCESS, level) && level != sm.LEVELNORMAL {
		return p.compareAccessLevels(level, readPerm)
	}
	return readPerm == sm.LEVELNORMAL || readPerm == sm.LEVELOWN
}

func (p *PermDomainService) compareAccessLevels(level, readPerm string) bool {
	levelCount, _ := p.accessLevelIndex(level)
	compareCount, foundCompare := p.accessLevelIndex(readPerm)
	return compareCount >= levelCount && foundCompare
}

func (p *PermDomainService) accessLevelIndex(targetLevel string) (int, bool) {
	count := 0
	found := false
	for _, l := range sm.READLEVELACCESS {
		if l == targetLevel {
			found = true
			break
		} else if !found {
			count++
		}
	}
	return count, found
}
