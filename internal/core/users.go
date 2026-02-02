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

	db, err := jdb.GetDb(database)
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
		idx := users.GenKey()
		err := users.PutObject(idx, et.Json{
			jdb.ID:     idx,
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
func CreateUser(username, password string) error {
	if !utility.ValidStr(username, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}
	if !utility.ValidStr(password, 3, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "password")
	}

	err := initUsers()
	if err != nil {
		return err
	}

	_, err = users.
		Insert(et.Json{
			jdb.ID:     users.GenKey(),
			"username": username,
			"password": password,
		}).
		Execute(nil)
	return err
}

/**
* DropUser: Drops a user
* @param username string
* @return error
**/
func DropUser(username string) error {
	if !utility.ValidStr(username, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}

	err := initUsers()
	if err != nil {
		return err
	}

	_, err = users.
		Delete().
		Where(jdb.Eq("username", username)).
		Execute(nil)
	return err
}

/**
* GetUser: Gets a user
* @param username, password string
* @return et.Json, error
**/
func GetUser(username, password string) (et.Json, error) {
	if !utility.ValidStr(username, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}
	if !utility.ValidStr(password, 3, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "password")
	}

	err := initUsers()
	if err != nil {
		return nil, err
	}

	item, err := users.
		Selects().
		Where(jdb.Eq("username", username)).
		And(jdb.Eq("password", password)).
		Run(nil)
	if err != nil {
		return nil, err
	}
	if len(item) == 0 {
		return nil, errors.New(msg.MSG_AUTHENTICATION_FAILED)
	}

	return item[0], nil
}

/**
* ChanguePassword: Changues the password of a user
* @param username, password string
* @return error
**/
func ChanguePassword(username, password string) error {
	if !utility.ValidStr(username, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}
	if !utility.ValidStr(password, 6, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "password")
	}

	err := initUsers()
	if err != nil {
		return err
	}

	ok, err := users.IsExisted("username", username)
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
		Where(jdb.Eq("username", username)).
		Execute(nil)
	return err
}
