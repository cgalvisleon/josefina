package jdb

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/josefina/internal/msg"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/microsoft/go-mssqldb"
	goora "github.com/sijms/go-ora/v2"
	_ "modernc.org/sqlite"
)

const (
	DataSourcePostgres  = "postgres"
	DataSourceMysql     = "mysql"
	DataSourceSqlite    = "sqlite"
	DataSourceOracle    = "oracle"
	DataSourceSqlserver = "sqlserver"
)

/**
* buildDataSourceDSN: Resolves the database/sql driver name and DSN for an external data source.
* @param driver string, host string, port int, user string, password string, database string, sslmode string
* @return string, string, error
**/
func buildDataSourceDSN(driver, host string, port int, user, password, database, sslmode string) (string, string, error) {
	switch strings.ToLower(driver) {
	case DataSourcePostgres:
		if sslmode == "" {
			sslmode = "disable"
		}
		dsn := fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s",
			url.QueryEscape(user),
			url.QueryEscape(password),
			host,
			port,
			database,
			sslmode,
		)
		return DataSourcePostgres, dsn, nil
	case DataSourceMysql:
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, host, port, database)
		return DataSourceMysql, dsn, nil
	case DataSourceSqlite:
		if !utility.ValidStr(database, 1, []string{}) {
			return "", "", fmt.Errorf(msg.MSG_ARG_REQUIRED, "database")
		}
		return DataSourceSqlite, database, nil
	case DataSourceOracle:
		dsn := goora.BuildUrl(host, port, database, user, password, nil)
		return DataSourceOracle, dsn, nil
	case DataSourceSqlserver:
		query := url.Values{}
		query.Add("database", database)
		u := &url.URL{
			Scheme:   DataSourceSqlserver,
			User:     url.UserPassword(user, password),
			Host:     fmt.Sprintf("%s:%d", host, port),
			RawQuery: query.Encode(),
		}
		return DataSourceSqlserver, u.String(), nil
	default:
		return "", "", fmt.Errorf(msg.MSG_DRIVER_NOT_SUPPORTED, driver)
	}
}

/**
* jUploadDb: Imports rows from an external Postgres, MySQL, SQLite, Oracle or SQL Server table into a model.
* @param driver string, host string, port int, user string, password string, database string, sslmode string
* @param table string, fields []string, idField string, schema string, nameModel string, atribs map[string]string
* @return et.Items, error
**/
func (s *DB) jUploadDb(driver, host string, port int, user, password, database, sslmode, table string, fields []string, idField, schema, nameModel string, atribs map[string]string) (et.Items, error) {
	if !utility.ValidStr(driver, 1, []string{}) {
		return et.Items{}, fmt.Errorf(msg.MSG_ARG_REQUIRED, "driver")
	}

	if !utility.ValidStr(table, 1, []string{}) {
		return et.Items{}, fmt.Errorf(msg.MSG_ARG_REQUIRED, "table")
	}

	if !utility.ValidStr(idField, 1, []string{}) {
		return et.Items{}, fmt.Errorf(msg.MSG_ARG_REQUIRED, "idField")
	}

	if !utility.ValidStr(schema, 1, []string{}) {
		return et.Items{}, fmt.Errorf(msg.MSG_ARG_REQUIRED, "schema")
	}

	if !utility.ValidStr(nameModel, 1, []string{}) {
		return et.Items{}, fmt.Errorf(msg.MSG_ARG_REQUIRED, "nameModel")
	}

	model, err := s.GetModel(schema, nameModel)
	if err != nil {
		return et.Items{}, err
	}
	err = model.DefinePrimaryKeys(idField)
	if err != nil {
		return et.Items{}, err
	}

	driverName, dsn, err := buildDataSourceDSN(driver, host, port, user, password, database, sslmode)
	if err != nil {
		return et.Items{}, err
	}

	conn, err := sql.Open(driverName, dsn)
	if err != nil {
		return et.Items{}, err
	}
	defer conn.Close()

	err = conn.Ping()
	if err != nil {
		return et.Items{}, err
	}

	selected := "*"
	if len(fields) > 0 {
		selected = strings.Join(fields, ", ")
	}

	rows, err := conn.Query(fmt.Sprintf("SELECT %s FROM %s", selected, table))
	if err != nil {
		return et.Items{}, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return et.Items{}, err
	}

	tf := len(atribs)
	n := 0
	for rows.Next() {
		values := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}

		err = rows.Scan(ptrs...)
		if err != nil {
			return et.Items{}, err
		}

		item := et.Json{}
		for i, col := range columns {
			value := values[i]
			if bt, ok := value.([]byte); ok {
				value = string(bt)
			}

			if tf == 0 {
				item[col] = value
			} else {
				atrb, exists := atribs[col]
				if exists {
					item[atrb] = value
				}
			}
		}

		_, err = model.
			Insert(item).
			Exec()
		if err != nil {
			return et.Items{}, err
		}

		n++
	}
	if err = rows.Err(); err != nil {
		return et.Items{}, err
	}

	return et.Items{
		Ok:    true,
		Count: 1,
		Result: []et.Json{
			{
				"message": fmt.Sprintf("Uploaded %d rows", n),
			},
		},
	}, nil
}
