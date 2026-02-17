package jdb

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/tcp"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/msg"
)

/**
* authenticate: Authenticates a user
* @param token string
* @return *claim.Token, error
**/
func (s *Node) authenticate(token string) (*claim.Claim, error) {
	if !utility.ValidStr(token, 0, []string{""}) {
		return nil, errors.New(msg.MSG_CLIENT_NOT_AUTHENTICATION)
	}

	token = utility.PrefixRemove("Bearer", token)
	result, err := claim.ParceToken(token)
	if err != nil {
		return nil, errors.New(msg.MSG_CLIENT_NOT_AUTHENTICATION)
	}

	key := fmt.Sprintf("%s:%s:%s", result.App, result.Device, result.Username)

	var session *Session
	err = s.GetCache(key, &session)
	if err != nil {
		return nil, errors.New(msg.MSG_CLIENT_NOT_AUTHENTICATION)
	}

	if session == nil {
		return nil, errors.New(msg.MSG_CLIENT_NOT_AUTHENTICATION)
	}

	if session.Token != token {
		return nil, errors.New(msg.MSG_CLIENT_NOT_AUTHENTICATION)
	}

	return result, nil
}

/**
* SignIn
* @param device, username, password string
* @return *Session, error
**/
func SignIn(device, username, password string, tpConn TpConnection, database string) (*tcp.Client, error) {
	if node == nil {
		return nil, errors.New(msg.MSG_JOSEFINA_NOT_STARTED)
	}

	user, err := node.GetUser(username, password)
	if err != nil {
		return nil, err
	}

	if !user.Ok {
		return nil, errors.New(msg.MSG_AUTHENTICATION_FAILED)
	}

	session, err := node.CreateSession(device, username, tpConn, database)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%s:%s:%s", appName, device, username)
	err = node.SetCache(key, session, 0)
	if err != nil {
		return nil, err
	}

	client := tcp.NewClient(session.Address)
	return client, nil
}
