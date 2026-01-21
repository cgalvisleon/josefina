package rds

import "github.com/cgalvisleon/et/et"

type Follow struct{}

func (s *Follow) Select(require et.Json, response *et.Item) error {
	return nil
}
