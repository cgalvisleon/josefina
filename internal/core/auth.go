package core

import (
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
		return nil, msg.ERROR_CLIENT_NOT_AUTHENTICATION.Error()
	}

	token = utility.PrefixRemove("Bearer", token)
	result, err := claim.ParceToken(token)
	if err != nil {
		return nil, msg.ERROR_CLIENT_NOT_AUTHENTICATION.Error()
	}

	key := fmt.Sprintf("%s:%s:%s", result.App, result.Device, result.Username)
	session, exists := cache.GetStr(key)
	if !exists {
		return nil, msg.ERROR_CLIENT_NOT_AUTHENTICATION.Error()
	}

	if session != token {
		return nil, msg.ERROR_CLIENT_NOT_AUTHENTICATION.Error()
	}

	return result, nil
}
