package jdb

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/josefina/internal/core"
	"github.com/cgalvisleon/josefina/internal/msg"
	"github.com/cgalvisleon/josefina/internal/stmt"
)

/**
* quoted
* @param val any
* @return any
**/
func quoted(val any) any {
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
* sqlParse
* @param sql string, args ...any
* @return string
**/
func sqlParse(sql string, args ...any) string {
	for i := range args {
		old := strs.Format(`$%d`, i+1)
		new := strs.Format(`{$%d}`, i+1)
		sql = strings.ReplaceAll(sql, old, new)
	}

	for i, arg := range args {
		old := fmt.Sprintf(`{$%d}`, i+1)
		new := fmt.Sprintf(`%v`, quoted(arg))
		sql = strings.ReplaceAll(sql, old, new)
	}

	return sql
}

/**
* exec
* @param stmts []stmt.Stmt
* @return []et.Json, error
**/
func exec(stmts []stmt.Stmt) ([]et.Json, error) {
	result := []et.Json{}
	res := func(item et.Json, err error) ([]et.Json, error) {
		if err != nil {
			return result, err
		}

		result = append(result, item)
		return result, nil
	}

	for _, st := range stmts {
		switch s := st.(type) {
		case stmt.CreateDbStmt:
			_, err := core.CreateDb(s.Name)
			if err != nil {
				return res(et.Json{}, err)
			}

			_, err = res(et.Json{
				"message": "Database created successfully",
			}, nil)
			if err != nil {
				return nil, err
			}
		default:
			return res(et.Json{}, fmt.Errorf(msg.MSG_UNSUPPORTED_STATEMENT_TYPE, st))
		}
	}

	return result, nil
}

/**
* query
* @param sql string, args ...any
* @return []et.Json, error
**/
func query(sql string, args ...any) ([]et.Json, error) {
	sql = sqlParse(sql, args...)
	stmts, err := stmt.ParseText(sql)
	if err != nil {
		return []et.Json{}, err
	}

	return exec(stmts)
}

/**
* jquery
* @param query et.Json
* @return []et.Json, error
**/
func jquery(query et.Json) ([]et.Json, error) {
	stmts, err := stmt.ParseJson(query)
	if err != nil {
		return []et.Json{}, err
	}

	return exec(stmts)
}
