package stmt

import (
	"encoding/json"

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

	return result, nil
}

type Stmt interface {
	stmt()
	ToJson() (et.Json, error)
}

type BaseStmt struct{}

type CreateUserStmt struct {
	BaseStmt
	Username string
	Password string
}

func (CreateUserStmt) stmt() {}
func (s CreateUserStmt) ToJson() (et.Json, error) {
	return ToJson(s)
}

type GetUserStmt struct {
	BaseStmt
	Username string
	Password string
}

func (GetUserStmt) stmt() {}
func (s GetUserStmt) ToJson() (et.Json, error) {
	return ToJson(s)
}

type DropUserStmt struct {
	BaseStmt
	Username string
	Password string
}

func (DropUserStmt) stmt() {}
func (s DropUserStmt) ToJson() (et.Json, error) {
	return ToJson(s)
}

type ChangePasswordStmt struct {
	BaseStmt
	Username    string
	OldPassword string
	NewPassword string
}

func (ChangePasswordStmt) stmt() {}
func (s ChangePasswordStmt) ToJson() (et.Json, error) {
	return ToJson(s)
}

type CreateDbStmt struct {
	BaseStmt
	Name string
}

func (CreateDbStmt) stmt() {}
func (s CreateDbStmt) ToJson() (et.Json, error) {
	return ToJson(s)
}

type GetDbStmt struct {
	BaseStmt
	Name string
}

func (GetDbStmt) stmt() {}
func (s GetDbStmt) ToJson() (et.Json, error) {
	return ToJson(s)
}

type DropDbStmt struct {
	BaseStmt
	Name string
}

func (DropDbStmt) stmt() {}
func (s DropDbStmt) ToJson() (et.Json, error) {
	return ToJson(s)
}

type UseDbStmt struct {
	BaseStmt
	Name string
}

func (UseDbStmt) stmt() {}
func (s UseDbStmt) ToJson() (et.Json, error) {
	return ToJson(s)
}

type CreateSerieStmt struct {
	BaseStmt
	Tag    string
	Format string
	Value  int
}

func (CreateSerieStmt) stmt() {}
func (s CreateSerieStmt) ToJson() (et.Json, error) {
	return ToJson(s)
}

type SetSerieStmt struct {
	BaseStmt
	Tag   string
	Value int
}

func (SetSerieStmt) stmt() {}
func (s SetSerieStmt) ToJson() (et.Json, error) {
	return ToJson(s)
}

type GetSerieStmt struct {
	BaseStmt
	Tag string
}

func (GetSerieStmt) stmt() {}
func (s GetSerieStmt) ToJson() (et.Json, error) {
	return ToJson(s)
}

type DropSerieStmt struct {
	BaseStmt
	Tag string
}

func (DropSerieStmt) stmt() {}
func (s DropSerieStmt) ToJson() (et.Json, error) {
	return ToJson(s)
}
