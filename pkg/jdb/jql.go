package jdb

import (
	"fmt"
	"strings"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

type Request struct {
	et.Json
	Token string `json:"token"`
}

/**
* SetToken
* @param token string
**/
func (s *Request) SetToken(token string) {
	prefix := envar.GetStr("TOKEN_PREFIX", "Bearer ")
	if strings.HasPrefix(token, prefix) {
		s.Token = strings.TrimPrefix(token, prefix)
	}
}

/**
* SetBody
* @param values et.Json
**/
func (s *Request) SetBody(values et.Json) {
	s.Json = values
}

type Response struct {
	Error  error     `json:"error"`
	Result []et.Json `json:"result"`
}

/**
* Add: Adds an item to the result
* @param item et.Json
**/
func (s *Response) Add(item et.Json) {
	s.Result = append(s.Result, item)
}

type HandlerFunc func(*Request, *Response)

/**
* Execute
* @param request *Request, response *Response
**/
func (s HandlerFunc) Execute(request *Request, response *Response) {
	s(request, response)
}

type Handler interface {
	Execute(*Request, *Response)
}

/**
* errorResponse: Creates an error response
* @param msg msg.MessageError, err error, response *Response
**/
func errorResponse(msg msg.MessageError, err error, response *Response) {
	response.Error = err
	response.Add(et.Json{
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

type JqlHandler struct{}

func (s *JqlHandler) Execute(request *Request, response *Response) {
	Jql(request, response)
}

/**
* Jql
* @param request *Request, response *Response
**/
func Jql(request *Request, response *Response) {
}
