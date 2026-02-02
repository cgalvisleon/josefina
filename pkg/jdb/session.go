package jdb

import (
	"fmt"
	"time"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/cache"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

var (
	appName = "josefina"
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
* CreateSession: Creates a new session
* @param device, username string
* @return *Session, error
**/
func CreateSession(device, username string) (*Session, error) {
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
	_, err = cache.Delete(key)
	if err != nil {
		return err
	}

	return nil
}
