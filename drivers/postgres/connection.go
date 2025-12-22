package postgres

import (
	"database/sql"
	"fmt"

	"github.com/cgalvisleon/et/et"
)

type Connection struct {
	Database string `json:"database"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	App      string `json:"app"`
	Version  int    `json:"version"`
}

/**
* ToJson
* @return et.Json
**/
func (s *Connection) ToJson() et.Json {
	return et.Json{
		"database": s.Database,
		"host":     s.Host,
		"port":     s.Port,
		"username": s.Username,
		"password": s.Password,
		"app":      s.App,
		"version":  s.Version,
	}
}

/**
* load
* @param params et.Json
* @return error
**/
func (s *Connection) Load(params et.Json) error {
	s.Database = params.Str("database")
	s.Host = params.Str("host")
	s.Port = params.Int("port")
	s.Username = params.Str("username")
	s.Password = params.Str("password")
	s.App = params.Str("app")
	s.Version = params.Int("version")

	return s.validate()
}

/**
* defaultChain
* @return string, error
**/
func (s *Connection) defaultChain() (string, error) {
	return fmt.Sprintf(`%s://%s:%s@%s:%d/%s?sslmode=disable&application_name=%s`, driver, s.Username, s.Password, s.Host, s.Port, "postgres", s.App), nil
}

/**
* chain
* @return string, error
**/
func (s *Connection) chain() (string, error) {
	err := s.validate()
	if err != nil {
		return "", err
	}

	result := fmt.Sprintf(`%s://%s:%s@%s:%d/%s?sslmode=disable&application_name=%s`, driver, s.Username, s.Password, s.Host, s.Port, s.Database, s.App)

	return result, nil
}

/**
* validate
* @return error
**/
func (s *Connection) validate() error {
	if s.Database == "" {
		return fmt.Errorf("database is required")
	}
	if s.Host == "" {
		return fmt.Errorf("host is required")
	}
	if s.Port == 0 {
		return fmt.Errorf("port is required")
	}
	if s.Username == "" {
		return fmt.Errorf("username is required")
	}

	if s.Password == "" {
		return fmt.Errorf("password is required")
	}

	if s.App == "" {
		return fmt.Errorf("app is required")
	}

	return nil
}

/**
* connectTo
* @param chain string
* @return *sql.DB, error
**/
func connectTo(chain string) (*sql.DB, error) {
	db, err := sql.Open(driver, chain)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
