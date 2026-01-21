package rds

import (
	"fmt"
	"time"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

type Session struct {
	CreatedAt time.Time `json:"created_at"`
	Username  string    `json:"username"`
	Token     string    `json:"token"`
}

/**
* NewSession: Creates a new session
* @param device, username string
* @return *Session, error
**/
func newSession(device, username string) (*Session, error) {
	token, err := claim.NewToken(packageName, device, username, et.Json{}, 0)
	if err != nil {
		return nil, err
	}

	return &Session{
		CreatedAt: time.Now(),
		Username:  username,
		Token:     token,
	}, nil
}

var sessions []*Session

func init() {
	sessions = make([]*Session, 0)
}

func signIn(device, database, username, password string) (*Session, error) {
	if !utility.ValidStr(username, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_USERNAME_REQUIRED)
	}
	if !utility.ValidStr(password, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_PASSWORD_REQUIRED)
	}

	users.
		Selects("password").
		Where(Eq("username", username)).
		And(Eq("password", password)).
		Rows(nil)		
	return nil, nil
}
