package jql

import (
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/msg"
)

var (
	commands map[string]Command = map[string]Command{
		"SELECT": SELECT,
		"INSERT": INSERT,
		"UPDATE": UPDATE,
		"DELETE": DELETE,
		"UPSERT": UPSERT,
		"CREATE": CREATE,
		"DROP":   DROP,
		"CHANGE": CHANGE,
		"select": SELECT,
		"insert": INSERT,
		"update": UPDATE,
		"delete": DELETE,
		"upsert": UPSERT,
		"create": CREATE,
		"drop":   DROP,
		"change": CHANGE,
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
