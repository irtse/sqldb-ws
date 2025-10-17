package utils

import (
	"net/url"
	connector "sqldb-ws/infrastructure/connector/db"
	"strings"
	"sync"
)

type Params struct {
	Values map[string]string
	Mutex  *sync.RWMutex
}

func NewParams(vals map[string]string) Params {
	return Params{
		Values: vals,
		Mutex:  &sync.RWMutex{},
	}
}

func (p Params) GetLine() string {
	l := []string{}
	for k, v := range p.Values {
		l = append(l, k+":"+v)
	}
	return strings.Join(l, ",")
}

func (p Params) GetAsArgs(key string) []string {
	p.Mutex.RLock()
	defer p.Mutex.RUnlock()
	if arg, ok := p.Values[key]; ok {
		return []string{arg}
	}
	return []string{}
}

func (p Params) Set(key string, value string) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	p.Values[key] = value
}

func (p Params) Has(key string) bool {
	p.Mutex.RLock()
	defer p.Mutex.RUnlock()
	_, ok := p.Values[key]

	return ok
}

func (p Params) Get(key string) (string, bool) {
	p.Mutex.RLock()
	defer p.Mutex.RUnlock()
	k, ok := p.Values[key]

	return k, ok
}

func (p Params) SimpleDelete(key string) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	delete(p.Values, key)
}

func (p Params) GetOrder(condition func(string) bool, order []string) []string {
	p.Mutex.RLock()
	defer p.Mutex.RUnlock()

	direction := []string{}
	if orderBy, ok := p.Values[RootOrderParam]; ok {
		if dir, ok2 := p.Values[RootDirParam]; ok2 {
			direction = strings.Split(ToString(dir), ",")
		}
		order = strings.Split(ToString(orderBy), ",")
	}
	return connector.FormatSQLOrderBy(order, direction, func(el string) bool {
		return condition(el) && el != SpecialIDParam
	})
}

func (p Params) Copy() Params {
	p.Mutex.RLock()
	defer p.Mutex.RUnlock()

	to := map[string]string{}
	for k, v := range p.Values {
		to[k] = v
	}
	return Params{
		Values: to,
		Mutex:  &sync.RWMutex{},
	}
}

func (p Params) GetLimit(limited string) string {
	p.Mutex.RLock()
	defer p.Mutex.RUnlock()

	if limit, ok := p.Values[RootLimit]; ok {
		if offset, ok2 := p.Values[RootOffset]; ok2 {
			return connector.FormatLimit(limit, offset)
		}
		return connector.FormatLimit(limit, "")
	}
	return limited
}

func (p Params) RootShallow() Params {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()

	p.Values[RootShallow] = "enable"
	return p
}

func (p Params) RootRaw() Params {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()

	p.Values[RootRawView] = "enable"
	return p
}

func (p Params) Enrich(param map[string]interface{}) Params {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()

	for k, v := range param {
		p.Values[k] = ToString(v)
	}
	return p
}

func (p Params) Add(k string, val interface{}, condition func(string) bool) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()

	if val == nil || val == "" || !condition(k) {
		return
	}
	v, _ := url.QueryUnescape(ToString(ToString(val)))
	p.Values[k] = v
}

func (p Params) AddMap(vals map[string]interface{}, condition func(string) bool) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()

	for k, val := range vals {
		p.Add(k, val, condition)
	}
}

func (p Params) UpdateParamsWithFilters(view, dir string) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	if view != "" {
		p.Values[RootColumnsParam] = view
	}
	if dir != "" {
		p.Values[RootDirParam] = dir
	}
}

func (p Params) EnrichCondition(flat map[string]string, condition func(string) bool) Params {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()

	for k, v := range flat {
		if condition(k) {
			if k == SpecialSubIDParam {
				k = SpecialIDParam
			}
			p.Values[k] = v
		}
	}
	return p
}

func (p Params) Anonymized() map[string]interface{} {
	p.Mutex.RLock()
	defer p.Mutex.RUnlock()

	newM := map[string]interface{}{}
	for k, v := range p.Values {
		newM[k] = v
	}
	return newM
}

func (p Params) Delete(condition func(string) bool) Params {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()

	toDelete := []string{}
	for k := range p.Values {
		if condition(k) {
			toDelete = append(toDelete, k)
		}
	}
	for _, k := range toDelete {
		delete(p.Values, k)
	}
	return p
}

func AllParams(table string) Params {
	return Params{
		Values: map[string]string{
			RootTableParam: table, RootRowsParam: ReservedParam,
		},
		Mutex: &sync.RWMutex{},
	}
}

func GetTableTargetParameters(tableName interface{}) Params {
	if tableName == nil {
		return Params{}
	}
	return Params{Values: map[string]string{RootTableParam: ToString(tableName)}, Mutex: &sync.RWMutex{}}
}

func GetColumnTargetParameters(tableName interface{}, col interface{}) Params {
	if col == nil || ToString(col) == "" {
		col = ReservedParam
	}
	return Params{Values: map[string]string{RootTableParam: ToString(tableName), RootColumnsParam: ToString(col)}, Mutex: &sync.RWMutex{}}
}

func GetRowTargetParameters(tableName interface{}, row interface{}) Params {
	if row == nil || ToString(row) == "" {
		row = ReservedParam
	}
	return Params{Values: map[string]string{RootTableParam: ToString(tableName), RootRowsParam: ToString(row)}, Mutex: &sync.RWMutex{}}
}

const ReservedParam = "all" // IMPORTANT IS THE DEFAULT PARAMS FOR ROWS & COLUMNS PARAMS
const RootTableParam = "table"
const RootRowsParam = "rows"
const RootColumnsParam = "columns"
const RootOrderParam = "orderby"
const RootDirParam = "dir"
const RootFilterNewState = "filter_status" // all - new - old
const RootFilterLine = "filter_line"
const RootFilterMode = "filter_mode" // + == "and" | == "or" ~ == "like" : == "=" > == ">" < == "<"
const RootRawView = "rawview"
const RootExport = "export"
const RootFilename = "filename"
const RootShallow = "shallow"
const RootDestIDParam = "dest_id"
const RootDestTableIDParam = "dbdest_table_id"
const RootCommandRow = "command_row"
const RootCommandCols = "command_columns"
const RootLimit = "limit"
const RootOffset = "offset"
const RootScope = "scope"
const RootGroupBy = "group_by"

var RootParamsDesc = map[string]string{
	RootRowsParam:    "needed on a rows level request (value=all for post/put method or a get/delete all)",
	RootColumnsParam: "needed on a columns level request (POST/PUT/DELETE with no rows query params) will set up a view on row level (show only expected columns)",
	RootShallow:      "activate a lightest response (name only)",
	RootOrderParam:   "sets up a sql order in query (don't even try to inject you little joker)",
	RootDirParam:     "sets up a sql direction in query (ex.ASC|DESC) (don't even try to inject you little joker)",
	RootRawView:      "set 'enable' to activate a response without the main response format",
	RootFilterLine:   "set a filter command line compose as (key~value(+|))",
}
var HiddenParams = []string{}
var RootParams = []string{RootRowsParam, RootColumnsParam, RootOrderParam, RootDirParam, RootLimit, RootOffset,
	RootShallow, RootRawView, RootExport, RootFilename, RootFilterNewState, RootFilterLine,
	RootCommandRow, SpecialIDParam, RootGroupBy, RootScope, RootDestIDParam}

const SpecialIDParam = "id"
const SpecialSubIDParam = "subid"
