package client

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/sql"
)

var (
	app = "josefina"
)

type Service struct {
	node *sql.Client
}

func New() *Service {
	user := envar.GetStr("USER", "admin")
	password := envar.GetStr("PASSWORD", "admin")
	host := envar.GetStr("HOST", "localhost:1377")
	database := envar.GetStr("DATABASE", "josefina")
	result := &Service{
		node: sql.NewClient(host, user, password, database),
	}

	return result
}

/**
* Start
* @return
**/
func (s *Service) Start() {
	s.node.Start()

	utility.AppWait()

	s.onClose()
}

/**
* onClose
* @return
**/
func (s *Service) onClose() {

}
