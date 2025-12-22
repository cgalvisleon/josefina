package postgres

var (
	MSG_ATRIB_REQUIRED = "Atrib required (%s)"
	MSG_CREATE_MODEL   = "Create model:%s v:%d"
	MSG_MUTATE_MODEL   = "Mutate model:%s v:%d"
	MSG_LOAD_MODEL     = "Load model:%s v:%d"
)

func loadMsg(language string) {
	switch language {
	case "es":
		MSG_ATRIB_REQUIRED = "Atributo requerido (%s)"
		MSG_CREATE_MODEL = "Crear modelo:%s v:%d"
		MSG_MUTATE_MODEL = "Mutar modelo:%s v:%d"
		MSG_LOAD_MODEL = "Cargar modelo:%s v:%d"
	}
}
