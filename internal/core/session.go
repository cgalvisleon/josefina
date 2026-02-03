package core

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/mod"
	"github.com/cgalvisleon/josefina/internal/msg"
)

var sessions *mod.Model

/**
* initSessions: Initializes the sessions model
* @param db *DB
* @return error
**/
func initSessions() error {
	if sessions != nil {
		return nil
	}

	db, err := mod.CoreDb()
	if err != nil {
		return err
	}

	sessions, err = db.NewModel("", "sessions", true, 1)
	if err != nil {
		return err
	}
	if err := series.Init(); err != nil {
		return err
	}

	return nil
}

type Session struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Username  string    `json:"username"`
	Token     string    `json:"token"`
}

/**
* serialize
* @return []byte, error
**/
func (s *Session) serialize() ([]byte, error) {
	result, err := json.Marshal(s)
	if err != nil {
		return []byte{}, err
	}

	return result, nil
}

/**
* toJson: Converts the session to a json
* @return et.Json, error
**/
func (s *Session) ToJson() (et.Json, error) {
	definition, err := s.serialize()
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

/**
* CreateSession: Creates a new session
* @param device, username string
* @return *Session, error
**/
func CreateSession(device, username string) (*Session, error) {
	leader, ok := syn.getLeader()
	if ok {
		return syn.createSession(leader, device, username)
	}

	if !utility.ValidStr(device, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "device")
	}
	if !utility.ValidStr(username, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}

	token, err := claim.NewToken(appName, device, username, et.Json{}, 0)
	if err != nil {
		return nil, err
	}

	result := &Session{
		CreatedAt: time.Now(),
		Username:  username,
		Token:     token,
	}

	err = initSessions()
	if err != nil {
		return nil, err
	}

	bt, err := result.serialize()
	if err != nil {
		return nil, err
	}

	key := result.Token
	err = sessions.Put(key, bt)
	if err != nil {
		return nil, err
	}

	return result, nil
}
