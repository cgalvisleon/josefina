package jdb

import (
	"database/sql"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
)

/**
* SqlTx
* @param tx *Tx, sql string, arg ...any
* @return et.Items, error
**/
func (s *DB) SqlTx(tx *Tx, _sql string, arg ...any) (et.Items, error) {
	query := SQLParse(_sql, arg...)

	data := et.Json{
		"db_id":   s.Id,
		"db_name": s.Name,
		"sql":     query,
	}

	var err error
	var rows *sql.Rows
	if tx != nil {
		err = tx.Begin(s.Db)
		if err != nil {
			return et.Items{}, err
		}

		rows, err = tx.Tx.Query(query)
		if err != nil {
			err = fmt.Errorf(`%s: %w`, query, err)
			data["error"] = err.Error()
			event.Publish(EVENT_SQL_ERROR, data)
			errRollback := tx.Rollback()
			if errRollback != nil {
				err = fmt.Errorf("error on rollback: %w: %s", errRollback, err)
			}

			return et.Items{}, err
		}
	} else {
		rows, err = s.Db.Query(query)
		if err != nil {
			err = fmt.Errorf(`%s: %w`, query, err)
			data["error"] = err.Error()
			event.Publish(EVENT_SQL_ERROR, data)
			return et.Items{}, err
		}
	}

	tp := TipoSQL(query)
	channel := fmt.Sprintf("sql:%s", tp)
	event.Publish(channel, data)
	defer rows.Close()
	result := RowsToItems(rows)
	return result, nil
}

/**
* Sql
* @param sql string, arg ...any
* @return et.Items, error
**/
func (s *DB) Sql(_sql string, arg ...any) (et.Items, error) {
	return s.SqlTx(nil, _sql, arg...)
}
