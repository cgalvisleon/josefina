package jdb

import "github.com/cgalvisleon/et/et"

type HandlerFunc func(et.Json, et.Json)

/**
* Execute
* @param request et.Json, response et.Json
**/
func (s HandlerFunc) Execute(request et.Json, response et.Json) {
	s(request, response)
}

type Handler interface {
	Execute(et.Json, et.Json)
}

/**
* jqlIsExisted
* @param to *From, field string, key string
* @return (bool, error)
**/
func jqlIsExisted(to *From, field, key string) (bool, error) {
	return false, nil
}

/**
* query
* @param token string, jqls []et.Json
* @return (et.Json, error)
**/
func query(token string, jqls []et.Json) (et.Json, error) {
	return et.Json{}, nil
}
