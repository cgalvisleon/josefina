package jdb

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

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
* errorResponse: Creates an error response
* @param code string, err error
* @return et.Json
**/
func errorResponse(msg msg.MessageError, err error, response et.Json) {
	response.Set("ok", false)
	response.Set("result", et.Json{
		"error": fmt.Sprintf(`%s - %s`, msg.Message, err.Error()),
		"code":  msg.Code,
	})
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
