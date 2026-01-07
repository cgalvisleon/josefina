package model

type DB struct {
	Name    string             `json:"name"`
	Version int                `json:"version"`
	Release int                `json:"release"`
	Schemas map[string]*Schema `json:"schemas"`
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
