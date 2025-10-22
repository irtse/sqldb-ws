package models

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"sqldb-ws/domain/utils"
	"strconv"
	"strings"
	"sync"
	"time"
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

func CompareList(operator string, typ string, val string, val2 []string, record utils.Record, meth utils.Method) (bool, error) {
	if len(val2) == 1 {
		v := val2[0]
		if record[v] != nil {
			v = fmt.Sprintf("%v", record[fmt.Sprintf("%v", v)])
		}
		return Compare(operator, typ, val, v, record, meth)
	} else {
		found := false
		if strings.ToUpper(typ) != "IN" {
			found = true
		}
		for _, v := range val2 {
			vv := v
			if record[vv] != nil {
				vv = fmt.Sprintf("%v", record[fmt.Sprintf("%v", vv)])
			}
			ok, _ := Compare(operator, typ, val, vv, record, meth)
			if strings.ToUpper(typ) == "IN" && ok {
				found = true
			} else if strings.ToUpper(typ) == "NOT IN" && ok {
				found = false
			} else if !ok {
				found = false
			}
		}
		if found {
			return true, nil
		}
		return false, errors.New("list comparison failed " + operator)
	}
}

func Compare(operator string, typ string, val string, val2 string, record utils.Record, meth utils.Method) (bool, error) {
	if record[val2] != nil {
		val2 = fmt.Sprintf("%v", record[val2])
	}
	if ok, a, b := IsDateComparable(typ, val, val2, record, operator, meth); ok {
		switch operator {
		case ">":
			return a.After(b), nil
		case "<":
			return a.Before(b), nil
		case ">=":
			return a.After(b) || a == b, nil
		case "<=":
			return a.Before(b) || a == b, nil
		case "=", "==", "IN":
			return a == b, nil
		case "!=", "<>", "NOT IN":
			return a != b, nil
		}
	}

	if ok, a, b := IsFloatComparable(typ, val, val2); ok {
		switch operator {
		case ">":
			return a > b, nil
		case "<":
			return a < b, nil
		case ">=":
			return a >= b, nil
		case "<=":
			return a <= b, nil
		case "=", "==", "IN":
			return a == b, nil
		case "!=", "<>", "NOT IN":
			return a != b, nil
		}
	}

	if (strings.ToLower(fmt.Sprintf("%v", val)) == "true" || strings.ToLower(fmt.Sprintf("%v", val)) == "false") && (strings.ToLower(fmt.Sprintf("%v", val2)) == "true" || strings.ToLower(fmt.Sprintf("%v", val2)) == "false") {
		switch operator {
		case "=", "==", "IN":
			return strings.ToLower(fmt.Sprintf("%v", val)) == strings.ToLower(fmt.Sprintf("%v", val2)), nil
		case "!=", "<>", "NOT IN":
			return strings.ToLower(fmt.Sprintf("%v", val)) != strings.ToLower(fmt.Sprintf("%v", val2)), nil
		}
	}

	if ok, a, b := IsStringComparable(typ, val, val2); ok {
		switch operator {
		case "LIKE":
			return strings.Contains(a, b), nil
		case "=", "==", "IN":
			return a == b, nil
		case "!=", "<>", "NOT IN":
			return a != b, nil
		}
	}
	if ok, a, b := IsBoolComparable(typ, val, val2); ok {
		switch operator {
		case "=", "==", "IN":
			return a == b, nil
		case "!=", "<>", "NOT IN":
			return a != b, nil
		}
	}
	return false, fmt.Errorf("unknown comparator: %s", operator)
}

var layouts = []string{
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02",
	"01/02/2006",
	"02/01/2006",
	"2006-01-02T15:04:05",
	"January 2, 2006 3:04 PM",
}

func parseDate(input string) (time.Time, error) {
	for _, layout := range layouts {
		if t, err := time.Parse(layout, input); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unknown date format: %s", input)
}

func IsDateComparable(typ string, val string, val2 string, record utils.Record, operator string, meth utils.Method) (bool, time.Time, time.Time) {
	if slices.Contains([]string{"TIME", "DATE", "TIMESTAMP"}, strings.ToUpper(typ)) {
		time1, err := parseDate(val)
		if strings.Contains(strings.ToUpper(val2), "NOW") || strings.Contains(strings.ToUpper(val2), "CURRENT_DATE") {

			now := time.Now().UTC()
			if meth == utils.UPDATE {
				now = now.Add(time.Duration(-175200) * time.Hour)
			}
			rnow, _ := time.Parse("2006-01-02", now.Format("2006-01-02"))
			return err == nil, time1, rnow.UTC()
		}
		if strings.Contains(val2, "date") && record[val2] == nil {
			if strings.Contains(operator, "<") {
				return err == nil, time1, time.Now().Add(time.Hour * 24)
			} else {
				return err == nil, time1, time.Now()
			}
		}
		time2, err2 := parseDate(val2)
		return err == nil && err2 == nil, time1, time2
	}
	return false, time.Now().UTC(), time.Now().UTC()
}

func IsBoolComparable(typ string, val string, val2 string) (bool, bool, bool) {
	if slices.Contains([]string{"BOOLEAN"}, strings.ToUpper(typ)) {
		return true, strings.ToLower(val) == "true", strings.ToLower(val2) == "true"
	}
	return false, false, false
}

func IsStringComparable(typ string, val string, val2 string) (bool, string, string) {
	if slices.Contains([]string{"VARCHAR(32)", "VARCHAR(64)", "VARCHAR(128)", "VARCHAR(255)",
		"TEXT", "VARCHAR(6)", "URL", "UPLOAD", "UPLOAD_MULTIPLE", "HTML"}, strings.ToUpper(typ)) || strings.Contains(typ, "ENUM") {
		return true, val, val2
	}
	// HERE IT IS A POSSIBILITY OF EVAL... UPPER LOWER ETC
	return false, "", ""
}

func IsFloatComparable(typ string, val string, val2 string) (bool, float64, float64) {
	if slices.Contains([]string{"SMALLINT", "INTEGER", "BIGINT", "DOUBLE PRECISION", "DECIMAL", "LINK_ADD", "LINK"}, strings.ToUpper(typ)) {
		f, err := strconv.ParseFloat(val, 64)
		if strings.Contains(strings.ToUpper(val2), "RAND") {
			return err == nil, f, rand.Float64()
		}
		// HERE IT IS A POSSIBILITY OF EVAL... SUM MAX ETC.
		f2, err2 := strconv.ParseFloat(val2, 64)
		return err == nil && err2 == nil, f, f2
	}
	return false, 0, 0
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
