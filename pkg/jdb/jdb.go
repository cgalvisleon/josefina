package jdb

import (
	"encoding/gob"
	"os"
	"time"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/mem"
	"github.com/cgalvisleon/josefina/internal/cache"
	"github.com/cgalvisleon/josefina/internal/core"
	"github.com/cgalvisleon/josefina/internal/jql"
	"github.com/cgalvisleon/josefina/internal/mod"
)

var (
	appName string = "josefina"
	version string = "0.0.1"
	node    *Node
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

/**
* Load: Initializes josefine
* @return error
**/
func Load() error {
	if node != nil {
		return nil
	}

	err := mod.Load(node.getLeader)
	if err != nil {
		return err
	}

	err = core.Load(node.getLeader)
	if err != nil {
		return err
	}

	err = cache.Load(node.getLeader)
	if err != nil {
		return err
	}

	err = jql.Load(node.getLeader, node.nextHost, node.isStrict)
	if err != nil {
		return err
	}

	hostname, _ := os.Hostname()
	port := envar.GetInt("RPC_PORT", 4200)
	isStrict := envar.GetBool("IS_STRICT", false)
	node = newNode(hostname, port, isStrict)

	err = node.mount(node)
	if err != nil {
		return err
	}

	go node.start()

	return nil
}

/**
* HelpCheck: Returns the help check
* @return et.Item
**/
func HelpCheck() et.Item {
	if !node.started {
		return et.Item{
			Ok: false,
			Result: et.Json{
				"status":  false,
				"message": "josefina is not started",
			},
		}
	}

	return et.Item{
		Ok:     true,
		Result: node.toJson(),
	}
}
