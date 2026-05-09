package client

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/josefina/internal/cli"
)

type Service struct {
	console *cli.CLI
}

/**
* New
* @return (*Service, error)
**/
func New() (*Service, error) {
	dataPath := envar.GetStr("DATA_PATH", "./data")
	username := envar.GetStr("USERNAME", "admin")
	password := envar.GetStr("PASSWORD", "")
	database := envar.GetStr("DATABASE", "")

	console, err := cli.New(cli.NodeParams{
		DataPath: dataPath,
		Username: username,
		Password: password,
		Database: database,
	})
	if err != nil {
		return nil, err
	}

	return &Service{console: console}, nil
}

/**
* Start
**/
func (s *Service) Start() {
	s.console.Start()
}
