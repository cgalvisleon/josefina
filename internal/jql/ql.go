package jql

import "github.com/cgalvisleon/et/et"

type Ql struct {
	address string
}

/**
* ToQl: Returns a Ql
* @param query et.Json
* @return *Ql, error
**/
func ToQl(query et.Json) (*Ql, error) {
	command, err := getCommand(query)
	if err != nil {
		return nil, err
	}

	switch command {
	case SELECT:
		return &Ql{}, nil
	case INSERT, UPDATE, DELETE, UPSERT:
		return &Ql{}, nil
	}

	return &Ql{}, nil
}

/**
* Run: Runs the query
* @return et.Items, error
**/
func (s *Ql) Run() (et.Items, error) {
	return et.Items{}, nil
}
