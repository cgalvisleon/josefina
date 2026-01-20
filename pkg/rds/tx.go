package rds

import (
	"encoding/json"
	"slices"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

var transactions *Model

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

	err := transactions.put(key, data)
	if err != nil {
		return "", err
	}

	return key, nil
}

type record struct {
	tx      *Tx
	command Command
	idx     string
	data    et.Json
	status  Status
}

/**
* commit: Commits the transaction
**/
func (s *record) commit() error {
	s.status = Processed
	return s.tx.save()
}

/**
* newTransaction: Creates a new transaction
* @param model *Model
* @return *transaction
**/
type transaction struct {
	tx      *Tx
	model   *Model
	records []*record
}

/**
* add: Adds data to the transaction
* @param cmd Command, idx string, data et.Json
* @return void
**/
func (s *transaction) add(cmd Command, idx string, data et.Json) error {
	item := &record{
		tx:      s.tx,
		command: cmd,
		idx:     idx,
		data:    data,
		status:  Pending,
	}
	s.records = append(s.records, item)
	return s.tx.save()
}

/**
* newTransaction: Creates a new transaction
* @param model *Model
* @return *transaction
**/
func newTransaction(tx *Tx, model *Model) *transaction {
	return &transaction{
		tx:      tx,
		model:   model,
		records: make([]*record, 0),
	}
}

type Tx struct {
	startedAt    time.Time
	endedAt      time.Time
	id           string
	transactions []*transaction
}

/**
* getTx: Creates a new transaction
* @param tx *Tx
* @return *Tx
**/
func getTx(tx *Tx) (*Tx, error) {
	if tx != nil {
		return tx, nil
	}

	tx = &Tx{
		startedAt:    timezone.Now(),
		endedAt:      time.Time{},
		id:           reg.GenULID("transaction"),
		transactions: make([]*transaction, 0),
	}
	if err := tx.save(); err != nil {
		return nil, err
	}

	return tx, nil
}

/**
* serialize
* @return []byte, error
**/
func (s Tx) serialize() ([]byte, error) {
	result, err := json.Marshal(s)
	if err != nil {
		return []byte{}, err
	}

	return result, nil
}

/**
* toJson
* @return et.Json, error
**/
func (s Tx) toJson() (et.Json, error) {
	definition, err := s.serialize()
	if err != nil {
		return et.Json{}, err
	}

	result := et.Json{}
	err = json.Unmarshal(definition, &result)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* save: Saves the transaction
* @return error
**/
func (s *Tx) save() error {
	data, err := s.toJson()
	if err != nil {
		return err
	}

	_, err = setTransaction(s.id, data)
	if err != nil {
		return err
	}

	return nil
}

/**
* add: Adds data to the transaction
* @param name string, data et.Json
**/
func (s *Tx) add(model *Model, cmd Command, key string, data et.Json) error {
	idx := slices.IndexFunc(s.transactions, func(t *transaction) bool { return t.model.Name == model.Name })
	if idx == -1 {
		tx := newTransaction(s, model)
		s.transactions = append(s.transactions, tx)
	}

	tx := s.transactions[idx]
	return tx.add(cmd, key, data)
}

/**
* commit: Commits the transaction
* @return error
**/
func (s *Tx) commit() error {
	for _, tx := range s.transactions {
		model := tx.model
		for _, record := range tx.records {
			cmd := record.command
			idx := record.idx
			if cmd == DELETE {
				err := model.remove(idx)
				if err != nil {
					return err
				}
			} else {
				data := record.data
				err := model.put(idx, data)
				if err != nil {
					return err
				}
			}
			err := record.commit()
			if err != nil {
				return err
			}
		}
	}

	s.endedAt = timezone.Now()
	return nil
}
