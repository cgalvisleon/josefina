package jdb

import (
	"errors"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
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
	ID         string        `json:"id"`
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
		"id":          s.ID,
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
* setSession: Sets a session in the cache
* @param session *Session
* @return error
**/
func (s *Sessions) setSession(session *Session) error {
	s.mu.Lock()
	s.sessions[session.ID] = session
	s.mu.Unlock()
	return nil
}

/**
* getSession: Gets a session from the cache
* @param id string
* @return *Session, error
**/
func (s *Sessions) getSession(id string) (*Session, error) {
	s.mu.RLock()
	session, exists := s.sessions[id]
	s.mu.RUnlock()
	if !exists {
		return nil, errors.New(msg.MSG_SESSION_NOT_FOUND)
	}

	return session, nil
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
