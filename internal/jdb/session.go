package jdb

import (
	"errors"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/timezone"
	"github.com/josefina/internal/msg"
	"github.com/josefina/internal/store"
)

/**
* TpConnection: Identifies the transport protocol used by a client session.
**/
type TpConnection string

const (
	HTTP      TpConnection = "http"
	WebSocket TpConnection = "websocket"
	TCP       TpConnection = "tcp"
)

/**
* Device: Classifies the kind of client connecting to the node.
**/
type Device string

const (
	TpApi    Device = "api"
	TpClient Device = "client"
	TpNode   Device = "node"
)

/**
* Session: Represents an authenticated client connection with its JWT token cached for the duration.
**/
type Session struct {
	CreatedAt  time.Time     `json:"created_at"`
	LastAccess time.Time     `json:"last_access"`
	Duration   time.Duration `json:"duration"`
	Token      string        `json:"token"`
	Name       string        `json:"name"`
	Database   *DB           `json:"database"`
	UserId     string        `json:"user_id"`
	Type       TpConnection  `json:"type"`
	Payload    et.Json       `json:"payload"`
}

/**
* ToJson: Converts the session to a JSON object
* @return et.Json
**/
func (s *Session) ToJson() et.Json {
	return et.Json{
		"created_at":  s.CreatedAt.Format(time.RFC3339),
		"last_access": s.LastAccess.Format(time.RFC3339),
		"duration":    s.Duration,
		"token":       s.Token,
		"name":        s.Name,
		"database":    s.Database.Name,
		"user_id":     s.UserId,
		"type":        s.Type,
		"payload":     s.Payload,
	}
}

/**
* IsExpired
* @return bool
**/
func (s *Session) IsExpired() bool {
	if s.Duration == 0 {
		return false
	}
	return time.Now().After(s.CreatedAt.Add(s.Duration))
}

/**
* GetExpiresAt
* @return time.Time
**/
func (s *Session) GetExpiresAt() time.Time {
	return s.CreatedAt.Add(s.Duration)
}

type Sessions struct {
	sessions map[string]*Session
	mu       *sync.RWMutex
	store    *store.FileStore
}

/**
* addSession: Adds a session to the cache
* @param session *Session
* @return error
**/
func (s *Sessions) setSession(session *Session) {
	s.mu.Lock()
	s.sessions[session.Token] = session
	s.mu.Unlock()
}

/**
* getSession: Gets a session from the cache
* @param token string
* @return *Session, error
**/
func (s *Sessions) getSession(token string) (*Session, bool) {
	s.mu.RLock()
	session, exists := s.sessions[token]
	s.mu.RUnlock()

	if exists {
		session.LastAccess = timezone.Now()
		s.setSession(session)
	}

	return session, exists
}

/**
* removeSession: Removes a session from the cache
* @param id string
* @return error
**/
func (s *Sessions) removeSession(id string) error {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
	return nil
}

/**
* loadSessions: Loads the sessions
* @return error
**/
func (s *Server) loadSessions() error {
	store, err := store.Open(s.PathSystem, s.PathSystem, "sessions", store.ReadWrite)
	if err != nil {
		return err
	}

	result := &Sessions{
		sessions: make(map[string]*Session),
		mu:       &sync.RWMutex{},
		store:    store,
	}

	s.sessions = result

	return nil
}

/**
* newSession: Creates a new session
* @param token string
* @return *Session, error
**/
func (s *Server) newSession(token string, database *DB, userId string, name string, tp TpConnection, payload et.Json) (*Session, error) {
	now := timezone.Now()
	result := &Session{
		CreatedAt:  now,
		LastAccess: now,
		Duration:   0,
		Token:      token,
		Name:       name,
		Database:   database,
		UserId:     userId,
		Type:       tp,
		Payload:    payload,
	}

	s.sessions.setSession(result)

	return result, nil
}

/**
* getSession: Gets a session from the cache
* @param token string
* @return *Session, error
**/
func (s *Server) getSession(token string) (*Session, error) {
	result, exists := s.sessions.getSession(token)
	if !exists {
		return nil, errors.New(msg.MSG_SESSION_NOT_FOUND)
	}

	return result, nil
}
