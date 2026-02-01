package core

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/jdb"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

var users *jdb.Model

/**
* initUsers: Initializes the users model
* @param db *DB
* @return error
**/
func initUsers() error {
	if users != nil {
		return nil
	}

	db, err := node.GetDb(database)
	if err != nil {
		return err
	}

	users, err = db.NewModel("", "users", true, 1)
	if err != nil {
		return err
	}
	users.DefineAtrib(jdb.ID, jdb.TpKey, "")
	users.DefineAtrib("username", jdb.TpText, "")
	users.DefineAtrib("password", jdb.TpText, "")
	users.DefineHidden("password")
	users.DefinePrimaryKeys("username")
	users.DefineUnique(jdb.ID)
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
		idx := users.genKey()
		err := users.putObject(idx, et.Json{
			ID:         idx,
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
* createUser: Creates a new user
* @param username, password string
* @return error
**/
func createUser(username, password string) error {
	if !node.started {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}
	if !utility.ValidStr(username, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}
	if !utility.ValidStr(password, 3, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "password")
	}

	leader, ok := node.getLeader()
	if ok {
		err := syn.createUser(leader, username, password)
		if err != nil {
			return err
		}

		return nil
	}

	err := initUsers()
	if err != nil {
		return err
	}

	_, err = users.
		Insert(et.Json{
			ID:         users.genKey(),
			"username": username,
			"password": password,
		}).
		Execute(nil)
	return err
}

/**
* dropUser: Drops a user
* @param username string
* @return error
**/
func dropUser(username string) error {
	if !node.started {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}
	if !utility.ValidStr(username, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}

	leader, ok := node.getLeader()
	if ok {
		err := syn.dropUser(leader, username)
		if err != nil {
			return err
		}

		return nil
	}

	err := initUsers()
	if err != nil {
		return err
	}

	_, err = users.
		Delete().
		Where(Eq("username", username)).
		Execute(nil)
	return err
}

/**
* changuePassword: Changues the password of a user
* @param username, password string
* @return error
**/
func changuePassword(username, password string) error {
	if !node.started {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}
	if !utility.ValidStr(username, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}
	if !utility.ValidStr(password, 6, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "password")
	}

	leader, ok := node.getLeader()
	if ok {
		err := syn.changuePassword(leader, username, password)
		if err != nil {
			return err
		}

		return nil
	}

	err := initUsers()
	if err != nil {
		return err
	}

	ok, err = users.isExisted("username", username)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New(msg.MSG_USER_NOT_FOUND)
	}

	_, err = users.
		Update(et.Json{
			"password": password,
		}).
		Where(Eq("username", username)).
		Execute(nil)
	return err
}
