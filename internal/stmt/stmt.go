package stmt

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
)

type Stmt interface {
	stmt()
	ToJson() (et.Json, error)
}

type BaseStmt struct{}

func (b BaseStmt) ToJson() (et.Json, error) {
	bt, err := json.Marshal(b)
	if err != nil {
		return et.Json{}, err
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

type CreateUserStmt struct {
	BaseStmt
	Username string
	Password string
}

func (CreateUserStmt) stmt() {}

type GetUserStmt struct {
	BaseStmt
	Username string
	Password string
}

func (GetUserStmt) stmt() {}

type DropUserStmt struct {
	BaseStmt
	Username string
	Password string
}

func (DropUserStmt) stmt() {}

type ChangePasswordStmt struct {
	BaseStmt
	Username    string
	OldPassword string
	NewPassword string
}

func (ChangePasswordStmt) stmt() {}

type CreateDbStmt struct {
	BaseStmt
	Name string
}

func (CreateDbStmt) stmt() {}

type GetDbStmt struct {
	BaseStmt
	Name string
}

func (GetDbStmt) stmt() {}

type DropDbStmt struct {
	BaseStmt
	Name string
}

func (DropDbStmt) stmt() {}

type UseDbStmt struct {
	BaseStmt
	Name string
}

func (UseDbStmt) stmt() {}

type CreateSerieStmt struct {
	BaseStmt
	Tag    string
	Format string
	Value  int
}

func (CreateSerieStmt) stmt() {}

type SetSerieStmt struct {
	BaseStmt
	Tag   string
	Value int
}

func (SetSerieStmt) stmt() {}

type GetSerieStmt struct {
	BaseStmt
	Tag string
}

func (GetSerieStmt) stmt() {}

type DropSerieStmt struct {
	BaseStmt
	Tag string
}

func (DropSerieStmt) stmt() {}
