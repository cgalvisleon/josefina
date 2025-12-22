package postgres

import (
	"database/sql"

	"github.com/cgalvisleon/josefina/jdb"
)

/**
* ExistTable
* @param db *sql.DB, schema, name string
* @return bool, error
**/
func ExistTable(db *sql.DB, schema, name string) (bool, error) {
	rows, err := db.Query(`
	SELECT EXISTS(
		SELECT 1
		FROM information_schema.tables
		WHERE UPPER(table_schema) = UPPER($1)
		AND UPPER(table_name) = UPPER($2));`, schema, name)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	items := jdb.RowsToItems(rows)

	if items.Count == 0 {
		return false, nil
	}

	return items.Bool(0, "exists"), nil
}
