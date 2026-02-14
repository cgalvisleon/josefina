package client

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/cli"
	"github.com/cgalvisleon/josefina/pkg/jdb"
)

var (
	app = "josefina"
)

type Service struct {
	cli *cli.Console
}

func New() *Service {
	user := envar.GetStr("USER", "admin")
	// password := envar.GetStr("PASSWORD", "admin")
	host := envar.GetStr("HOST", "localhost:1377")
	database := envar.GetStr("DATABASE", "josefina")
	session := jdb.NewSession(user, host, jdb.TCP, database)

	cli := cli.NewConsole(session)
	return &Service{
		cli: cli,
	}
}

/**
* Start
* @return
**/
func (s *Service) Start() {
	go s.cli.Start()

	utility.AppWait()

	s.onClose()
}

/**
* onClose
* @return
**/
func (s *Service) onClose() {

}
