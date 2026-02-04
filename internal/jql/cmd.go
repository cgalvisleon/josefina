package jql

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/catalog"
)

type Cmd struct {
	address string
	command Command
	model   *catalog.Model
	where   *catalog.Wheres
	data    []et.Json
	new     et.Json
	tx      *catalog.Tx
	isDebug bool
}

func newCmd(command Command, model *catalog.Model) *Cmd {
	return &Cmd{
		address: model.Address,
		command: command,
		model:   model,
		where:   &catalog.Wheres{},
		data:    make([]et.Json, 0),
		new:     et.Json{},
	}
}

func toCmd(cmd et.Json) (*Cmd, error) {
	return &Cmd{}, nil
}

func (s *Cmd) toJson() et.Json {
	return et.Json{}
}

func (s *Cmd) debug() *Cmd {
	s.isDebug = true
	return s
}

func (s *Cmd) run(tx *catalog.Tx) (et.Items, error) {
	tx, _ = catalog.GetTx(tx)
	return et.Items{}, nil
}
