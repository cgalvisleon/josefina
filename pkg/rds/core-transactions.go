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
	transactions.defineAtrib("command", TpText, "")
	transactions.defineAtrib("status", TpText, "")
	transactions.definePrimaryKey(KEY)
	transactions.defineIndexe("type")
	if err := transactions.init(); err != nil {
		return err
	}

	return nil
}

/**
* setTransaction: Sets a transaction
* @param key string, cmd Command, status Status, args []interface{}
* @return string, error
**/
func setTransaction(key string, cmd Command, status Status, args []interface{}) (string, error) {
	if key == "" {
		key = transactions.getKey()
	}

	tx := getTx(nil)
	_, err := transactions.upsert(tx, et.Json{
		KEY:       key,
		"command": string(cmd),
		"args":    args,
		"status":  string(status),
	})
	if err != nil {
		return "", err
	}

	return key, tx.commit()
}
