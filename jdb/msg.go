package jdb

import "errors"

var (
	ErrModelNotFound         error  = errors.New("model not found")
	MSG_DRIVER_NOT_FOUND     string = "driver not found"
	MSG_NAME_REQUIRED        string = "name required"
	MSG_COLUMN_EXISTS        string = "column %s already exists"
	MSG_TYPE_COLUMN_REQUIRED string = "type column required"
	MSG_TYPE_DATA_REQUIRED   string = "type data required"
	MSG_MODEL_NOT_FOUND      string = "model %s not found"
	MSG_ATTRIBUTE_REQUIRED   string = "attribute %s required"
	MSG_DATABASE_NOT_FOUND   string = "database %s not found"
	MSG_DATABASE_REQUIRED    string = "database required"
	MSG_COMMAND_INVALID      string = "invalid command: %s"
	MSG_FROM_REQUIRED        string = "from required"
	MSG_DATA_REQUIRED        string = "data required"
)

func loadMsg(language string) {
	switch language {
	case "es":
		ErrModelNotFound = errors.New("modelo no encontrado")
		MSG_DRIVER_NOT_FOUND = "driver no encontrado"
		MSG_NAME_REQUIRED = "nombre requerido"
		MSG_COLUMN_EXISTS = "columna %s ya existe"
		MSG_TYPE_COLUMN_REQUIRED = "tipo columna requerido"
		MSG_TYPE_DATA_REQUIRED = "tipo datos requerido"
		MSG_MODEL_NOT_FOUND = "modelo %s no encontrado"
		MSG_ATTRIBUTE_REQUIRED = "atributo %s requerido"
		MSG_DATABASE_NOT_FOUND = "base de datos %s no encontrada"
		MSG_DATABASE_REQUIRED = "base de datos requerida"
		MSG_COMMAND_INVALID = "comando invalido: %s"
		MSG_FROM_REQUIRED = "from requerido"
		MSG_DATA_REQUIRED = "data requerida"
	}
}
