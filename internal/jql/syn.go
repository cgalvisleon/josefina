package jql

import (
	"encoding/gob"
	"fmt"
	"os"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
)

type Jql struct{}

var (
	syn      *Jql
	hostname string
)

func init() {
	gob.Register(Ql{})
	gob.Register(Cmd{})

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
* exec: Executes a comand
* @params to string, query et.Json
* @return error
**/
func (s *Jql) exec(cmd Cmd) (et.Items, error) {
	var response et.Items
	err := jrpc.CallRpc(cmd.host, "Jql.Exec", cmd, &response)
	if err != nil {
		return et.Items{}, err
	}

	return response, nil
}

/**
* Exec: Executes a comand
* @param require et.Json, response *et.Items
* @return error
**/
func (s *Jql) Exec(require Cmd, response et.Items) error {
	response = et.Items{}
	return nil
}

/**
* run: Executes a query
* @params to string, query et.Json
* @return error
**/
func (s *Jql) run(ql Ql) (et.Items, error) {
	var response et.Items
	err := jrpc.CallRpc(ql.host, "Jql.Run", ql, &response)
	if err != nil {
		return et.Items{}, err
	}

	return response, nil
}

/**
* Run: Executes a query
* @param require et.Json, response *et.Items
* @return error
**/
func (s *Jql) Run(require Ql, response et.Items) error {
	response = et.Items{}
	return nil
}
