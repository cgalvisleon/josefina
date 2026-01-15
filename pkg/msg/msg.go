package msg

import "github.com/cgalvisleon/et/envar"

var (
	MSG_NOT_FOUND              = "not found"
	MSG_MAX_SEGMENT_BYTES      = "maxSegmentBytes must be at least 1 MB"
	MSG_INVALID_ID_LENGTH      = "invalid id length"
	MSG_DATA_TOO_LARGE         = "data too large"
	MSG_FILE_IS_NIL            = "file is nil"
	MSG_CORRUPTED_RECORD       = "corrupted record"
	MSG_INVALID_SNAPSHOT       = "invalid snapshot"
	MSG_SNAPSHOT_CORRUPTED     = "snapshot corrupted"
	MSG_INVALID_SNAPSHOT_MAGIC = "invalid snapshot magic"
	MSG_ID_IS_REQUIRED         = "id is required"
	MSG_SCHEMA_NOT_FOUND       = "schema not found"
	MSG_MODEL_NOT_FOUND        = "model not found"
	MSG_DB_NOT_FOUND           = "database not found"
	MSG_ARG_REQUIRED           = "argument required (%s)"
	MSG_FIELD_REQUIRED         = "field required (%s)"
	MSG_TENNANT_NOT_FOUND      = "tennant not found"
	MSG_INDEX_NOT_FOUND        = "index not found"
	MSG_RECORD_EXISTS          = "record exists"
	MSG_RECORD_NOT_FOUND       = "record not found"
	MSG_PRIMARY_KEYS_NOT_FOUND = "primary key not found"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_NOT_FOUND = "no encontrado"
		MSG_MAX_SEGMENT_BYTES = "maxSegmentBytes debe ser al menos 1 MB"
		MSG_INVALID_ID_LENGTH = "Longitud de ID inválida"
		MSG_DATA_TOO_LARGE = "Datos demasiado grandes"
		MSG_FILE_IS_NIL = "file es nil"
		MSG_CORRUPTED_RECORD = "registro corrupto"
		MSG_INVALID_SNAPSHOT = "snapshot inválido"
		MSG_SNAPSHOT_CORRUPTED = "snapshot corrupto"
		MSG_INVALID_SNAPSHOT_MAGIC = "magic del snapshot inválido"
		MSG_ID_IS_REQUIRED = "id es requerido"
		MSG_SCHEMA_NOT_FOUND = "schema no encontrado"
		MSG_MODEL_NOT_FOUND = "model no encontrado"
		MSG_DB_NOT_FOUND = "database no encontrado"
		MSG_ARG_REQUIRED = "argumento requerido (%s)"
		MSG_FIELD_REQUIRED = "field requerido (%s)"
		MSG_TENNANT_NOT_FOUND = "tennant no encontrado"
		MSG_INDEX_NOT_FOUND = "index no encontrado"
		MSG_RECORD_EXISTS = "record exists"
		MSG_RECORD_NOT_FOUND = "record no encontrado"
		MSG_PRIMARY_KEYS_NOT_FOUND = "primary key no encontrado"
	}
}
