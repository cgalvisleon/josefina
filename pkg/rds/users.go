package rds

import (
	"fmt"

	"github.com/cgalvisleon/et/envar"
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
		err := node.createUser(useranme, password)
		if err != nil {
			return err
		}
	}

	return nil
}
