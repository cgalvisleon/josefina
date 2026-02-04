package jdb

import (
	"errors"

	"github.com/cgalvisleon/josefina/internal/core"
	"github.com/cgalvisleon/josefina/internal/mod"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type Leader struct{}

var leader *Leader

func init() {
	leader = &Leader{}
}

/**
* GetModel: Gets a model
* @param require *mod.From, response *mod.Model
* @return error
**/
func (s *Leader) GetModel(require *mod.From, response *mod.Model) error {
	exists, err := core.GetModel(require, response)
	if err != nil {
		return err
	}

	if !exists {
		return errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	if !response.IsInit {
		host := node.nextHost()
		response, err = mod.LoadModel(host, response)
		if err != nil {
			return err
		}

		mod.
	}

	return nil
}
