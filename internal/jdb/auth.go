package jdb

import (
	"errors"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/josefina/internal/msg"
)

/**
* Authenticate: Authenticates a user
* @param token string
* @return *claim.Token, error
**/
func (s *Node) Authenticate(token string) (*claim.Claim, error) {
	leader, imLeader := s.GetLeader()
	if imLeader {
		return s.lead.Authenticate(token)
	}

	if leader != nil {
		res := s.Request(leader, "Leader.Authenticate", token)
		if res.Error != nil {
			return nil, res.Error
		}

		var result *claim.Claim
		err := res.Get(&result)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	return nil, errors.New(msg.MSG_LEADER_NOT_FOUND)
}
