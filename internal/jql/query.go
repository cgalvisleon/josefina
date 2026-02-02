package jql

import "github.com/cgalvisleon/et/et"

type Query struct{}

func toQuery(query et.Json) (*Query, error) {
	return &Query{}, nil
}

func (s *Query) run() (et.Items, error) {
	return et.Items{}, nil
}
