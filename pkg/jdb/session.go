package jdb

import (
	"errors"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

type Session struct {
	ID        string    `json:"id"`
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

	token, err := claim.NewToken(node.PackageName, device, username, et.Json{}, 0)
	if err != nil {
		return nil, err
	}

	result := &Session{
		CreatedAt: time.Now(),
		Username:  username,
		Token:     token,
	}

	return result, nil
}

/**
* dropSession: Drops a user
* @param username string
* @return error
**/
func DropSession(token string) error {
	if !utility.ValidStr(token, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "token")
	}

	result, err := claim.ParceToken(token)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%s:%s", result.App, result.Device, result.Username)
	_, err = DeleteCache(key)
	if err != nil {
		return err
	}

	return nil
}

/**
* SignIn: Sign in a user
* @param device, username, password string
* @return *Session, error
**/
func Auth(device, database, username, password string) (*Session, error) {
	if !node.started {
		return nil, errors.New(msg.MSG_NODE_NOT_STARTED)
	}
	if !utility.ValidStr(username, 0, []string{""}) {
		return nil, errors.New(msg.MSG_USERNAME_REQUIRED)
	}
	if !utility.ValidStr(password, 0, []string{""}) {
		return nil, errors.New(msg.MSG_PASSWORD_REQUIRED)
	}

	if database == "" {
		database = node.PackageName
	}
	leader, ok := node.getLeader()
	if ok {
		result, err := methods.auth(leader, device, database, username, password)
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
		Run(nil)
	if err != nil {
		return nil, err
	}
	if len(item) == 0 {
		return nil, errors.New(msg.MSG_AUTHENTICATION_FAILED)
	}

	result, err := createSession(device, username)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%s:%s:%s", node.PackageName, device, username)
	_, err = SetCache(key, result.Token, 0)
	if err != nil {
		return nil, err
	}

	return result, nil
}
