package jql

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/stmt"
)

func Query(query string) ([]et.Json, error) {
	stmts, err := stmt.ParseText(query)
	if err != nil {
		return []et.Json{}, err
	}

	result := []et.Json{}
	res := func(item et.Json, err error) ([]et.Json, error) {
		if err != nil {
			return nil, err
		}

		result = append(result, item)
		return result, nil
	}

	for _, st := range stmts {
		item, err := st.ToJson()
		res(item, err)
		if err != nil {
			return nil, err
		}
		// switch s := st.(type) {
		// case stmt.CreateDbStmt:
		// 	_, err := core.CreateDb(s.Name)
		// 	if err != nil {
		// 		return res(et.Json{}, err)
		// 	}

		// 	_, err = res(et.Json{
		// 		"message": "Database created successfully",
		// 	}, nil)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// default:
		// 	return res(et.Json{}, fmt.Errorf(msg.MSG_UNSUPPORTED_STATEMENT_TYPE, st))
		// }
	}

	return result, nil
}
