package node

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/cache"
	"github.com/cgalvisleon/josefina/internal/msg"
)

/**
* Authenticate: Authenticates a user
* @param token string
* @return *claim.Token, error
**/
func Authenticate(token string) (*claim.Claim, error) {
	if !utility.ValidStr(token, 0, []string{""}) {
		return nil, errors.New(msg.MSG_CLIENT_NOT_AUTHENTICATION)
	}

	token = utility.PrefixRemove("Bearer", token)
	result, err := claim.ParceToken(token)
	if err != nil {
		return nil, errors.New(msg.MSG_CLIENT_NOT_AUTHENTICATION)
	}

	key := fmt.Sprintf("%s:%s:%s", result.App, result.Device, result.Username)
	session, exists, err := cache.GetStr(key)
	if err != nil {
		return nil, errors.New(msg.MSG_CLIENT_NOT_AUTHENTICATION)
	}
	if !exists {
		return nil, errors.New(msg.MSG_CLIENT_NOT_AUTHENTICATION)
	}

	if session != token {
		return nil, errors.New(msg.MSG_CLIENT_NOT_AUTHENTICATION)
	}

	return result, nil
}

/**
* SignIn
* @param device, username, password string
* @return *Session, error
**/
func (n *Node) SignIn(device, username, password string) (*Session, error) {
	item, err := GetUser(username, password)
	if err != nil {
		return nil, err
	}
	if len(item) == 0 {
		return nil, errors.New(msg.MSG_AUTHENTICATION_FAILED)
	}

	result, err := CreateSession(device, username)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%s:%s:%s", appName, device, username)
	_, err = cache.Set(key, result.Token, 0)
	if err != nil {
		return nil, err
	}

	return result, nil
}
