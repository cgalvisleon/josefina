package jql

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/mod"
)

type Cmd struct {
	address string
	command Command
	model   *mod.Model
	where   *mod.Wheres
	data    []et.Json
	new     et.Json
	tx      *mod.Tx
	isDebug bool
}

func newCmd(command Command, model *mod.Model) *Cmd {
	return &Cmd{
		address: model.Address,
		command: command,
		model:   model,
		where:   &mod.Wheres{},
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

func (s *Cmd) run(tx *mod.Tx) (et.Items, error) {
	tx, _ = mod.GetTx(tx)
	return et.Items{}, nil
}
