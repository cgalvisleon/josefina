package stmt

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/jdb"
)

type Jql struct {
	query   *catalog.Model
	command *jdb.Wheres
}

func ParseJson(input et.Json) ([]Stmt, error) {
	return []Stmt{}, nil
}
