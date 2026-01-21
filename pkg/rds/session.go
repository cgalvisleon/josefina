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

type Sessions struct {
	sessions []*Session `json:"-"`
}

/**
* add: Add a session
* @param session *Session
* @return void
**/
func (s *Sessions) add(session *Session) {
	s.sessions = append(s.sessions, session)
}

/**
* remove: Remove a session
* @param token string
* @return void
**/
func (s *Sessions) remove(token string) {
	for i, session := range s.sessions {
		if session.Token == token {
			s.sessions = append(s.sessions[:i], s.sessions[i+1:]...)
			break
		}
	}
}

/**
* get: Get a session
* @param token string
* @return *Session
**/
func (s *Sessions) get(token string) *Session {
	for _, session := range s.sessions {
		if session.Token == token {
			return session
		}
	}
	return nil
}

var sessions *Sessions

func init() {
	sessions = &Sessions{
		sessions: make([]*Session, 0),
	}
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

	result := &Session{
		CreatedAt: time.Now(),
		Username:  username,
		Token:     token,
	}

	sessions.add(result)
	return result, nil
}

/**
* signIn: Sign in a user
* @param device, username, password string
* @return *Session, error
**/
func signIn(device, username, password string) (*Session, error) {
	if !utility.ValidStr(username, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_USERNAME_REQUIRED)
	}
	if !utility.ValidStr(password, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_PASSWORD_REQUIRED)
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

	return newSession(device, username)
}
