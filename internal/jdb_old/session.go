package jdb

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type Status string

const (
	Active       Status = "active"
	Archived     Status = "archived"
	Canceled     Status = "canceled"
	OfSystem     Status = "of_system"
	ForDelete    Status = "for_delete"
	Pending      Status = "pending"
	Approved     Status = "approved"
	Rejected     Status = "rejected"
	Failed       Status = "failed"
	Processed    Status = "processed"
	Connected    Status = "connected"
	Disconnected Status = "disconnected"
)

type TpConnection string

const (
	HTTP      TpConnection = "http"
	WebSocket TpConnection = "websocket"
	TCP       TpConnection = "tcp"
)

type Session struct {
	CreatedAt time.Time    `json:"created_at"`
	ID        string       `json:"id"`
	Username  string       `json:"username"`
	Address   string       `json:"address"`
	Status    Status       `json:"status"`
	Type      TpConnection `json:"type"`
	Device    string       `json:"device"`
	Database  string       `json:"database"`
	Token     string       `json:"-"`
}

/**
* newSession
* @param username, address string, tp TpConnection, database string
* @return *Session
**/
func newSession(username, device, address string, tp TpConnection, database string) *Session {
	return &Session{
		CreatedAt: time.Now(),
		ID:        reg.ULID(),
		Username:  username,
		Address:   address,
		Status:    Connected,
		Type:      tp,
		Device:    device,
		Database:  database,
	}
}

/**
* Serialize
* @return []byte, error
**/
func (s *Session) Serialize() ([]byte, error) {
	result, err := json.Marshal(s)
	if err != nil {
		return []byte{}, err
	}

	return result, nil
}

/**
* ToJson
* @return et.Json, error
**/
func (s *Session) ToJson() (et.Json, error) {
	definition, err := s.Serialize()
	if err != nil {
		return et.Json{}, err
	}

	result := et.Json{}
	err = json.Unmarshal(definition, &result)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

var sessions *catalog.Model

/**
* initSessions: Initializes the sessions model
* @param db *DB
* @return error
**/
func (s *Node) initSessions() error {
	if sessions != nil {
		return nil
	}

	db, err := s.coreDb()
	if err != nil {
		return err
	}

	sessions, err = db.NewModel("", "sessions", true, 1)
	sessions.DefineAtrib("id", catalog.TpKey, "")
	sessions.DefineAtrib("device", catalog.TpText, "")
	sessions.DefineAtrib("username", catalog.TpText, "")
	sessions.DefineAtrib("address", catalog.TpText, "")
	sessions.DefineAtrib("status", catalog.TpText, "")
	sessions.DefineAtrib("type", catalog.TpText, "")
	sessions.DefinePrimaryKeys("id")
	sessions.DefineIndexes("device", "username", "status")
	if err != nil {
		return err
	}
	if err := sessions.Init(); err != nil {
		return err
	}

	return nil
}

/**
* CreateSession: Creates a new session
* @param device, username string
* @return *Session, error
**/
func (s *Node) CreateSession(device, username string, tpConn TpConnection, database string) (*Session, error) {
	if !utility.ValidStr(username, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}
	if !utility.ValidStr(database, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "database")
	}

	token, err := claim.NewToken(appName, device, username, et.Json{}, 0)
	if err != nil {
		return nil, err
	}

	result := newSession(username, device, s.Address(), tpConn, database)
	result.Token = token
	err = s.initSessions()
	if err != nil {
		return nil, err
	}

	bt, err := result.ToJson()
	if err != nil {
		return nil, err
	}

	err = sessions.PutObject(result.ID, bt)
	if err != nil {
		return nil, err
	}

	s.muSession.Lock()
	s.sessions[result.ID] = result
	s.muSession.Unlock()

	return result, nil
}
