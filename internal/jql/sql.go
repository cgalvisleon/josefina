package jql

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/core"
	"github.com/cgalvisleon/josefina/internal/msg"
	"github.com/cgalvisleon/josefina/internal/stmt"
)

func Sql(query string) ([]et.Item, error) {
	st, err := stmt.ParseText(query)
	if err != nil {
		return []et.Item{}, err
	}

	result := []et.Item{}
	res := func(item et.Json, err error) []et.Item {
		if err != nil {
			result = append(result, et.Item{
				Ok: false,
				Result: et.Json{
					"message": err.Error(),
				},
			})
		} else {
			result = append(result, et.Item{
				Ok:     true,
				Result: item,
			})
		}
		return result
	}

	switch s := st.(type) {
	case *stmt.CreateDbStmt:
		_, err := core.CreateDb(s.Name)
		if err != nil {
			return res(et.Json{}, err), err
		}

		return res(et.Json{
			"message": "Database created successfully",
		}, nil), nil
	default:
		return res(et.Json{}, fmt.Errorf(msg.MSG_UNSUPPORTED_STATEMENT_TYPE, st)), fmt.Errorf(msg.MSG_UNSUPPORTED_STATEMENT_TYPE, st)
	}
}
