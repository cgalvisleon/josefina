package postgres

import (
	"database/sql"
	"fmt"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/josefina/jdb"
	_ "github.com/lib/pq"
)

/**
* ExistDatabase
* @param db *DB, name string
* @return bool, error
**/
func ExistDatabase(db *sql.DB, name string) (bool, error) {
	sql := `
	SELECT EXISTS(
	SELECT 1
	FROM pg_database
	WHERE UPPER(datname) = UPPER($1));`
	rows, err := db.Query(sql, name)
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

/**
* CreateDatabase
* @param db *sql.DB, name string
* @return error
**/
func CreateDatabase(db *sql.DB, name string) error {
	exist, err := ExistDatabase(db, name)
	if err != nil {
		return err
	}

	if exist {
		return nil
	}

	sql := fmt.Sprintf(`CREATE DATABASE %s;`, name)
	_, err = db.Exec(sql)
	if err != nil {
		return err
	}

	logs.Logf(driver, `Database %s created`, name)

	return nil
}

/**
* DropDatabase
* @param db *sql.DB, name string
* @return error
**/
func DropDatabase(db *sql.DB, name string) error {
	sql := fmt.Sprintf(`DROP DATABASE %s;`, name)
	_, err := db.Exec(sql)
	if err != nil {
		return err
	}

	logs.Logf(driver, `Database %s droped`, name)

	return nil
}
