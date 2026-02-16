package catalog

import "github.com/cgalvisleon/et/envar"

var (
// node *Node
)

/**
* Load: Loads the cache
* @param app string
* @return error
**/
func Load(app string) error {
	if node != nil {
		return nil
	}

	tpcPort := envar.GetInt("TPC_PORT", 8080)
	node = newNode(tpcPort)

	return nil
}
