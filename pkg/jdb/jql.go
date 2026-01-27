package jdb

import (
	"strings"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
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
	Error  *msg.MessageError `json:"error"`
	Result []et.Json         `json:"result"`
}

/**
* Add: Adds an item to the result
* @param item et.Json
**/
func (s *Response) Add(item et.Json) {
	s.Result = append(s.Result, item)
}

type HandlerFunc func(*Request, *Response)
type MiddlewareFunc func(Handler) Handler

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
func errorResponse(msg *msg.MessageError, response *Response) {
	response.Error = msg
}

type JqlHandler struct {
	middleware []MiddlewareFunc
}

var jqlHandler *JqlHandler

func init() {
	jqlHandler = &JqlHandler{
		middleware: []MiddlewareFunc{
			authenticate,
		},
	}
}

/**
* applyMiddleware
* @param handler Handler
* @return Handler
**/
func (s *JqlHandler) applyMiddleware(handler Handler) Handler {
	for _, middleware := range s.middleware {
		handler = middleware(handler)
	}
	return handler
}

/**
* Execute
* @param request *Request, response *Response
**/
func (s *JqlHandler) Execute(request *Request, response *Response) {
	logs.Ping()
	// jql(request, response)
}

/**
* JqlHttp
* @param request *Request, response *Response
**/
func JqlHttp(request *Request, response *Response) {
	handler := jqlHandler.applyMiddleware(jqlHandler)
	if handler == nil {
		response.Error = &msg.ERROR_INTERNAL_ERROR
		return
	}

	handler.Execute(request, response)
}

/**
* jqlIsExisted
* @param to *From, field string, key string
* @return (bool, error)
**/
func jqlIsExisted(to *From, field, key string) (bool, error) {
	return false, nil
}
