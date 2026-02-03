package core

import (
	"github.com/cgalvisleon/josefina/internal/mod"
)

var transactions *mod.Model

/**
* initTransactions: Initializes the transactions model
* @return error
**/
func initTransactions() error {
	if transactions != nil {
		return nil
	}

	db, err := mod.CoreDb()
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
* @param key string, tx *mod.Tx
* @return error
**/
func SetTransaction(key string, tx *mod.Tx) error {
	leader, ok := syn.getLeader()
	if ok {
		return syn.setTransaction(leader, key, tx)
	}

	err := initTransactions()
	if err != nil {
		return err
	}

	if key == "" {
		key = transactions.GenKey()
	}

	data := tx.ToJson()
	err = transactions.PutObject(key, data)
	if err != nil {
		return err
	}

	return nil
}
