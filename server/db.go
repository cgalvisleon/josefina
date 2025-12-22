package server

type DB struct {
	Name    string            `json:"name"`
	Version int               `json:"version"`
	Release int               `json:"release"`
	Models  map[string]*Model `json:"models"`
}

type DBS map[string]*DB

var dbs DBS

func init() {
	dbs = make(DBS)
}

/**
* Start
* @return error
**/
func Start() error {
	return nil
}
