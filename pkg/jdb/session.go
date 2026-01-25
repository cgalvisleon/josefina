package jdb

import (
	"fmt"
	"time"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

var sessions *Model

/**
* initUsers: Initializes the users model
* @param db *DB
* @return error
**/
func initSessions() error {
	if !node.started {
		return fmt.Errorf(msg.MSG_NODE_NOT_STARTED)
	}

	if sessions != nil {
		return nil
	}

	db, err := getDb(packageName)
	if err != nil {
		return err
	}

	sessions, err = db.newModel("", "sessions", true, 1)
	if err != nil {
		return err
	}
	if err := sessions.init(); err != nil {
		return err
	}

	return nil
}

type Session struct {
	CreatedAt time.Time `json:"created_at"`
	Username  string    `json:"username"`
	Token     string    `json:"token"`
}

/**
* toJson: Converts the session to a json
* @return et.Json
**/
func (s *Session) ToJson() et.Json {
	return et.Json{
		"created_at": s.CreatedAt,
		"username":   s.Username,
		"token":      s.Token,
	}
}

/**
* NewSession: Creates a new session
* @param device, username string
* @return *Session, error
**/
func createSession(device, username string) (*Session, error) {
	if !utility.ValidStr(device, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "device")
	}
	if !utility.ValidStr(username, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}

	err := initSessions()
	if err != nil {
		return nil, err
	}

	token, err := claim.NewToken(packageName, device, username, et.Json{}, 0)
	if err != nil {
		return nil, err
	}

	result := &Session{
		CreatedAt: time.Now(),
		Username:  username,
		Token:     token,
	}

	err = sessions.put(result.Token, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* getSession: Get a session
* @param token string
* @return *Session
**/
func getSession(token string) (*Session, error) {
	if !utility.ValidStr(token, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "token")
	}

	err := initSessions()
	if err != nil {
		return nil, err
	}

	var result Session
	exists, err := sessions.get(token, &result)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf(msg.MSG_SESSION_NOT_FOUND)
	}

	return &result, nil
}

/**
* dropSession: Drops a user
* @param username string
* @return error
**/
func dropSession(token string) error {
	if !utility.ValidStr(token, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "token")
	}

	err := initSessions()
	if err != nil {
		return err
	}

	return sessions.remove(token)
}

/**
* SignIn: Sign in a user
* @param device, username, password string
* @return *Session, error
**/
func SignIn(device, database, username, password string) (*Session, error) {
	if !node.started {
		return nil, fmt.Errorf(msg.MSG_NODE_NOT_STARTED)
	}
	if !utility.ValidStr(username, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_USERNAME_REQUIRED)
	}
	if !utility.ValidStr(password, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_PASSWORD_REQUIRED)
	}

	leader := node.getLeader()
	if leader != node.host {
		result, err := methods.signIn(leader, device, database, username, password)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	err := initUsers()
	if err != nil {
		return nil, err
	}

	item, err := users.
		Selects().
		Where(Eq("username", username)).
		And(Eq("password", password)).
		Rows(nil)
	if err != nil {
		return nil, err
	}
	if len(item) == 0 {
		return nil, fmt.Errorf(msg.MSG_AUTHENTICATION_FAILED)
	}

	return createSession(device, username)
}
