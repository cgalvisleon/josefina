package msg

import "github.com/cgalvisleon/et/envar"

var (
	MSG_MAX_SEGMENT_BYTES      = "maxSegmentBytes must be at least 1 MB"
	MSG_INVALID_ID_LENGTH      = "invalid id length"
	MSG_DATA_TOO_LARGE         = "data too large"
	MSG_FILE_IS_NIL            = "file is nil"
	MSG_CORRUPTED_RECORD       = "corrupted record"
	MSG_INVALID_SNAPSHOT       = "invalid snapshot"
	MSG_SNAPSHOT_CORRUPTED     = "snapshot corrupted"
	MSG_INVALID_SNAPSHOT_MAGIC = "invalid snapshot magic"
	MSG_ID_IS_REQUIRED         = "id is required"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_MAX_SEGMENT_BYTES = "maxSegmentBytes debe ser al menos 1 MB"
		MSG_INVALID_ID_LENGTH = "Longitud de ID inválida"
		MSG_DATA_TOO_LARGE = "Datos demasiado grandes"
		MSG_FILE_IS_NIL = "file es nil"
		MSG_CORRUPTED_RECORD = "registro corrupto"
		MSG_INVALID_SNAPSHOT = "snapshot inválido"
		MSG_SNAPSHOT_CORRUPTED = "snapshot corrupto"
		MSG_INVALID_SNAPSHOT_MAGIC = "magic del snapshot inválido"
		MSG_ID_IS_REQUIRED = "id es requerido"
	}
}
