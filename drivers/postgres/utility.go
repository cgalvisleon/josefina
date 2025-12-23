package postgres

import (
	"fmt"

	"github.com/cgalvisleon/josefina/jdb"
)

func FieldAs(field *jdb.Field) string {
	if field.From == nil {
		return field.Name
	}

	return fmt.Sprintf(`%s.%s`, field.From.As, field.Name)
}
