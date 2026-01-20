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
func initUsers(db *DB) error {
	var err error
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

	if users.count() == 0 {
		useranme := envar.GetStr("USERNAME", "admin")
		password := envar.GetStr("PASSWORD", "admin")
		err := CreateUser(useranme, password)
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
func CreateUser(username, password string) error {
	_, err := users.insert(nil, et.Json{
		"username": username,
		"password": password,
	})
	return err
}

/**
* DropUser: Drops a user
* @param username string
* @return error
**/
func DropUser(username string) error {
	if users == nil {
		return errors.New(msg.MSG_USERS_NOT_FOUND)
	}

	if !utility.ValidStr(username, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}
	_, err := users.delete(nil, Where(Eq("username", username)))
	return err
}

/**
* ChanguePassword: Changues the password of a user
* @param username, newpassword string
* @return error
**/
func ChanguePassword(username, newpassword string) error {
	ok, err := users.isExisted("username", username)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New(msg.MSG_USER_NOT_FOUND)
	}

	_, err = users.update(nil, et.Json{
		"password": newpassword,
	}, users.where(Eq("username", username)))
	return err
}
