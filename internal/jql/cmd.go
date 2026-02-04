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
