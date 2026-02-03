package jql

import (
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type Command string

const (
	SELECT       Command = "SELECT"
	INSERT       Command = "INSERT"
	UPDATE       Command = "UPDATE"
	DELETE       Command = "DELETE"
	UPSERT       Command = "UPSERT"
	CREATE_DB    Command = "CREATE_DB"
	GET_DB       Command = "GET_DB"
	DROP_DB      Command = "DROP_DB"
	CREATE_MODEL Command = "CREATE_MODEL"
	GET_MODEL    Command = "GET_MODEL"
	DROP_MODEL   Command = "DROP_MODEL"
	CREATE_SERIE Command = "CREATE_SERIE"
	SET_SERIE    Command = "SET_SERIE"
	GET_SERIE    Command = "GET_SERIE"
	DROP_SERIE   Command = "DROP_SERIE"
	CREATE_USER  Command = "CREATE_USER"
	GET_USER     Command = "GET_USER"
	DROP_USER    Command = "DROP_USER"
)

var (
	commands map[string]Command = map[string]Command{
		"SELECT":    SELECT,
		"INSERT":    INSERT,
		"UPDATE":    UPDATE,
		"DELETE":    DELETE,
		"UPSERT":    UPSERT,
		"CREATE DB": CREATE_DB,
	}
)

/**
* getCommand: Gets the command from the json
* @param cmd et.Json
* @return Command, error
**/
func getCommand(cmd et.Json) (Command, error) {
	for k, v := range commands {
		_, ok := cmd[k]
		if ok {
			return v, nil
		}
	}

	return "", errors.New(msg.MSG_COMMAND_NOT_FOUND)
}
