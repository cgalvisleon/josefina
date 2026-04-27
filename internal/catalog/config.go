package catalog

import (
	"fmt"
	"time"

	"github.com/cgalvisleon/et/envar"
)

func loadConfig(db *DB) error {
	model, err := db.NewModel("", "conf", true, 1)
	if err != nil {
		return err
	}

	err = model.Init()
	if err != nil {
		return err
	}

	var config *Config
	idx := fmt.Sprintf("config:%s", db.Name)
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

	db.config = config
	return nil
}
