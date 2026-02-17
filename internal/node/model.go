package node

import (
	"errors"

	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/msg"
)

var models *catalog.Model

/**
* initModels: Initializes the models model
* @return error
**/
func (s *Node) initModels() error {
	if models != nil {
		return nil
	}

	db, err := s.coreDb()
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
	leader, imLeader := s.GetLeader()
	if imLeader {
		return s.lead.GetModel(from)
	}

	if leader != nil {
		res := s.Request(leader, "Leader.GetModel", from)
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

	return nil, false
}

/**
* DropModel: Removes a model
* @param from *catalog.From
* @return error
**/
func (s *Node) DropModel(from *catalog.From) error {
	leader, imLeader := s.GetLeader()
	if imLeader {
		return s.lead.DropModel(from)
	}

	if leader != nil {
		res := s.Request(leader, "Leader.DropModel", from)
		if res.Error != nil {
			return res.Error
		}

		return nil
	}

	return errors.New(msg.MSG_LEADER_NOT_FOUND)
}

/**
* SaveModel
* @param model *catalog.Model
* @return error
**/
func (s *Node) SaveModel(model *catalog.Model) error {
	leader, imLeader := s.GetLeader()
	if imLeader {
		return s.lead.SaveModel(model)
	}

	if leader != nil {
		res := s.Request(leader, "Leader.SaveModel", model)
		if res.Error != nil {
			return res.Error
		}

		return nil
	}

	return errors.New(msg.MSG_LEADER_NOT_FOUND)
}

/**
* CreateModel: Creates a model
* @param from *catalog.From
* @return *catalog.Model, error
**/
func (s *Node) CreateModel(database, schema, name string, version int) (*catalog.Model, error) {
	result, exists := s.GetModel(&catalog.From{
		Database: database,
		Schema:   schema,
		Name:     name,
	})

	if exists {
		return nil, errors.New(msg.MSG_MODEL_EXISTS)
	}

	db, exists := s.GetDb(database)
	if !exists {
		return nil, errors.New(msg.MSG_DB_NOT_FOUND)
	}

	result, err := db.NewModel(schema, name, false, version)
	if err != nil {
		return nil, err
	}

	err = s.SaveModel(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
