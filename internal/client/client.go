package client

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/sql"
)

type Service struct {
	node *sql.Client
}

func New() (*Service, error) {
	host := envar.GetStr("HOST", "localhost:1377")
	username := envar.GetStr("USERNAME", "admin")
	database := envar.GetStr("DATABASE", "josefina")
	client, err := sql.NewClient(host, username, database)
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
