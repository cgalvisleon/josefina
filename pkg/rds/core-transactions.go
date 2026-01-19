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
	transactions.defineAtrib("type", TpText, "")
	transactions.defineAtrib("status", TpText, "")
	transactions.definePrimaryKey(KEY)
	transactions.defineIndexe("type")
	if err := transactions.init(); err != nil {
		return err
	}

	return nil
}

func CreateTransaction(cmd Command, args []interface{}) (string, error) {
	key := transactions.getKey()
	_, err := transactions.insert(nil, et.Json{
		KEY:      key,
		"type":   string(cmd),
		"args":   args,
		"status": string(Pending),
	})
	return key, err
}
