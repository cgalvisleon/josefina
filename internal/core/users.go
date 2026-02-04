package core

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/msg"
)

var users *catalog.Model

/**
* initUsers: Initializes the users model
* @param db *DB
* @return error
**/
func initUsers() error {
	if users != nil {
		return nil
	}

	db, err := catalog.CoreDb()
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
func CreateUser(username, password string) error {
	leader, ok := syn.getLeader()
	if ok {
		return syn.createUser(leader, username, password)
	}

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
			catalog.ID: users.GenKey(),
			"username": username,
			"password": password,
		}).
		Execute(nil)
	return err
}

/**
* DropUser: Drops a user
* @param username, password string
* @return error
**/
func DropUser(username, password string) error {
	leader, ok := syn.getLeader()
	if ok {
		return syn.dropUser(leader, username, password)
	}

	if !utility.ValidStr(username, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}

	err := initUsers()
	if err != nil {
		return err
	}

	_, err = users.
		Delete().
		Where(catalog.Eq("username", username)).
		Execute(nil)
	return err
}

/**
* GetUser: Gets a user
* @param username, password string
* @return et.Json, error
**/
func GetUser(username, password string) (et.Json, error) {
	leader, ok := syn.getLeader()
	if ok {
		return syn.getUser(leader, username, password)
	}

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
		Where(catalog.Eq("username", username)).
		And(catalog.Eq("password", password)).
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
* @param username, oldPassword, newPassword string
* @return error
**/
func ChanguePassword(username, oldPassword, newPassword string) error {
	leader, ok := syn.getLeader()
	if ok {
		return syn.changuePassword(leader, username, oldPassword, newPassword)
	}

	if !utility.ValidStr(username, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}
	if !utility.ValidStr(oldPassword, 6, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "oldPassword")
	}
	if !utility.ValidStr(newPassword, 6, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "newPassword")
	}

	err := initUsers()
	if err != nil {
		return err
	}

	ok, err = users.IsExisted("username", username)
	if err != nil {
		return err
	}

	if !ok {
		return errors.New(msg.MSG_USER_NOT_FOUND)
	}

	_, err = users.
		Update(et.Json{
			"password": newPassword,
		}).
		Where(catalog.Eq("username", username)).
		And(catalog.Eq("password", oldPassword)).
		Execute(nil)
	return err
}
