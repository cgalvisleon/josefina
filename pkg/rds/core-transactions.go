package rds

import "github.com/cgalvisleon/et/et"

/**
* initTransactions: Initializes the transactions model
* @param db *DB
* @return error
**/
func initTransactions(db *DB) error {
	var err error
	transactions, err = db.newModel("", "transactions", true, 1)
	if err != nil {
		return err
	}
	transactions.defineAtrib(KEY, TpKey, "")
	transactions.definePrimaryKey(KEY)
	if err := transactions.init(); err != nil {
		return err
	}

	return nil
}

/**
* setTransaction: Sets a transaction
* @param key string, data et.Json
* @return string, error
**/
func setTransaction(key string, data et.Json) (string, error) {
	if key == "" {
		key = transactions.getKey()
	}

	tx := getTx(nil)
	data.Set(KEY, key)
	_, err := transactions.upsert(tx, data)
	if err != nil {
		return "", err
	}

	return key, tx.commit()
}
