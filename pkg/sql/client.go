package sql

import (
	"github.com/cgalvisleon/et/tcp"
	"github.com/cgalvisleon/josefina/pkg/cli"
)

type Client struct {
	node     *tcp.Client
	cli      *cli.Console
	user     string
	password string
	database string
}

func NewClient(host, user, password, database string) *Client {
	result := &Client{
		node:     tcp.NewClient(host),
		user:     user,
		password: password,
		database: database,
	}
	// result.cli = cli.NewConsole(result.node)

	return result
}

func (s *Client) Start() {
	if s.node != nil {
		s.node.Start()
	}

	go s.cli.Start()
}
