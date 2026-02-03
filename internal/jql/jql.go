package jql

import db "github.com/cgalvisleon/josefina/internal/dbs"

type Jql struct{}

var (
	dbs       map[string]*db.DB
	getLeader getLeaderFn
)

func init() {
	dbs = make(map[string]*db.DB, 0)
}

/**
* Load: Loads the cache
* @param fn getLeaderFn
* @return error
**/
func Load(fn getLeaderFn) error {
	getLeader = fn
	return nil
}
