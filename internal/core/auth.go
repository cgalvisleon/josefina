package core

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/cache"
	"github.com/cgalvisleon/josefina/pkg/msg"
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
	session, exists := cache.GetStr(key)
	if !exists {
		return nil, errors.New(msg.MSG_CLIENT_NOT_AUTHENTICATION)
	}

	if session != token {
		return nil, errors.New(msg.MSG_CLIENT_NOT_AUTHENTICATION)
	}

	return result, nil
}
