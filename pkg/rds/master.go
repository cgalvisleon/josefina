package rds

import "github.com/cgalvisleon/et/et"

type Master struct{}

func (s *Master) CreateDatabase(require et.Json, response *et.Item) error {
	return nil
}
