package connector

import (
	"database/sql"
	"fmt"
	"strconv"
)

/*
* RowResultToMap converts a row result to a map.
 */
func (db *Database) RowResultToMap(rows *sql.Rows,
	columnNames []string,
	columnType map[string]string) (map[string]interface{}, error) {
	columnPointers := make([]interface{}, len(columnNames))
	for i := range columnPointers {
		columnPointers[i] = new(interface{})
	}
	if err := rows.Scan(columnPointers...); err != nil {
		return nil, err
	}
	rowMap := map[string]interface{}{}
	for i, colName := range columnNames {
		rowMap[colName] = db.ParseColumnValue(columnType[colName], columnPointers[i].(*interface{}))
	}
	return rowMap, nil
}

/*
* ParseColumnValue converts the column value to the appropriate type.
 */
func (db *Database) ParseColumnValue(colType string, val *interface{}) interface{} {
	v := *val
	if v == nil {
		return v
	}
	if colType == "" {
		return fmt.Sprintf("%v", string(v.([]uint8)))
	}
	switch colType {
	case "MONEY", "NUMERIC", "DECIMAL", "DOUBLE", "FLOAT":
		if num, err := strconv.ParseFloat(string(v.([]uint8)), 64); err == nil {
			return num
		}
		return fmt.Sprintf("%v", v)
	case "TIMESTAMP", "DATE":
		if len(fmt.Sprintf("%v", *val)) > 10 {
			return fmt.Sprintf("%v", v)[:19]
		}
		return fmt.Sprintf("%v", v)
	default:
		return v
	}
}
