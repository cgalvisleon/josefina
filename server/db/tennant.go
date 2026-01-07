package db

import (
	"fmt"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/server/msg"
)

type Tennant struct {
	Name string         `json:"name"`
	Path string         `json:"path"`
	Dbs  map[string]*DB `json:"dbs"`
}

func NewTennant(name string) (*Tennant, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	return &Tennant{
		Name: name,
		Path: fmt.Sprintf("%s/%s", tennant.Path, name),
		Dbs:  make(map[string]*DB),
	}, nil
}

var tennant *Tennant

func init() {
	name := envar.GetStr("TENNANT_NAME", "josefina")
	tennant, _ = NewTennant(name)
}
