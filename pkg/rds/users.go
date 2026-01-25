package rds

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

var users *Model

/**
* initUsers: Initializes the users model
* @param db *DB
* @return error
**/
func initUsers() error {
	if !node.started {
		return fmt.Errorf(msg.MSG_NODE_NOT_STARTED)
	}

	if users != nil {
		return nil
	}

	db, err := getDb(packageName)
	if err != nil {
		return err
	}

	users, err = db.newModel("", "users", true, 1)
	if err != nil {
		return err
	}
	users.DefineAtrib("username", TpText, "")
	users.DefineAtrib("password", TpText, "")
	users.defineHidden("password")
	users.definePrimaryKey("username")
	if err := users.init(); err != nil {
		return err
	}

	count, err := users.count()
	if err != nil {
		return err
	}

	if count == 0 {
		useranme := envar.GetStr("USERNAME", "admin")
		password := envar.GetStr("PASSWORD", "admin")
		idx := users.genKey()
		err := users.putObject(idx, et.Json{
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
		return fmt.Errorf(msg.MSG_NODE_NOT_STARTED)
	}
	if !utility.ValidStr(username, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}
	if !utility.ValidStr(password, 3, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "password")
	}

	leader := node.getLeader()
	if leader != node.host {
		err := methods.createUser(leader, username, password)
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
		return fmt.Errorf(msg.MSG_NODE_NOT_STARTED)
	}
	if !utility.ValidStr(username, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}

	leader := node.getLeader()
	if leader != node.host {
		err := methods.dropUser(leader, username)
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
		return fmt.Errorf(msg.MSG_NODE_NOT_STARTED)
	}
	if !utility.ValidStr(username, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}
	if !utility.ValidStr(password, 6, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "password")
	}

	leader := node.getLeader()
	if leader != node.host {
		err := methods.changuePassword(leader, username, password)
		if err != nil {
			return err
		}

		return nil
	}

	err := initUsers()
	if err != nil {
		return err
	}

	ok, err := users.isExisted("username", username)
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
