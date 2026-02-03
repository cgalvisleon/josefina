package jql

import (
	"encoding/gob"
)

type Jql struct{}

var (
	syn *Jql
)

func init() {
	gob.Register(Ql{})
	gob.Register(Cmd{})
}
