package core

type Node interface {
}

var (
	node     Node
	database string = "josefina"
)

func SetNode(n Node) {
	node = n
}
