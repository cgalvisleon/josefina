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

func New() (*Service, error) {
	user := envar.GetStr("USER", "admin")
	password := envar.GetStr("PASSWORD", "admin")
	host := envar.GetStr("HOST", "localhost:1377")
	database := envar.GetStr("DATABASE", "josefina")
	client, err := sql.NewClient(host, user, password, database)
	if err != nil {
		return nil, err
	}

	result := &Service{
		node: client,
	}

	return result, nil
}

/**
* Start
* @return
**/
func (s *Service) Start() {
	go s.node.Start()

	utility.AppWait()

	s.onClose()
}

/**
* onClose
* @return
**/
func (s *Service) onClose() {

}
