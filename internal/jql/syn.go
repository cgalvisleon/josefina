package jql

import (
	"encoding/gob"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/catalog"
)

type Jql struct {
	getLeader func() (string, bool)
	address   string
	isStrict  bool
}

var (
	syn *Jql
)

func init() {
	gob.Register(Ql{})
	gob.Register(Cmd{})
	syn = &Jql{}
}

func (j *Jql) LoadModel(require et.Json, response *catalog.Model) error {

	return nil
}
