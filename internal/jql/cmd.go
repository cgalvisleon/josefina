package jql

type Command string

const (
	SELECT       Command = "SELECT"
	INSERT       Command = "INSERT"
	UPDATE       Command = "UPDATE"
	DELETE       Command = "DELETE"
	UPSERT       Command = "UPSERT"
	CREATE_DB    Command = "CREATE_DB"
	GET_DB       Command = "GET_DB"
	USE_DB       Command = "USE_DB"
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

type Cmd struct{}
