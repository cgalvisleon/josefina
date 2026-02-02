package jql

import (
	"encoding/gob"
	"fmt"
	"os"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/mem"
)

type Jql struct{}

type getLeaderFn func() (string, bool)

var (
	syn       *Jql
	hostname  string
	getLeader getLeaderFn
)

func init() {
	gob.Register(mem.Item{})

	hostname, _ = os.Hostname()
	port := envar.GetInt("RPC_PORT", 4200)
	hostname = fmt.Sprintf("%s:%d", hostname, port)

	syn = &Jql{}
	_, err := jrpc.Mount(hostname, syn)
	if err != nil {
		logs.Panic(err)
	}
}

/**
* Load: Loads the cache
* @param fn getLeaderFn
* @return error
**/
func Load(fn getLeaderFn) error {
	getLeader = fn
	return nil
}

/**
* query: Executes a query
* @params to string, query et.Json
* @return error
**/
func (s *Jql) jquery(to string, query et.Json) (et.Items, error) {
	var response et.Items
	err := jrpc.CallRpc(to, "Jql.Jquery", query, &response)
	if err != nil {
		return et.Items{}, err
	}

	return response, nil
}

/**
* Jquery: Sets a cache value
* @param require et.Json, response *et.Items
* @return error
**/
func (s *Jql) Jquery(require et.Json, response et.Items) error {
	response = et.Items{}
	return nil
}
