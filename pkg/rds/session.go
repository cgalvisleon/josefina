package rds

import (
	"github.com/cgalvisleon/et/et"
)

type Session struct {
	Id       string `json:"id"`
	Username string `json:"username"`
	Expiry   int64  `json:"expiry"`
}

func (s *Session) Insert(model *Model, new et.Json) (et.Json, error) {
	return model.Insert(nil, new)
}
