package db

type Tenant struct {
	Name string         `json:"name"`
	Path string         `json:"path"`
	Dbs  map[string]*DB `json:"dbs"`
}

var tennant *Tenant

func init() {
	tennant = &Tenant{
		Name: "",
		Path: "/data",
		Dbs:  make(map[string]*DB),
	}
}
