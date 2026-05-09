package stmt

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/cgalvisleon/et/et"
)

func ToJson(v any) (et.Json, error) {
	bt, err := json.Marshal(v)
	if err != nil {
		return et.Json{}, err
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}, err
	}

	tipoStruct := reflect.TypeOf(v)
	structName := tipoStruct.String()
	list := strings.Split(structName, ".")
	structName = list[len(list)-1]

	result["type"] = structName
	return result, nil
}

type Stmt interface {
	stmt()
}
