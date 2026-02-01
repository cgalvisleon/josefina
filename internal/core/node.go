package core

import "github.com/cgalvisleon/josefina/internal/jdb"

type Node interface {
	GetDb(name string) (*jdb.DB, error)
}

var (
	node     Node
	database string = "josefina"
)

func SetNode(n Node) {
	node = n
}
