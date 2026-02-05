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

type CreateUserStmt struct {
	Username string
	Password string
}

func (CreateUserStmt) stmt() {}

type GetUserStmt struct {
	Username string
	Password string
}

func (GetUserStmt) stmt() {}

type DropUserStmt struct {
	Username string
	Password string
}

func (DropUserStmt) stmt() {}

type ChangePasswordStmt struct {
	Username    string
	OldPassword string
	NewPassword string
}

func (ChangePasswordStmt) stmt() {}

type CreateDbStmt struct {
	Name string
}

func (CreateDbStmt) stmt() {}

type GetDbStmt struct {
	Name string
}

func (GetDbStmt) stmt() {}

type DropDbStmt struct {
	Name string
}

func (DropDbStmt) stmt() {}

type UseDbStmt struct {
	Name string
}

func (UseDbStmt) stmt() {}
