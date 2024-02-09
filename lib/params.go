package lib

// API COMMON query params !
type Params map[string]string

const ReservedParam = "all" // IMPORTANT IS THE DEFAULT PARAMS FOR ROWS & COLUMNS PARAMS

const RootTableParam = "table" 
const RootToTableParam = "totable" 
const RootRowsParam = "rows" 
const RootColumnsParam = "columns" 
const RootOrderParam = "orderby" 
const RootDirParam = "dir"
const RootRawView = "rawview"
const RootShallow = "shallow"
const RootSuperCall = "super"
const RootEmpty = "empty"
const RootDestTableIDParam = "dest_table_id" 

var RootParamsDesc = map[string]string{
	RootRowsParam : "needed on a rows level request (value=all for post/put method or a get/delete all)",
    RootColumnsParam : "needed on a columns level request (POST/PUT/DELETE with no rows query params) will set up a view on row level (show only expected columns)",
	RootShallow : "activate a lightest response (name only)",
	RootOrderParam : "sets up a sql order in query (don't even try to inject you little joker)",
	RootDirParam : "sets up a sql direction in query (ex.ASC|DESC) (don't even try to inject you little joker)",
	RootRawView : "set 'enable' to activate a response without the main response format (only available if super admin)",
}
var HiddenParams = []string{RootDestTableIDParam}
var RootParams = []string{RootRowsParam, RootColumnsParam, RootOrderParam, RootDirParam, RootShallow, RootRawView, RootSuperCall}

const SpecialIDParam = "id" 

var MAIN_PREFIX = "generic"