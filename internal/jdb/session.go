package jdb

import (
	"errors"
	"sync"
	"time"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/timezone"
	"github.com/josefina/internal/msg"
	"github.com/josefina/internal/store"
)

var (
	ErrorSessionNotFound = errors.New(msg.MSG_SESSION_NOT_FOUND)
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
	CreatedAt   time.Time     `json:"created_at"`
	LastAccess  time.Time     `json:"last_access"`
	Duration    time.Duration `json:"duration"`
	Token       string        `json:"token"`
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	DB          *DB           `json:"db"`
	Type        TpConnection  `json:"type"`
	Permissions []Cmd         `json:"permissions"`
	Payload     et.Json       `json:"payload"`
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
		"id":          s.ID,
		"name":        s.Name,
		"database":    s.DB.Name,
		"type":        s.Type,
		"permissions": s.Permissions,
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
	return timezone.Now().After(s.CreatedAt.Add(s.Duration))
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
	mu       sync.RWMutex
	store    *store.FileStore
}

/**
* add: Adds a session to the cache
* @param session *Session
* @return error
**/
func (s *Sessions) add(session *Session) {
	s.mu.Lock()
	s.sessions[session.Token] = session
	s.mu.Unlock()
}

/**
* get: Gets a session from the cache
* @param token string
* @return *Session, bool
**/
func (s *Sessions) get(token string) (*Session, bool) {
	s.mu.RLock()
	session, exists := s.sessions[token]
	s.mu.RUnlock()

	return session, exists
}

/**
* remove: Removes a session from the cache
* @param id string
* @return error
**/
func (s *Sessions) remove(token string) error {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
	return nil
}

/**
* set: Sets a session in the cache and the store
* @param session *Session
* @return error
**/
func (s *Sessions) set(session *Session) error {
	_, _, err := s.store.PutObject(session.Token, session.ToJson())
	if err != nil {
		return err
	}
	s.add(session)
	return nil
}

/**
* load: Loads a session from the store
* @param token string
* @return *Session, error
**/
func (s *Sessions) load(token string) (*Session, error) {
	result, exists := s.get(token)
	if !exists {
		exists, object, err := s.store.GetObject(token)
		if err != nil {
			return nil, err
		}

		if !exists {
			return nil, ErrorSessionNotFound
		}

		database := object.Str("database")
		db, err := GetDb(database)
		if err != nil {
			return nil, err
		}

		tp := TpConnection(object.Str("type"))
		payload := object.Json("payload")
		permissions := make([]Cmd, len(object.Array("permissions")))
		for i, v := range object.Array("permissions") {
			permissions[i] = Cmd(v.(string))
		}
		result = &Session{
			CreatedAt:   object.Time("created_at"),
			LastAccess:  object.Time("last_access"),
			Duration:    object.ValDuration(0, "duration"),
			Token:       token,
			ID:          object.Str("id"),
			Name:        object.Str("name"),
			DB:          db,
			Type:        tp,
			Permissions: permissions,
			Payload:     payload,
		}
	}

	return result, nil
}

/**
* delete: Deletes a session from the store
* @param token string
* @return error
**/
func (s *Sessions) delete(token string) error {
	_, err := s.store.Delete(token)
	if err != nil {
		return err
	}
	return s.remove(token)
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
func (s *Server) newSession(token string, db *DB, tp TpConnection, payload et.Json) (*Session, error) {
	clm, err := claim.ParceToken(token)
	if err != nil {
		return nil, err
	}

	now := timezone.Now()
	result := &Session{
		CreatedAt:   now,
		LastAccess:  now,
		Duration:    clm.Duration,
		Token:       token,
		ID:          clm.SessionID,
		Name:        clm.Name,
		DB:          db,
		Type:        tp,
		Permissions: []Cmd{QUERY, INSERT, UPDATE, DELETE, UPSERT, BULK},
		Payload:     payload,
	}

	s.sessions.set(result)

	return result, nil
}

/**
* getSession: Gets a session from the cache
* @param token string
* @return *Session, error
**/
func (s *Server) getSession(token string) (*Session, error) {
	result, err := s.sessions.load(token)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* deleteSession: Deletes a session from the server
* @param token string
* @return error
**/
func (s *Server) deleteSession(token string) error {
	return s.sessions.delete(token)
}
