package jsql

import (
	"errors"
	"sync"

	"github.com/cgalvisleon/et/tcp"
	"github.com/cgalvisleon/josefina/internal/stmt"
)

// QueryArgs is the request payload for the Query.Exec RPC.
type QueryArgs struct {
	SQL       string `json:"sql"`
	Database  string `json:"database"`
	SessionID string `json:"session_id"`
}

// QueryService implements tcp.Service and handles SQL query execution over TCP.
type QueryService struct {
	node *Server
	mu   sync.RWMutex
	sess map[string]*Session
}

/**
* newQueryService
* @param node *Server
* @return *QueryService
**/
func newQueryService(node *Server) *QueryService {
	return &QueryService{
		node: node,
		sess: make(map[string]*Session),
	}
}

/**
* Execute: dispatches incoming TCP method calls for the Query service.
* @param method string
* @param request *tcp.Message
* @return *tcp.Response
**/
func (s *QueryService) Execute(method string, request *tcp.Message) *tcp.Response {
	switch method {
	case "Exec":
		return s.exec(request)
	case "SetDialect":
		return s.setDialect(request)
	case "Connect":
		return s.connect(request)
	default:
		return tcp.TcpError("Query: unknown method " + method)
	}
}

/**
* connect: registers a new session and returns its ID.
* Args: username string, database string, address string
* @param request *tcp.Message
* @return *tcp.Response
**/
func (s *QueryService) connect(request *tcp.Message) *tcp.Response {
	var username, database, address string
	if err := request.GetArgs(&username, &database, &address); err != nil {
		return tcp.TcpError(err)
	}

	session := NewSession(username, address, TCP, database)

	s.mu.Lock()
	s.sess[session.ID] = session
	s.mu.Unlock()

	return tcp.TcpResponse(session.ID)
}

/**
* exec: executes a SQL query and returns the result.
* Args: QueryArgs
* @param request *tcp.Message
* @return *tcp.Response
**/
func (s *QueryService) exec(request *tcp.Message) *tcp.Response {
	var args QueryArgs
	if err := request.GetArgs(&args); err != nil {
		return tcp.TcpError(err)
	}

	if args.SQL == "" {
		return tcp.TcpError(errors.New("sql is required"))
	}

	db, err := s.node.GetDb(args.Database)
	if err != nil {
		return tcp.TcpError(err)
	}

	items, err := stmt.ExecSQL(db, args.SQL)
	if err != nil {
		return tcp.TcpError(err)
	}

	return tcp.TcpResponse(items)
}

/**
* setDialect: updates the SQL dialect for a session.
* Args: sessionID string, dialect string
* @param request *tcp.Message
* @return *tcp.Response
**/
func (s *QueryService) setDialect(request *tcp.Message) *tcp.Response {
	var sessionID, dialectStr string
	if err := request.GetArgs(&sessionID, &dialectStr); err != nil {
		return tcp.TcpError(err)
	}

	s.mu.Lock()
	session, ok := s.sess[sessionID]
	if ok {
		session.SqlDialect = stmt.SqlDialect(dialectStr)
	}
	s.mu.Unlock()

	if !ok {
		return tcp.TcpError(errors.New("session not found"))
	}

	return tcp.TcpResponse("OK")
}
