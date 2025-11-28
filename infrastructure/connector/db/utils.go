package connector

import (
	"fmt"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

var COUNTREQUEST = 0

var SpecialTypes = []string{"char", "text", "date", "time", "interval", "var", "blob", "set", "enum", "year", "USER-DEFINED", "url",
	"upload", "upload_multiple", "html", "link_add"}

func Quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func RemoveLastChar(s string) string {
	r := []rune(s)
	if len(r) > 0 {
		return string(r[:len(r)-1])
	}
	return string(r)
}

func FormatMathViewQuery(algo string, col string, naming ...string) string {
	resName := "result"
	if len(naming) > 0 {
		resName = naming[0]
	}
	return strings.ToUpper(algo) + "(" + col + ") as " + resName
}

func DeleteFieldInInjection(injection string, searchField string) string {
	injection = SQLInjectionProtector(injection)
	ands := strings.Split(injection, "+")
	deleteOr := ""

	for _, andUndecoded := range ands {
		and, _ := url.QueryUnescape(fmt.Sprint(andUndecoded))
		ors := strings.Split(and, "|")
		if len(ors) == 0 {
			continue
		}
		for _, or := range ors {
			keyVal := []string{}
			if strings.Contains(or, "<>~") {
				keyVal = strings.Split(or, "<>~")
			} else if strings.Contains(or, "~") {
				keyVal = strings.Split(or, "~")
			} else if strings.Contains(or, ":") {
				keyVal = strings.Split(or, ":")
			}
			if len(keyVal) != 2 {
				continue
			}
			if keyVal[0] == searchField {
				deleteOr = or
				break
			}
		}
	}
	if deleteOr != "" {
		return strings.ReplaceAll(injection, deleteOr, "")
	}
	return injection
}

func GetFieldInInjection(injection string, searchField string) (string, string) {
	injection = SQLInjectionProtector(injection)
	ands := strings.Split(injection, "+")
	for _, andUndecoded := range ands {
		and, _ := url.QueryUnescape(fmt.Sprint(andUndecoded))
		ors := strings.Split(and, "|")
		if len(ors) == 0 {
			continue
		}
		for _, or := range ors {
			operator := "~"
			keyVal := []string{}
			if strings.Contains(or, "<>~") {
				keyVal = strings.Split(or, "<>~")
				operator = " NOT LIKE "
			} else if strings.Contains(or, "~") {
				keyVal = strings.Split(or, "~")
				operator = " LIKE "
			} else if strings.Contains(or, ":") {
				keyVal = strings.Split(or, ":")
				operator = "="
			}
			if len(keyVal) != 2 {
				continue
			}
			if keyVal[0] == searchField {
				return keyVal[1], operator
			}
		}
	}
	return "", ""
}

func FormatSQLRestrictionWhereInjection(injection string, getTypeAndLink func(string, string, string, func(string, string)) (string, string, string, string, string, error), special func(string, string)) string {
	alterRestr := ""
	injection = SQLInjectionProtector(injection)
	ands := strings.Split(injection, "+")
	for _, andUndecoded := range ands {
		and, _ := url.QueryUnescape(fmt.Sprint(andUndecoded))
		ors := strings.Split(and, "|")
		if len(ors) == 0 {
			continue
		}
		orRestr := ""
		for _, or := range ors {
			keyVal, operator := Compare(or)
			if len(keyVal) != 2 {
				continue
			}
			var err error
			var typ, link string
			keyVal[0], keyVal[1], operator, typ, link, err = getTypeAndLink(keyVal[0], keyVal[1], operator, special)
			if err != nil && keyVal[0] != "id" {
				continue
			}
			if len(strings.Trim(orRestr, " ")) > 0 {
				orRestr += " OR "
			}
			_, _, _, orRestr = MakeSqlItem(orRestr, typ, link, keyVal[0], keyVal[1], operator)
		}
		if len(orRestr) > 0 {
			if len(strings.Trim(alterRestr, " ")) > 0 {
				alterRestr += " AND "
			}
			alterRestr += "( " + orRestr + " )"
		}
	}
	alterRestr = strings.ReplaceAll(strings.ReplaceAll(alterRestr, " OR ()", ""), " AND ()", "")
	alterRestr = strings.ReplaceAll(strings.ReplaceAll(alterRestr, " () OR ", ""), "() AND ", "")
	alterRestr = strings.ReplaceAll(strings.ReplaceAll(alterRestr, " OR  OR ", ""), " AND  AND ", "")
	alterRestr = strings.ReplaceAll(alterRestr, "()", "")
	return alterRestr
}

func Compare(or string) ([]string, string) {
	operator := "~"
	keyVal := []string{}

	if strings.Contains(or, "<>~") {
		keyVal = strings.Split(or, "<>~")
		operator = " NOT LIKE "
	} else if strings.Contains(or, "~") {
		keyVal = strings.Split(or, "~")
		operator = " LIKE "
	} else if strings.Contains(or, "<>") {
		keyVal = strings.Split(or, "<>")
		operator = "!="
	} else if strings.Contains(or, "<:") {
		keyVal = strings.Split(or, "<:")
		operator = "<="
	} else if strings.Contains(or, ">:") {
		keyVal = strings.Split(or, ">:")
		operator = ">="
	} else if strings.Contains(or, ":") {
		keyVal = strings.Split(or, ":")
		operator = "="
	} else if strings.Contains(or, "<") {
		keyVal = strings.Split(or, "<")
		operator = "<"
	} else if strings.Contains(or, ">") {
		keyVal = strings.Split(or, ">")
		operator = ">"
	}
	return keyVal, operator
}

func MakeSqlItem(alterRestr string, typ string, foreignName string, key string, or string, operator string) (string, string, string, string) {
	sql := or
	sql = FormatForSQL(typ, sql)
	if sql == "" {
		return "", "", "", alterRestr
	}
	if strings.Contains(sql, "NULL") {
		operator = "IS "
	}
	if strings.Contains(or, "SELECT") {
		alterRestr += key + " " + operator + " " + sql
		return key, operator, sql, alterRestr
	}
	if foreignName != "" {
		if strings.Contains(sql, "%") {
			// LIKE
			subAlt := ""
			ssql := strings.Split(sql, " ")
			for _, s := range ssql {
				if s == "" {
					continue
				}
				if len(subAlt) > 0 {
					subAlt += " AND "
				}
				s = strings.ReplaceAll(s, "'%", "")
				s = strings.ReplaceAll(s, "%'", "")
				s = strings.ReplaceAll(s, "'", "''")
				subAlt += "(LOWER(name::text) LIKE LOWER('%" + s + "%') OR LOWER(id::text) LIKE LOWER('%" + s + "%'))"
			}
			alterRestr += key + " IN (SELECT id FROM " + foreignName + " WHERE " + subAlt + ")"
			return key, "IN", "(SELECT id FROM " + foreignName + " WHERE " + subAlt + ")", alterRestr
		} else {
			if strings.Contains(sql, "'") {
				if strings.Contains(sql, "NULL") {
					alterRestr += key + " IN (SELECT id FROM " + foreignName + " WHERE name IS " + sql + ")"
					return key, "IN", "(SELECT id FROM " + foreignName + " WHERE name IS " + sql + ")", alterRestr
				} else {
					alterRestr += key + " IN (SELECT id FROM " + foreignName + " WHERE LOWER(name) = LOWER(" + sql + "))"
					return key, "IN", "(SELECT id FROM " + foreignName + " WHERE LOWER(name) = LOWER(" + sql + "))", alterRestr
				}
			} else {
				alterRestr += key + " IN (SELECT id FROM " + foreignName + " WHERE id " + operator + " " + sql + ")"
				return key, "IN", "(SELECT id FROM " + foreignName + " WHERE id " + operator + " " + sql + ")", alterRestr
			}
		}
	} else if strings.Contains(sql, "%") && !strings.Contains(typ, "many") {
		no := "LIKE"
		if strings.Contains(operator, "NOT") || strings.Contains(operator, "!") {
			no = "NOT LIKE"
		}
		// LIKE
		ssql := strings.Split(sql, " ")
		for _, s := range ssql {
			if s == "" {
				continue
			}
			if len(alterRestr) > 0 {
				alterRestr += " AND "
			}
			s = strings.ReplaceAll(s, "'%", "")
			s = strings.ReplaceAll(s, "%'", "")
			alterRestr += "(LOWER(" + key + "::text) " + no + " LOWER('%" + strings.ReplaceAll(s, "'", "''") + "%'))"
		}
		return key, no, or, alterRestr
	} else {
		if strings.Contains(sql, "'") && !strings.Contains(typ, "enum") && !strings.Contains(typ, "many") {
			alterRestr += "LOWER(" + key + ") " + operator + " LOWER(" + sql + ")"
			return key, operator, or, alterRestr
		} else {
			alterRestr += key + " " + operator + " " + sql
			return key, operator, or, alterRestr
		}
	}
}

func FormatLimit(limited string, offset interface{}) string {
	if i, err := strconv.Atoi(limited); err == nil {
		limited = "LIMIT " + fmt.Sprintf("%v", i)
		if offset != nil && offset != "" {
			if i2, err := strconv.Atoi(fmt.Sprintf("%v", offset)); err == nil {
				limited += " OFFSET " + fmt.Sprintf("%v", i2)
			}
		}
	}
	return limited
}

func FormatOperatorSQLRestriction(operator interface{}, separator interface{}, name string, value interface{}, typ string) string {
	if operator == nil || separator == nil {
		return ""
	}
	filter := ""
	if len(filter) > 0 {
		filter += " " + fmt.Sprintf("%v", separator) + " "
	}
	if fmt.Sprintf("%v", operator) == "LIKE" {
		filter += "LOWER(" + name + "::text) " + fmt.Sprintf("%v", operator) + " LOWER('%" + fmt.Sprintf("%v", value) + "%')"
	} else {
		filter += name + " " + fmt.Sprintf("%v", operator) + " " + FormatForSQL(typ, value)
	}
	return filter
}

func FormatSQLRestrictionByList(SQLrestriction string, restrictions []interface{}, isOr bool) string {
	for _, v := range restrictions {
		if strings.Trim(fmt.Sprintf("%v", v), " ") == "" {
			continue
		}
		if len(strings.Trim(SQLrestriction, " ")) > 0 {
			if isOr {
				SQLrestriction += " OR "
			} else {
				SQLrestriction += " AND "
			}
		}
		SQLrestriction += fmt.Sprintf("%v", v)
	}
	return SQLrestriction
}

func FormatSQLRestrictionWhereByMap(SQLrestriction string, restrictions map[string]interface{}, isOr bool) string {
	for k, r := range restrictions {
		if r == "" {
			continue
		}
		k2 := k
		karr := strings.Split(k, "_")
		latest := karr[len(karr)-1]
		if _, err := strconv.Atoi(latest); err == nil {
			k2 = strings.ReplaceAll(k, "_"+latest, "")
		}
		if len(SQLrestriction) > 0 {
			if isOr {
				SQLrestriction += " OR "
			} else {
				SQLrestriction += " AND "
			}
		}
		if r == nil {
			if strings.Contains(k2, "!") {
				k2 = strings.ReplaceAll(k2, "!", "")
				SQLrestriction += k2 + " IS NOT NULL"
			} else {
				SQLrestriction += k2 + " IS NULL"
			}
		} else {
			not := strings.Contains(k2, "!")
			k2 = strings.ReplaceAll(k2, "!", "")
			divided := strings.Split(fmt.Sprintf("%v", r), " ")
			if len(divided) > 1 && slices.Contains([]string{"SELECT", "INSERT", "UPDATE", "DELETE"}, strings.ToUpper(divided[0])) {
				if not {
					SQLrestriction += k2 + " NOT IN (" + fmt.Sprintf("%v", r) + ")"
				} else {
					SQLrestriction += k2 + " IN (" + fmt.Sprintf("%v", r) + ")"
				}

			} else if len(divided) > 1 && slices.Contains([]string{"!SELECT", "!INSERT", "!UPDATE", "!DELETE"}, strings.ToUpper(divided[0])) {
				r = strings.ReplaceAll(fmt.Sprintf("%v", r), "!SELECT", "SELECT")
				r = strings.ReplaceAll(fmt.Sprintf("%v", r), "!INSERT", "INSERT")
				r = strings.ReplaceAll(fmt.Sprintf("%v", r), "!UPDATE", "UPDATE")
				r = strings.ReplaceAll(fmt.Sprintf("%v", r), "!DELETE", "DELETE")
				SQLrestriction += k2 + " NOT IN (" + fmt.Sprintf("%v", r) + ")"
			} else if reflect.TypeOf(r).Kind() == reflect.Slice {
				if not {
					SQLrestriction += k2 + " NOT IN (" + strings.Join(r.([]string), ",") + ")"
				} else {
					SQLrestriction += k2 + " IN (" + strings.Join(r.([]string), ",") + ")"
				}
			} else if strings.Contains(fmt.Sprintf("%v", r), "SELECT") {
				SQLrestriction += k2 + " IN (" + fmt.Sprintf("%v", r) + ")"
			} else {
				if strings.Contains(k2, "'") {
					k2 = "LOWER(" + k2 + ")"
				}
				str := fmt.Sprintf("%v", r)
				if strings.Contains(k2, "'") {
					str = "LOWER(" + k2 + ")"
				}
				if not {
					SQLrestriction += k2 + "!=" + str
				} else {
					SQLrestriction += k2 + "=" + str
				}

			}
		}
	}
	return SQLrestriction
}

func FormatSQLRestrictionWhere(SQLrestriction string, restriction string, verify func() bool, additionnalRestr ...string) (string, string) {
	if restriction != "" && verify() && len(restriction) > 0 {
		if len(SQLrestriction) > 0 {
			SQLrestriction += " AND "
		}
		SQLrestriction += restriction
	}
	lateAddition := ""
	for _, restr := range additionnalRestr {
		if strings.Contains(restr, " IN ") {
			if len(lateAddition) > 0 {
				lateAddition += " AND "
			}
			lateAddition += restr
			continue
		}
		if len(SQLrestriction) > 0 && len(restr) > 0 {
			SQLrestriction = restr + " AND " + SQLrestriction
		} else {
			SQLrestriction = restr
		}
	}
	return SQLrestriction, lateAddition
}

func FormatSQLOrderBy(orderBy []string, ascDesc []string, verify func(string) bool) []string {
	var order []string = []string{}
	if len(orderBy) == 0 {
		return []string{"id DESC"}
	}
	for i, el := range orderBy {
		if verify(el) && len(ascDesc) > i {
			upperAscDesc := strings.Replace(strings.ToUpper(ascDesc[i]), " ", "", -1)
			if upperAscDesc == "ASC" || upperAscDesc == "DESC" {
				order = append(order, SQLInjectionProtector(el+" "+upperAscDesc))
				continue
			}
			order = append(order, SQLInjectionProtector(el+" ASC"))
		}
	}
	return order
}

func FormatForSQL(datatype string, value interface{}) string {
	if value == nil {
		return ""
	}
	strval := fmt.Sprintf("%v", value)
	if len(strval) == 0 {
		return ""
	}
	if strval == "NULL" || strval == "NOT NULL" || strings.Contains(strval, "SELECT") {
		return strval
	}
	for _, typ := range SpecialTypes {
		if strings.Contains(datatype, typ) {
			if value == "CURRENT_TIMESTAMP" {
				return fmt.Sprint(value)
			} else {
				if strings.Contains(strings.ToUpper(datatype), "UPLOAD") {
					datatype = "varchar"
				} else if strings.Contains(strings.ToUpper(datatype), "HTML") {
					datatype = "text"
				}
				decodedValue := fmt.Sprint(value)
				if strings.Contains(strings.ToUpper(datatype), "DATE") || strings.Contains(strings.ToUpper(datatype), "TIME") {
					if len(decodedValue) > 10 {
						decodedValue = decodedValue[:10]
					}
				}
				if strings.Contains(fmt.Sprintf("%v", value), "()") {
					return strings.Replace(SQLInjectionProtector(decodedValue), "'", "''", -1)
				}
				return Quote(strings.Replace(SQLInjectionProtector(decodedValue), "'", "''", -1))
			}
		}
	}
	if strings.Contains(strval, "%") {
		decodedValue := fmt.Sprint(value)
		return Quote(strings.Replace(SQLInjectionProtector(decodedValue), "'", "''", -1))
	}
	return SQLInjectionProtector(strval)
}

func SQLInjectionProtector(injection string) string {
	quoteCounter := strings.Count(injection, "'")
	quoteCounter2 := strings.Count(injection, "\"")
	if (quoteCounter%2) != 0 || (quoteCounter2%2) != 0 {
		log.Error().Msg("injection alert: strange activity of quoting founded")
		return ""
	}
	notAllowedChar := []string{"«", "#", "union", ";", "%27", "%22", "%23", "%3B", "%29", "{", "}", "%7b", "%7d"}
	for _, char := range notAllowedChar {
		if strings.Contains(strings.ToLower(injection), char) {
			log.Error().Msg("injection alert: not allowed " + char + " filter")
			return ""
		}
	}
	return injection
}

func FormatEnumName(name string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(name), ",", "_"), "'", ""), "(", "__"), ")", ""), " ", "")
}

func FormatReverseEnumName(name string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(name), "__", "('"), "_", "','") + "')"
}
