package jdb

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
)

const (
	COMMAND     = "command"
	QUERY       = "query"
	DEFINITION  = "definition"
	CONTROL     = "control"
	TRANSACTION = "transaction"
	OTHER       = "other"
)

/**
* tipoSQL
* @param query string
* @return string
**/
func TipoSQL(query string) string {
	q := strings.TrimSpace(strings.ToUpper(query))

	parts := strings.Fields(q)
	if len(parts) == 0 {
		return OTHER
	}

	cmd := parts[0]

	switch cmd {
	case "SELECT":
		return QUERY
	case "INSERT", "UPDATE", "DELETE", "MERGE":
		return COMMAND
	case "CREATE", "ALTER", "DROP", "TRUNCATE":
		return DEFINITION
	case "GRANT", "REVOKE":
		return CONTROL
	case "COMMIT", "ROLLBACK", "SAVEPOINT", "SET":
		return TRANSACTION
	default:
		return OTHER
	}
}

/**
* SQLParse
* @param sql string
* @param args ...any
* @return string
**/
func SQLParse(sql string, args ...any) string {
	for i := range args {
		old := fmt.Sprintf(`$%d`, i+1)
		new := fmt.Sprintf(`{$%d}`, i+1)
		sql = strings.ReplaceAll(sql, old, new)
	}

	for i, arg := range args {
		old := fmt.Sprintf(`{$%d}`, i+1)
		new := fmt.Sprintf(`%v`, Quoted(arg))
		sql = strings.ReplaceAll(sql, old, new)
	}

	return sql
}

/**
* SQLUnQuote
* @param sql string
* @param args ...any
* @return string
**/
func SQLUnQuote(sql string, args ...any) string {
	for i := range args {
		old := fmt.Sprintf(`$%d`, i+1)
		new := fmt.Sprintf(`{$%d}`, i+1)
		sql = strings.ReplaceAll(sql, old, new)
	}

	for i, arg := range args {
		old := fmt.Sprintf(`{$%d}`, i+1)
		new := fmt.Sprintf(`%v`, arg)
		sql = strings.ReplaceAll(sql, old, new)
	}

	return sql
}

/**
* Quote
* @param val any
* @return any
**/
func Quoted(val any) any {
	format := `'%v'`
	switch v := val.(type) {
	case string:
		return fmt.Sprintf(format, v)
	case int:
		return v
	case float64:
		return v
	case float32:
		return v
	case int16:
		return v
	case int32:
		return v
	case int64:
		return v
	case bool:
		return v
	case time.Time:
		return fmt.Sprintf(format, v.Format("2006-01-02 15:04:05"))
	case et.Json:
		return fmt.Sprintf(format, v.ToString())
	case map[string]interface{}:
		return fmt.Sprintf(format, et.Json(v).ToString())
	case []string, []et.Json, []interface{}, []map[string]interface{}:
		bt, err := json.Marshal(v)
		if err != nil {
			logs.Errorf("Quote, type:%v, value:%v, error marshalling array: %v", reflect.TypeOf(v), v, err)
			return strs.Format(format, `[]`)
		}
		return fmt.Sprintf(format, string(bt))
	case []uint8:
		b := []byte(val.([]uint8))
		return fmt.Sprintf("'\\x%s'", hex.EncodeToString(b))
	case nil:
		return fmt.Sprintf(`%s`, "NULL")
	default:
		logs.Errorf("Quote, type:%v, value:%v", reflect.TypeOf(v), v)
		return val
	}
}

/**
* Literal
* @param val any
* @return any
**/
func Literal(val any) any {
	format := `"%v"`
	switch v := val.(type) {
	case string:
		return fmt.Sprintf(format, v)
	case int:
		return v
	case float64:
		return v
	case float32:
		return v
	case int16:
		return v
	case int32:
		return v
	case int64:
		return v
	case bool:
		return v
	case time.Time:
		return fmt.Sprintf(format, v.Format("2006-01-02 15:04:05"))
	case et.Json:
		return fmt.Sprintf(format, v.ToString())
	case map[string]interface{}:
		return fmt.Sprintf(format, et.Json(v).ToString())
	case []string, []et.Json, []interface{}, []map[string]interface{}:
		bt, err := json.Marshal(v)
		if err != nil {
			logs.Errorf("Quote, type:%v, value:%v, error marshalling array: %v", reflect.TypeOf(v), v, err)
			return strs.Format(format, `[]`)
		}
		return fmt.Sprintf(format, string(bt))
	case []uint8:
		b := []byte(val.([]uint8))
		return fmt.Sprintf("'\\x%s'", hex.EncodeToString(b))
	case nil:
		return fmt.Sprintf(`%s`, "NULL")
	default:
		logs.Errorf("Quote, type:%v, value:%v", reflect.TypeOf(v), v)
		return val
	}
}

/**
* RowsToItems
* @param rows *sql.Rows
* @return et.Items
**/
func RowsToItems(rows *sql.Rows) et.Items {
	var result = et.Items{Result: []et.Json{}}

	append := func(item et.Json) {
		result.Add(item)
	}

	for rows.Next() {
		var item et.Json
		item.ScanRows(rows)

		if len(item) == 1 {
			for _, v := range item {
				switch val := v.(type) {
				case et.Json:
					append(val)
				case map[string]interface{}:
					append(et.Json(val))
				default:
					append(item)
				}
			}
		} else {
			append(item)
		}
	}

	return result
}
