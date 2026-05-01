package jdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jwt"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type Status string

const ()

type TpConnection string

const (
	HTTP      TpConnection = "http"
	WebSocket TpConnection = "websocket"
	TCP       TpConnection = "tcp"
)

type Device string

const (
	TpApi    Device = "api"
	TpClient Device = "client"
	TpNode   Device = "node"
)

type Session struct {
	CreatedAt  time.Time     `json:"created_at"`
	LastAccess time.Time     `json:"last_access"`
	Duration   time.Duration `json:"duration"`
	Idx        string        `json:"idx"`
	UserId     string        `json:"user_id"`
	Username   string        `json:"username"`
	Address    string        `json:"address"`
	App        string        `json:"app"`
	Type       TpConnection  `json:"type"`
	Payload    et.Json       `json:"payload"`
	Database   string        `json:"database"`
	db         *DB           `json:"-"`
	node       *Node         `json:"-"`
}

/**
* newSession
* @param db *DB, userId, username, address, app string, tp TpConnection, duration time.Duration
* @return string, err
**/
func newSession(db *DB, userId, username, address string, tp TpConnection, payload et.Json, duration time.Duration) (string, error) {
	key := fmt.Sprintf("%s:%s", userId, db.Name)
	err := db.DeleteCache(key)
	if err != nil {
		return "", err
	}

	idx := db.node.sessions.GenKey()
	result := &Session{
		CreatedAt: time.Now(),
		Duration:  duration,
		Idx:       idx,
		UserId:    userId,
		Username:  username,
		Address:   address,
		Type:      tp,
		Database:  db.Name,
		db:        db,
		node:      db.node,
	}

	token, err := jwt.New(db.Name, idx, userId, username, payload, duration)
	if err != nil {
		return "", err
	}

	err = db.SetCache(key, token, duration)
	if err != nil {
		return "", err
	}

	if err := result.save(); err != nil {
		return "", err
	}

	return token, nil
}

/**
* save
* @return error
**/
func (s *Session) save() error {
	if s.node == nil {
		return errors.New(msg.MSG_NODE_IS_NIL)
	}

	jData, err := s.ToJson()
	if err != nil {
		return err
	}

	err = s.node.sessions.Put(s.Idx, jData)
	if err != nil {
		return err
	}

	return nil
}

/**
* ToJson
* @return (et.Json, error)
**/
func (s *Session) ToJson() (et.Json, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return et.Json{}, err
	}

	result := et.Json{}
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
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

/**
* loadSessions: Load the sessions
* @return error
**/
func (s *Node) loadSessions() error {
	var err error
	s.sessions, err = s.catalog.Define(Define{
		Name:    "sessions",
		IsCore:  true,
		Version: 1,
	})
	if err != nil {
		return err
	}

	if err = s.sessions.Init(); err != nil {
		return err
	}

	return nil
}
