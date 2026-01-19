package rds

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
)

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
	users.defineAtrib("username", TpText, "")
	users.defineAtrib("password", TpText, "")
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
