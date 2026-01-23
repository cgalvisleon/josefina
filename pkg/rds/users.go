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
		err := createUser(useranme, password)
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
func (s *Node) createUser(username, password string) error {
	if users == nil {
		return errors.New(msg.MSG_USERS_NOT_FOUND)
	}
	if !utility.ValidStr(username, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}
	if !utility.ValidStr(password, 3, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "password")
	}

	_, err := users.
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
	if users == nil {
		return errors.New(msg.MSG_USERS_NOT_FOUND)
	}
	if !utility.ValidStr(username, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}

	_, err := users.
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
	if users == nil {
		return errors.New(msg.MSG_USERS_NOT_FOUND)
	}
	if !utility.ValidStr(username, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}
	if !utility.ValidStr(password, 6, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "password")
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
