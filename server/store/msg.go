package store

import "github.com/cgalvisleon/et/envar"

var (
	MSG_MAX_SEGMENT_BYTES = "maxSegmentBytes must be at least 1 MB"
	MSG_INVALID_ID_LENGTH = "invalid id length"
	MSG_DATA_TOO_LARGE    = "data too large"
)

func init() {
	lang := envar.GetStr("LANG", "en")

	if lang == "es" {
		MSG_MAX_SEGMENT_BYTES = "maxSegmentBytes debe ser al menos 1 MB"
		MSG_INVALID_ID_LENGTH = "Longitud de ID inválida"
		MSG_DATA_TOO_LARGE = "Datos demasiado grandes"
	}
}
