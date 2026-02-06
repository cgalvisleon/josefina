package stmt

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/catalog"
)

type Jql struct {
	query   *catalog.Model
	command *catalog.Wheres
}

func ParseJson(input et.Json) ([]Stmt, error) {
	return []Stmt{}, nil
}
