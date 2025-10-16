package connector

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

func (db *Database) BuildDeleteQueryWithRestriction(name string, restrictions map[string]interface{}, isOr bool) string {
	if db == nil || db.Conn == nil {
		db = Open(db)
		defer db.Close()
	}

	query := fmt.Sprintf("DELETE FROM %s", name)
	if t := FormatSQLRestrictionWhereByMap(db.SQLRestriction, restrictions, isOr); t != "" {
		query += " WHERE " + t
	}
	query = db.applyOrderAndLimit(query)
	return query
}

func (db *Database) BuildSimpleMathQueryWithRestriction(algo string, name string,
	restrictions interface{}, isOr bool, restr ...string) string {
	if db == nil || db.Conn == nil {
		db = Open(db)
		defer db.Close()
	}

	col := "*" // default to all columns
	query := "SELECT " + strings.ToUpper(algo) + "(" + col + ") as result FROM " + name
	kind := reflect.TypeOf(restrictions).Kind()
	if kind == reflect.Map && len(restrictions.(map[string]interface{})) > 0 {
		if t := FormatSQLRestrictionWhereByMap("", restrictions.(map[string]interface{}), isOr); t != "" {
			query += " WHERE " + t
		}
	} else if (kind == reflect.Array || kind == reflect.Slice) && len(restrictions.([]interface{})) > 0 {
		if t := FormatSQLRestrictionByList("", restrictions.([]interface{}), isOr); t != "" {
			query += " WHERE " + t
		}
	} else if db.SQLRestriction != "" {
		query += " WHERE " + db.SQLRestriction
	}
	return query
}

func (db *Database) BuildSelectQueryWithRestriction(name string, restrictions interface{}, isOr bool, view ...string) string {
	if db == nil || db.Conn == nil {
		db = Open(db)
		defer db.Close()
	}
	viewStr := "*"
	if db.SQLView != "" {
		viewStr = db.SQLView
	}
	if len(view) > 0 {
		viewStr = strings.Join(view, ",")
	}

	query := fmt.Sprintf("SELECT %s FROM %s", viewStr, name)
	kind := reflect.TypeOf(restrictions).Kind()
	if (kind == reflect.Map && len(restrictions.(map[string]interface{})) > 0) || ((kind == reflect.Array || kind == reflect.Slice) && len(restrictions.([]interface{})) > 0) || db.SQLRestriction != "" {
		query += " WHERE "
	}
	isAnd := false
	if kind == reflect.Map && len(restrictions.(map[string]interface{})) > 0 {
		if t := FormatSQLRestrictionWhereByMap("", restrictions.(map[string]interface{}), isOr); t != "" {
			query += t
			isAnd = true
		}
	} else if (kind == reflect.Array || kind == reflect.Slice) && len(restrictions.([]interface{})) > 0 {
		if t := FormatSQLRestrictionByList("", restrictions.([]interface{}), isOr); t != "" {
			if isAnd {
				query += " AND "
			}
			query += t
			isAnd = true
		}
	}
	if db.SQLRestriction != "" {
		if isAnd {
			query += " AND "
		}
		query += db.SQLRestriction
	}
	if len(query) > 5 && (query[len(query)-5:len(query)-1] == " AND") {
		query = query[0 : len(query)-5]
	}
	query = db.applyOrderAndLimit(query)
	return strings.ReplaceAll(query, "WHERE  ", "")
}

func (db *Database) BuildMathQuery(algo string, name string, naming ...string) string {
	if db == nil || db.Conn == nil {
		db = Open(db)
		defer db.Close()
	}
	resName := "result"
	if len(naming) > 0 {
		resName = naming[0]
	}
	col := "*" // default to all columns
	cols := strings.Split(db.SQLView, ",")
	if len(cols) > 0 { // if there are columns specified
		for _, c := range cols {
			if c != "id" && c != "" { // ignore id column
				if strings.Contains(strings.ToLower(c), " as ") {
					col = "(" + strings.Split(strings.ToLower(c), " as ")[0] + ")"
				} else {
					col = c
				}
				break
			}
		}
	}

	query := "SELECT " + strings.ToUpper(algo) + "(" + col + ") as " + resName + " FROM " + name
	if db.SQLRestriction != "" {
		query += " WHERE " + db.SQLRestriction
	}
	query = db.applyOrderAndLimit(query)
	return query
}

func (db *Database) BuildDeleteQuery(tableName string, colName string) string {
	if db == nil || db.Conn == nil {
		db = Open(db)
		defer db.Close()
	}

	if colName == "" { // if no column name is specified then delete in rows
		return "DELETE FROM " + tableName + " WHERE " + db.SQLRestriction
	}
	return "ALTER TABLE " + tableName + " DROP " + colName
}

func (db *Database) BuildDropTableQueries(name string) []string {
	return []string{
		"DROP TABLE " + name,
		"DROP SEQUENCE " + name + "_id_seq",
	}
}

func (db *Database) BuildSchemaQuery(name string) string {
	if !slices.Contains(drivers, db.Driver) {
		log.Error().Msg("Invalid DB driver!")
		return ""
	}
	switch db.Driver {
	case MySQLDriver:
		return "SELECT COLUMN_NAME as name, column_default as default_value, IS_NULLABLE as null, CONCAT(DATA_TYPE, COALESCE(CONCAT('(' , CHARACTER_MAXIMUM_LENGTH, ')'), '')) as type FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = " + Quote(name) + ";"
	case PostgresDriver:
		return "SELECT column_name :: varchar as name, column_default as default_value, IS_NULLABLE as null, REPLACE(REPLACE(data_type,'character varying','varchar'),'character','char') || COALESCE('(' || character_maximum_length || ')', '') as type, col_description('public." + name + "'::regclass, ordinal_position) as comment  from INFORMATION_SCHEMA.COLUMNS where table_name =" + Quote(name) + ";"
	default:
		return ""
	}
}

func (db *Database) BuildListTableQuery() string {
	if db == nil || db.Conn == nil {
		db = Open(db)
		defer db.Close()
	}
	if !slices.Contains(drivers, db.Driver) {
		log.Error().Msg("Invalid DB driver!")
		return ""
	}
	switch db.Driver {
	case MySQLDriver:
		return "SELECT TABLE_NAME as name FROM information_schema.TABLES WHERE TABLE_TYPE LIKE 'BASE_TABLE';"
	case PostgresDriver:
		return "SELECT table_name :: varchar as name FROM information_schema.tables WHERE table_schema = 'public' ORDER BY table_name;"
	default:
		return ""
	}
}

func (db *Database) BuildCreateTableQuery(name string) string {
	return fmt.Sprintf("CREATE TABLE %s (id SERIAL PRIMARY KEY, active BOOLEAN DEFAULT TRUE, is_draft BOOLEAN DEFAULT FALSE)", name)
}

func (db *Database) BuildCreateQueries(tableName string, values string, cols string, typ string) []string {
	if db == nil || db.Conn == nil {
		db = Open(db)
		defer db.Close()
	}
	queries := []string{}
	if typ != "" {
		if typ == "" || typ == "<nil>" || cols == "" || cols == "<nil>" {
			return queries
		}
		if strings.Contains(strings.ToLower(typ), "enum") && db.Driver == PostgresDriver {
			if strings.Contains(strings.ToLower(typ), "')") {
				queries = append(queries, "CREATE TYPE "+FormatEnumName(typ)+" AS "+typ)
			} else {
				queries = append(queries, "CREATE TYPE "+FormatEnumName(typ)+" AS "+FormatReverseEnumName(typ))
			}
			queries = append(queries, "ALTER TABLE "+tableName+" ADD "+cols+" "+FormatEnumName(typ)+" NULL")
		} else {
			queries = append(queries, "ALTER TABLE "+tableName+" ADD "+cols+" "+typ+" NULL")
		}
	} else {
		if values == "" || cols == "" {
			return []string{""}
		}
		queries = append(queries, "INSERT INTO "+tableName+"("+cols+") VALUES ("+values+")")
		if db.Driver == PostgresDriver {
			queries[len(queries)-1] = queries[len(queries)-1] + " RETURNING ID"
		}
	}
	return queries

}

func (db *Database) ApplyQueryFilters(restr string, order string, limit string, views string, additionnalRestriction ...string) {
	if db == nil || db.Conn == nil {
		db = Open(db)
		defer db.Close()
	}
	if restr != "" {
		if len(db.SQLRestriction) > 0 {
			db.SQLRestriction = db.SQLRestriction + " AND " + restr
		} else {
			db.SQLRestriction = restr
		}
	}
	if len(additionnalRestriction) > 0 {
		for _, r := range additionnalRestriction {
			if r == "" {
				continue
			}
			if len(db.SQLRestriction) > 0 {
				db.SQLRestriction = db.SQLRestriction + " AND (" + r + ")"
			} else {
				db.SQLRestriction = r
			}
		}
	}
	if order != "" {
		db.SQLOrder = order
	}
	if limit != "" {
		db.SQLLimit = limit
	}
	if views != "" {
		db.SQLView = views
	}
}

func (db *Database) BuildUpdateQuery(tablename string, col string, value interface{}, set string,
	cols []string, colValues []string, isUpdate bool, verify func(string) (string, bool)) (string, []string, []string) {
	if db == nil || db.Conn == nil {
		db = Open(db)
		defer db.Close()
	}
	if col == "id" && fmt.Sprintf("%v", value) != "0" && fmt.Sprintf("%v", value) != "" && isUpdate {
		db.SQLRestriction = "id=" + fmt.Sprintf("%v", value) + " "
	}
	if typ, ok := verify(col); ok && (!slices.Contains([]string{"NULL", "null", "'null'", ""}, FormatForSQL(strings.Split(typ, ":")[0], value)) || typ == "") {
		if value == nil || value == "" {
			set += " " + col + " IS " + Quote(strings.ReplaceAll(fmt.Sprintf("%v", value), "'", "''")) + ","
			cols = append(cols, col)
			colValues = append(colValues, "NULL")
			return set, cols, colValues
		}
		if value == "" || (reflect.TypeOf(value) != nil && reflect.TypeOf(value).Kind().String() == "string") {
			set += " " + col + "=" + Quote(strings.ReplaceAll(fmt.Sprintf("%v", value), "'", "''")) + ","
			cols = append(cols, col)
			colValues = append(colValues, Quote(strings.ReplaceAll(fmt.Sprintf("%v", value), "'", "''")))
		} else {
			set += " " + col + "=" + FormatForSQL(strings.Split(typ, ":")[0], value) + ","
			cols = append(cols, col)
			colValues = append(colValues, FormatForSQL(strings.Split(typ, ":")[0], value))
		}
	}
	return set, cols, colValues
}

func (db *Database) BuildUpdateQueryWithRestriction(tableName string, record map[string]interface{}, restrictions map[string]interface{}, isOr bool) (string, error) {
	if db == nil || db.Conn == nil {
		db = Open(db)
		defer db.Close()
	}
	set := ""
	for key, element := range record {
		set, _, _ = db.BuildUpdateQuery(tableName, key, element, set, []string{}, []string{}, true, func(s string) (string, bool) { return "", true })
	}
	set = RemoveLastChar(set)
	if set == "" {
		return "", errors.New("no value to update")
	}
	if strings.Contains(FormatSQLRestrictionWhereByMap("", restrictions, isOr), "main.") {
		tableName = tableName + " as main "
	}
	query := "UPDATE " + tableName + " SET " + set
	if t := FormatSQLRestrictionWhereByMap("", restrictions, isOr); t != "" {
		query += " WHERE " + t
	}
	return query, nil
}

func (db *Database) BuildUpdateRowQuery(tableName string, record map[string]interface{}, verify func(string) (string, bool)) (string, error) {
	if db == nil || db.Conn == nil {
		db = Open(db)
		defer db.Close()
	}
	set := ""
	for key, element := range record {
		set, _, _ = db.BuildUpdateQuery(tableName, key, element, set, []string{}, []string{}, true, verify)
	}
	set = RemoveLastChar(set)
	if set == "" {
		return "", errors.New("no value to update")
	}
	if strings.Contains(db.SQLRestriction, "main.") {
		tableName = tableName + " as main "
	}
	query := "UPDATE " + tableName + " SET " + set
	if db.SQLRestriction != "" {
		query += " WHERE " + db.SQLRestriction
	}

	query = db.applyOrderAndLimit(query)
	return query, nil
}

func (db *Database) BuildUpdateColumnQueries(tableName string, record map[string]interface{}, verify func(string) (string, bool)) ([]string, error) {
	if db == nil || db.Conn == nil {
		db = Open(db)
		defer db.Close()
	}
	queries := []string{}
	typ := fmt.Sprintf("%v", record["type"])
	name := fmt.Sprintf("%v", record["name"])
	if typ == "" || typ == "<nil>" || name == "" || name == "<nil>" {
		return queries, errors.New("missing one of the needed value type & name")
	}
	if strings.TrimSpace(fmt.Sprintf("%v", record["constraints"])) != "" && strings.TrimSpace(fmt.Sprintf("%v", record["constraints"])) != "<nil>" {
		queries = append(queries, "ALTER TABLE "+tableName+" DROP CONSTRAINT "+tableName+"_"+name+"_"+fmt.Sprintf("%v", record["constraints"])+";")
		queries = append(queries, "ALTER TABLE "+tableName+" ADD CONSTRAINT "+tableName+"_"+name+"_"+fmt.Sprintf("%v", record["constraints"])+" "+strings.ToUpper(fmt.Sprintf("%v", record["constraints"]))+"("+name+");")
	}
	if strings.TrimSpace(fmt.Sprintf("%v", record["foreign_table"])) != "" && strings.TrimSpace(fmt.Sprintf("%v", record["foreign_table"])) != "<nil>" {
		queries = append(queries, "ALTER TABLE "+tableName+" DROP CONSTRAINT fk_"+name+";")
		queries = append(queries, "ALTER TABLE "+tableName+" ADD CONSTRAINT  fk_"+name+" FOREIGN KEY("+name+") REFERENCES "+fmt.Sprintf("%v", record["foreign_table"])+"(id) ON DELETE CASCADE;")
	}
	if strings.TrimSpace(fmt.Sprintf("%v", record["default_value"])) != "" && strings.TrimSpace(fmt.Sprintf("%v", record["default_value"])) != "<nil>" && FormatForSQL(typ, fmt.Sprintf("%v", record["default_value"])) != "NULL" {
		queries = append(queries, "ALTER TABLE "+tableName+" ALTER "+name+" SET DEFAULT "+FormatForSQL(typ, fmt.Sprintf("%v", record["default_value"]))+";") // then iterate on field to update value if null
	}
	return queries, nil
}

func (db *Database) applyOrderAndLimit(query string) string {
	if db == nil || db.Conn == nil {
		db = Open(db)
		defer db.Close()
	}
	if db.SQLOrder != "" {
		query += " ORDER BY " + db.SQLOrder
	}
	if db.SQLGroupBy != "" {
		query += " GROUP BY " + db.SQLLimit
	}
	if db.SQLLimit != "" {
		query += " " + db.SQLLimit
	}
	return query
}
