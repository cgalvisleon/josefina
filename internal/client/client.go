package client

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/josefina/internal/cli"
	"github.com/cgalvisleon/josefina/internal/jsql"
)

// Starter is the common interface for both local and TCP REPL modes.
type Starter interface {
	Start()
}

type localStarter struct{ c *cli.CLI }

func (s *localStarter) Start() { s.c.Start() }

type tcpStarter struct{ c *jsql.Client }

func (s *tcpStarter) Start() { s.c.Start() }

type Service struct {
	impl Starter
}

/**
* New: creates a local or remote client depending on whether HOST is set.
* HOST set  → TCP mode: connects to a running jsql server.
* HOST empty → local mode: loads the jdb engine directly.
* @return (*Service, error)
**/
func New() (*Service, error) {
	host := envar.GetStr("HOST", "")
	username := envar.GetStr("USERNAME", "admin")
	password := envar.GetStr("PASSWORD", "")
	database := envar.GetStr("DATABASE", "")

	if host != "" {
		if username == "" {
			return nil, errors.New("username is required for TCP mode")
		}
		c, err := jsql.NewClient(host, username, database)
		if err != nil {
			return nil, fmt.Errorf("TCP connect to %s: %w", host, err)
		}
		return &Service{impl: &tcpStarter{c: c}}, nil
	}

	dataPath := envar.GetStr("DATA_PATH", "./data")
	console, err := cli.New(cli.NodeParams{
		DataPath: dataPath,
		Username: username,
		Password: password,
		Database: database,
	})
	if err != nil {
		return nil, err
	}
	return &Service{impl: &localStarter{c: console}}, nil
}

/**
* Start
**/
func (s *Service) Start() {
	s.impl.Start()
}
