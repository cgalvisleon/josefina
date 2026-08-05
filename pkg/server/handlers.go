package server

import (
	"github.com/cgalvisleon/et/et"
	"github.com/josefina/internal/jdb"
)

/**
* signin: Signs in a user
* @param database, username, password string
* @return et.Item, error
**/
func signin(database, username, password string) (et.Item, error) {
	result, err := jdb.SignIn(database, username, password)
	if err != nil {
		return et.Item{}, err
	}
	return result, nil
}

/**
* query
* @param database, query string
* @return et.Item, error
**/
func jquery(query et.Json) (et.Items, error) {
	result, err := jdb.JQuery(query)
	if err != nil {
		return et.Items{}, err
	}
	return result, nil
}
