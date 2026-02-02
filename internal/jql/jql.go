package jql

import (
	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/core"
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
	token = utility.PrefixRemove("Bearer", token)
	s.Token = token
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

/**
* jQlAuthenticate: Authenticates a user
* @param next Handler
* @return Handler
**/
func jQlAuthenticate(next Handler) Handler {
	return HandlerFunc(func(request *Request, response *Response) {
		token := request.Token
		result, err := core.Authenticate(token)
		if err != nil {
			errorResponse(&msg.ERROR_CLIENT_NOT_AUTHENTICATION, response)
			return
		}

		request.Set("app", result.App)
		request.Set("device", result.Device)
		request.Set("username", result.Username)
		next.Execute(request, response)
	})
}

func init() {
	jqlHandler = &JqlHandler{
		middleware: []MiddlewareFunc{
			jQlAuthenticate,
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
	response.Add(request.Json)
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
* Query: Executes a query
* @param session *claim.Claim, query string
* @return et.Json
**/
func Query(session *claim.Claim, query string) et.Json {
	return et.Json{}
}
