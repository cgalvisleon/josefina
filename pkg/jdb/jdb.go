package jdb

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/mem"
	"github.com/cgalvisleon/josefina/internal/cache"
	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/core"
	"github.com/cgalvisleon/josefina/internal/msg"
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
	gob.Register(mem.Entry{})
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

	hostname, _ := os.Hostname()
	rpcPort := envar.GetInt("RPC_PORT", 4200)
	isStrict := envar.GetBool("IS_STRICT", false)
	node = newNode(hostname, rpcPort, isStrict)

	err := node.mount(node)
	if err != nil {
		return err
	}

	err = catalog.Load(node.getLeader)
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

/**
* authenticate: Authenticates a user
* @param ctx context.Context
* @return error
**/
func authenticate(ctx context.Context) error {
	app := ctx.Value("app").(string)
	device := ctx.Value("device").(string)
	username := ctx.Value("username").(string)
	key := fmt.Sprintf("%s:%s:%s", app, device, username)
	_, exists, err := cache.GetStr(key)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(msg.MSG_CLIENT_NOT_AUTHENTICATION)
	}

	return nil
}

/**
* Query: Executes a query
* @param ctx context.Context, query string
* @return et.Items, error
**/
func Query(ctx context.Context, sql string, args ...any) (et.Items, error) {
	err := authenticate(ctx)
	if err != nil {
		return et.Items{}, err
	}

	items, err := query(sql, args...)
	if err != nil {
		return et.Items{}, err
	}

	result := et.Items{}
	result.Add(items...)
	return result, nil
}

/**
* JQuery: Executes a query
* @param ctx context.Context, query et.Json
* @return et.Items, error
**/
func JQuery(ctx context.Context, query et.Json) (et.Items, error) {
	err := authenticate(ctx)
	if err != nil {
		return et.Items{}, err
	}

	items, err := jquery(query)
	if err != nil {
		return et.Items{}, err
	}

	result := et.Items{}
	result.Add(items...)
	return result, nil
}
