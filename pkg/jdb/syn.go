package jdb

import (
	"encoding/gob"
	"time"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/mem"
	"github.com/cgalvisleon/josefina/internal/core"
)

func init() {
	gob.Register(time.Time{})
	gob.Register(et.Json{})
	gob.Register([]et.Json{})
	gob.Register(et.Item{})
	gob.Register(et.Items{})
	gob.Register(et.List{})
	gob.Register(claim.Claim{})
	gob.Register(core.Session{})
	gob.Register(RequestVoteArgs{})
	gob.Register(RequestVoteReply{})
	gob.Register(HeartbeatArgs{})
	gob.Register(HeartbeatReply{})
	gob.Register(mem.Item{})
	gob.Register(Client{})
}

type Nodes struct{}

var syn *Nodes

func init() {
	syn = &Nodes{}
}
