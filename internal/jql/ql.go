package jql

import "github.com/cgalvisleon/et/et"

type Ql struct {
	host string
}

/**
* getQl: Returns a Ql
* @param query et.Json
* @return *Ql, error
**/
func getQl(query et.Json) (*Ql, error) {
	return &Ql{}, nil
}

/**
* run: Runs the query
* @return et.Items, error
**/
func (s *Ql) run() (et.Items, error) {
	return et.Items{}, nil
}
