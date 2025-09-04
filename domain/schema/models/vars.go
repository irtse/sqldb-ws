package models

import (
	"strings"
	"sync"
)

var CacheMutex = sync.Mutex{}

var NAMEKEY = "name"
var LABELKEY = "label"
var TYPEKEY = "type"

var FOREIGNTABLEKEY = "foreign_table"
var CONSTRAINTKEY = "constraints"
var LINKKEY = "link_id"

var STARTKEY = "start_date"
var ENDKEY = "end_date"

var STATEPENDING = "pending"
var STATEPROGRESSING = "progressing"
var STATEDISMISS = "dismiss"
var STATEREFUSED = "refused"
var STATECANCELED = "canceled"
var STATECOMPLETED = "completed"

var LEVELADMIN = "admin"
var LEVELMODERATOR = "moderator"
var LEVELRESPONSIBLE = "responsible"
var LEVELNORMAL = "normal"
var LEVELOWN = "own"
var READLEVELACCESS = []string{LEVELOWN, LEVELNORMAL, LEVELRESPONSIBLE, LEVELMODERATOR, LEVELADMIN}

var COUNT = "COUNT"
var AVG = "AVG"
var MIN = "MIN"
var MAX = "MAX"
var SUM = "SUM"
var MATHFUNC = []string{COUNT, AVG, MIN, MAX, SUM}

type DataType int

const (
	SMALLINT DataType = iota + 1
	INTEGER
	BIGINT
	FLOAT8
	DECIMAL
	TIME
	DATE
	TIMESTAMP
	BOOLEAN
	SMALLVARCHAR
	MEDIUMVARCHAR
	VARCHAR
	BIGVARCHAR
	TEXT
	ENUMOPERATOR
	ENUMSEPARATOR
	ENUMLEVEL
	ENUMLEVELCOMPLETE
	ENUMSTATE
	ENUMURGENCY
	ENUMLIFESTATE
	ENUMMATHFUNC
	ONETOMANY
	MANYTOMANY
	URL // is a link to the outside
	ENUMPLATFORM
	ENUMTRANSFORM
	ENUMBOOLEAN
	ENUMTIME
	ENUMCHART
	ENUMTRIGGER
	UPLOAD
	UPLOAD_MULTIPLE
	ENUMMODE
	HTML
	LINKADD
	MANYTOMANYADD
	ONETOMANYADD
)

func DataTypeToEnum() string {
	enum := "enum("
	for _, val := range DataTypeList() {
		enum += "'" + enumName(strings.ToLower(val)) + "', "
	}
	return enum[:len(enum)-2] + ")"
}

func enumName(name string) string {
	if strings.Contains(name, "enum") {
		return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(name), ",", "_"), "'", ""), "''", "'"), "(", "__"), ")", ""), " ", "")
	}
	return name
}

func DataTypeList() []string {
	return []string{"SMALLINT", "INTEGER", "BIGINT", "DOUBLE PRECISION", "DECIMAL", "TIME",
		"DATE", "TIMESTAMP", "BOOLEAN", "VARCHAR(32)", "VARCHAR(64)", "VARCHAR(128)", "VARCHAR(255)",
		"TEXT", "VARCHAR(6)", "ENUM('and', 'or')",
		"ENUM('" + LEVELADMIN + "', '" + LEVELMODERATOR + "', '" + LEVELRESPONSIBLE + "', '" + LEVELNORMAL + "')",
		"ENUM('" + LEVELADMIN + "', '" + LEVELMODERATOR + "', '" + LEVELRESPONSIBLE + "', '" + LEVELNORMAL + "', '" + LEVELOWN + "')",
		"ENUM('" + STATEPENDING + "', '" + STATEPROGRESSING + "', '" + STATEDISMISS + "', '" + STATEREFUSED + "', '" + STATECANCELED + "', '" + STATECOMPLETED + "')",
		"ENUM('low', 'normal', 'high')",
		"ENUM('all', 'new', 'old')",
		"ENUM('" + COUNT + "', '" + AVG + "', '" + MIN + "', '" + MAX + "', '" + SUM + "')",
		"ONETOMANY",
		"MANYTOMANY",
		"URL",
		"ENUM('email', 'sms', 'teams')",
		"ENUM('lowercase', 'uppercase')",
		"ENUM('yes', 'no', 'i don't know')",
		"ENUM('second', 'minute', 'hour','day', 'week', 'month', 'year')",
		"ENUM('line', 'pie', 'bar')", // enrich later
		"ENUM('mail', 'sms', 'teams notification', 'data')",
		"UPLOAD",
		"UPLOAD_MULTIPLE",
		"ENUM('manual', 'auto')",
		"HTML",
		"LINK_ADD",
		"MANYTOMANY_ADD",
		"ONETOMANY_ADD",
	}
}

func (s DataType) String() string { return strings.ToLower(DataTypeList()[s-1]) }

var CREATEPERMS = "write"
var UPDATEPERMS = "update"
var DELETEPERMS = "delete"
var READPERMS = "read"

var ADMINROLE = "admin"
var WRITEROLE = "manager"
var CREATEROLE = "creator"
var UPDATEROLE = "updater"
var READERROLE = "reader"
var PERMS = []string{CREATEPERMS, UPDATEPERMS, DELETEPERMS, READPERMS}

type MappedPerms map[string]bool

func (m MappedPerms) Anonymized() map[string]interface{} {
	anonimyzed := map[string]interface{}{}
	for key, val := range m {
		anonimyzed[key] = val
	}
	return anonimyzed
}

var MAIN_PERMS = map[string]MappedPerms{
	ADMINROLE:  {CREATEPERMS: true, UPDATEPERMS: true, DELETEPERMS: true},
	WRITEROLE:  {CREATEPERMS: true, UPDATEPERMS: true, DELETEPERMS: false},
	CREATEROLE: {CREATEPERMS: true, UPDATEPERMS: false, DELETEPERMS: false},
	UPDATEROLE: {CREATEPERMS: false, UPDATEPERMS: true, DELETEPERMS: false},
	READERROLE: {CREATEPERMS: false, UPDATEPERMS: false, DELETEPERMS: false},
}
