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
	CREATE Command = "CREATE"
	DROP   Command = "DROP"
	CHANGE Command = "CHANGE"
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

func newCmd(command Command, model *dbs.Model) *Cmd {
	return &Cmd{
		address:      model.Address,
		command:      command,
		model:        model,
		where:        &dbs.Wheres{},
		data:         make([]et.Json, 0),
		new:          et.Json{},
		beforeInsert: make(map[string]dbs.Trigger, 0),
		beforeUpdate: make(map[string]dbs.Trigger, 0),
		beforeDelete: make(map[string]dbs.Trigger, 0),
		afterInsert:  make(map[string]dbs.Trigger, 0),
		afterUpdate:  make(map[string]dbs.Trigger, 0),
		afterDelete:  make(map[string]dbs.Trigger, 0),
	}
}

func toCmd(cmd et.Json) (*Cmd, error) {
	command, err := getCommand(cmd)
	if err != nil {
		return nil, err
	}
	database := cmd.Str("database")
	schema := cmd.Str("schema")
	name := cmd.Str("name")
	model, err := node.getModel(database, schema, name)
	if err != nil {
		return nil, err
	}
	return &Cmd{}, nil
}

func (s *Cmd) toJson() et.Json {
	return et.Json{}
}

func (s *Cmd) debug() *Cmd {
	s.isDebug = true
	return s
}

func (s *Cmd) run(tx *dbs.Tx) (et.Items, error) {
	tx, _ = dbs.GetTx(tx)
	return et.Items{}, nil
}
