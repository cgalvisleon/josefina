package core

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/jdb"
)

var transactions *jdb.Model

/**
* initTransactions: Initializes the transactions model
* @return error
**/
func initTransactions() error {
	if transactions != nil {
		return nil
	}

	db, err := jdb.GetDb(database)
	if err != nil {
		return err
	}

	transactions, err = db.NewModel("", "transactions", true, 1)
	if err != nil {
		return err
	}
	if err := transactions.Init(); err != nil {
		return err
	}

	return nil
}

/**
* SetTransaction: Sets a Transaction
* @param key string, data et.Json
* @return string, error
**/
func SetTransaction(key string, data et.Json) (string, error) {
	err := initTransactions()
	if err != nil {
		return "", err
	}

	if key == "" {
		key = transactions.GenKey()
	}

	err = transactions.PutObject(key, data)
	if err != nil {
		return "", err
	}

	return key, nil
}
