package jdb

import (
	"errors"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/msg"
)

var users *catalog.Model

/**
* initUsers: Initializes the users model
* @param db *DB
* @return error
**/
func (s *Node) initUsers() error {
	if users != nil {
		return nil
	}

	db, err := s.coreDb()
	if err != nil {
		return err
	}

	users, err = db.NewModel("", "users", true, 1)
	if err != nil {
		return err
	}
	users.DefineAtrib(catalog.ID, catalog.TpKey, "")
	users.DefineAtrib("username", catalog.TpText, "")
	users.DefineAtrib("password", catalog.TpText, "")
	users.DefineHidden("password")
	users.DefinePrimaryKeys("username")
	users.DefineUnique(catalog.ID)
	if err := users.Init(); err != nil {
		return err
	}

	count, err := users.Count()
	if err != nil {
		return err
	}

	if count == 0 {
		useranme := envar.GetStr("USERNAME", "admin")
		password := envar.GetStr("PASSWORD", "admin")
		idx := users.GenKey()
		err := users.PutObject(idx, et.Json{
			catalog.ID: idx,
			"username": useranme,
			"password": password,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* CreateUser: Creates a new user
* @param username, password string
* @return error
**/
func (s *Node) CreateUser(username, password string) error {
	leader, imLeader := s.GetLeader()
	if imLeader {
		return s.lead.CreateUser(username, password)
	}

	if leader != nil {
		res := s.Request(leader, "Leader.CreateUser", username, password)
		if res.Error != nil {
			return res.Error
		}
		return nil
	}

	return errors.New(msg.MSG_LEADER_NOT_FOUND)
}

/**
* DropUser: Drops a user
* @param username, password string
* @return error
**/
func (s *Node) DropUser(username, password string) error {
	leader, imLeader := s.GetLeader()
	if imLeader {
		return s.lead.DropUser(username, password)
	}

	if leader != nil {
		res := s.Request(leader, "Leader.DropUser", username, password)
		if res.Error != nil {
			return res.Error
		}
		return nil
	}

	return errors.New(msg.MSG_LEADER_NOT_FOUND)
}

/**
* GetUser: Gets a user
* @param username, password string
* @return et.Item, error
**/
func (s *Node) GetUser(username, password string) (et.Item, error) {
	leader, imLeader := s.GetLeader()
	if imLeader {
		return s.lead.GetUser(username, password)
	}

	if leader != nil {
		res := s.Request(leader, "Leader.GetUser", username, password)
		if res.Error != nil {
			return et.Item{}, res.Error
		}

		var result et.Item
		err := res.Get(&result)
		if err != nil {
			return et.Item{}, err
		}
		return result, nil
	}

	return et.Item{}, errors.New(msg.MSG_LEADER_NOT_FOUND)
}

/**
* ChanguePassword: Changues the password of a user
* @param username, oldPassword, newPassword string
* @return error
**/
func (s *Node) ChanguePassword(username, oldPassword, newPassword string) error {
	leader, imLeader := s.GetLeader()
	if imLeader {
		return s.lead.ChanguePassword(username, oldPassword, newPassword)
	}

	if leader != nil {
		res := s.Request(leader, "Leader.ChanguePassword", username, oldPassword, newPassword)
		if res.Error != nil {
			return res.Error
		}

		return nil
	}

	return errors.New(msg.MSG_LEADER_NOT_FOUND)
}
