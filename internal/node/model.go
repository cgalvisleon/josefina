package node

import (
	"errors"

	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/msg"
)

var (
	models *catalog.Model
)

/**
* initModels: Initializes the models model
* @return error
**/
func initModels() error {
	if models != nil {
		return nil
	}

	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	db, err := node.coreDb()
	if err != nil {
		return err
	}

	models, err = db.NewModel("", "models", true, 1)
	if err != nil {
		return err
	}
	if err := models.Init(); err != nil {
		return err
	}

	return nil
}

/**
* getModel: Gets a model
* @param from *catalog.From
* @return *catalog.Model, bool
**/
func (s *Node) GetModel(from *catalog.From) (*catalog.Model, bool) {
	leader, imLeader := node.GetLeader()
	if !imLeader && leader != nil {
		res := node.Request(leader, "Leader.GetModel", from)
		if res.Error != nil {
			return nil, false
		}

		var result *catalog.Model
		err := res.Get(&result)
		if err != nil {
			return nil, false
		}

		return result, true
	}

	return s.lead.GetModel(from)
}
