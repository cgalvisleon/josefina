package jdb

import (
	"fmt"
	"time"

	"github.com/cgalvisleon/et/envar"
)

/**
* loadConfig: Loads or initializes the configuration for a database.
* @param db *DB
* @return error
**/
func (s *DB) loadConfig() error {
	model, err := s.Define(DModel{
		Name:    "config",
		IsCore:  true,
		Version: 1,
	})
	if err != nil {
		return err
	}

	err = model.Init()
	if err != nil {
		return err
	}

	var config *Config
	idx := fmt.Sprintf("config:%s", s.Name)
	exists, err := model.Get(idx, &config)
	if err != nil {
		return err
	}

	if !exists {
		ttl := time.Duration(envar.GetInt("TTL_TRANSACCION", 300)) * time.Second
		config = &Config{
			TransactionTTL: ttl,
			model:          model,
		}
	}

	s.config = config
	return nil
}
