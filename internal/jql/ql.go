package jql

import "github.com/cgalvisleon/et/et"

type Ql struct {
	host string
}

/**
* ToQl: Returns a Ql
* @param query et.Json
* @return *Ql, error
**/
func ToQl(query et.Json) (*Ql, error) {
	return &Ql{}, nil
}

/**
* Run: Runs the query
* @return et.Items, error
**/
func (s *Ql) Run() (et.Items, error) {
	return et.Items{}, nil
}
