package jql

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/dbs"
)

type Command string

const (
	SELECT Command = "SELECT"
	INSERT Command = "INSERT"
	UPDATE Command = "UPDATE"
	DELETE Command = "DELETE"
	UPSERT Command = "UPSERT"
)

type Cmd struct {
	address      string
	command      Command
	model        *dbs.Model
	where        *dbs.Wheres
	data         []et.Json
	new          et.Json
	beforeInsert map[string]dbs.Trigger
	beforeUpdate map[string]dbs.Trigger
	beforeDelete map[string]dbs.Trigger
	afterInsert  map[string]dbs.Trigger
	afterUpdate  map[string]dbs.Trigger
	afterDelete  map[string]dbs.Trigger
	tx           *dbs.Tx
	isDebug      bool
}

func newCmd(command Command) *Cmd {
	return &Cmd{
		command: command,
	}
}

func toCmd(cmd et.Json) (*Cmd, error) {
	return &Cmd{}, nil
}

func (s *Cmd) run() (et.Items, error) {
	return et.Items{}, nil
}
