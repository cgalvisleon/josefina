package rds

import (
	"slices"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

var transactions *Model

/**
* initTransactions: Initializes the transactions model
* @return error
**/
func initTransactions() error {
	if transactions != nil {
		return nil
	}

	db, err := newDb(packageName, node.version)
	if err != nil {
		return err
	}

	transactions, err = db.newModel("", "transactions", true, 1)
	if err != nil {
		return err
	}
	transactions.definePrimaryKey(INDEX)
	if err := transactions.init(); err != nil {
		return err
	}

	return nil
}

/**
* setTransaction: Sets a Transaction
* @param key string, data et.Json
* @return string, error
**/
func setTransaction(key string, data et.Json) (string, error) {
	if key == "" {
		key = transactions.genKey()
	}

	err := transactions.putData(key, data)
	if err != nil {
		return "", err
	}

	return key, nil
}

type Record struct {
	tx      *Tx
	Command Command `json:"command"`
	Idx     string  `json:"idx"`
	Data    et.Json `json:"data"`
	Status  Status  `json:"status"`
}

/**
* commit: Commits the Transaction
**/
func (s *Record) commit() error {
	s.Status = Processed
	return s.tx.save()
}

/**
* newTransaction: Creates a new Transaction
* @param model *Model
* @return *Transaction
**/
type Transaction struct {
	tx      *Tx
	Model   *Model    `json:"model"`
	Records []*Record `json:"records"`
}

/**
* getFrom: Gets the from
* @return *From
**/
func (s *Transaction) toJson() et.Json {
	records := []et.Json{}
	for _, record := range s.Records {
		records = append(records, et.Json{
			"command": record.Command,
			"idx":     record.Idx,
			"data":    record.Data,
			"status":  record.Status,
		})
	}

	return et.Json{
		"model": et.Json{
			"database": s.Model.Database,
			"schema":   s.Model.Schema,
			"name":     s.Model.Name,
			"host":     s.Model.Host(),
			"version":  s.Model.Version,
		},
		"records": records,
	}
}

/**
* add: Adds data to the Transaction
* @param cmd Command, idx string, data et.Json
* @return void
**/
func (s *Transaction) add(cmd Command, idx string, data et.Json) error {
	item := &Record{
		tx:      s.tx,
		Command: cmd,
		Idx:     idx,
		Data:    data,
		Status:  Pending,
	}
	s.Records = append(s.Records, item)
	return s.tx.save()
}

/**
* newTransaction: Creates a new Transaction
* @param model *Model
* @return *Transaction
**/
func newTransaction(tx *Tx, model *Model) *Transaction {
	return &Transaction{
		tx:      tx,
		Model:   model,
		Records: make([]*Record, 0),
	}
}

type Tx struct {
	StartedAt    time.Time      `json:"startedAt"`
	EndedAt      time.Time      `json:"endedAt"`
	Id           string         `json:"id"`
	Transactions []*Transaction `json:"transactions"`
	isDebug      bool           `json:"-"`
}

/**
* getTx: Returns the Transaction for the session
* @param tx *Tx
* @return (*Tx, bool)
**/
func getTx(tx *Tx) (*Tx, bool) {
	if tx != nil {
		return tx, false
	}

	id := reg.GenULID("transaction")
	tx = &Tx{
		StartedAt:    timezone.Now(),
		EndedAt:      time.Time{},
		Id:           id,
		Transactions: make([]*Transaction, 0),
	}
	return tx, true
}

/**
* toJson
* @return et.Json, error
**/
func (s Tx) toJson() et.Json {
	transactions := []et.Json{}
	for _, transaction := range s.Transactions {
		transactions = append(transactions, transaction.toJson())
	}

	return et.Json{
		"startedAt":    s.StartedAt,
		"endedAt":      s.EndedAt,
		"id":           s.Id,
		"transactions": transactions,
	}
}

/**
* save: Saves the transaction
* @return error
**/
func (s *Tx) save() error {
	s.EndedAt = timezone.Now()
	data := s.toJson()
	if s.isDebug {
		logs.Debug(data.ToString())
	}

	_, err := setTransaction(s.Id, data)
	if err != nil {
		return err
	}

	return nil
}

/**
* getTx: Gets the Transaction
* @return error
**/
func (s *Tx) getRecors(name string) []*Record {
	idx := slices.IndexFunc(s.Transactions, func(item *Transaction) bool { return item.Model.Name == name })
	if idx == -1 {
		return []*Record{}
	}

	tra := s.Transactions[idx]
	return tra.Records
}

/**
* add: Adds data to the Transaction
* @param name string, data et.Json
**/
func (s *Tx) add(model *Model, cmd Command, key string, data et.Json) error {
	var tx *Transaction
	idx := slices.IndexFunc(s.Transactions, func(t *Transaction) bool { return t.Model.Name == model.Name })
	if idx == -1 {
		tx = newTransaction(s, model)
		s.Transactions = append(s.Transactions, tx)
	} else {
		tx = s.Transactions[idx]
	}

	return tx.add(cmd, key, data)
}

/**
* commit: Commits the Transaction
* @return error
**/
func (s *Tx) commit() error {
	for _, tr := range s.Transactions {
		model := tr.Model
		for _, record := range tr.Records {
			cmd := record.Command
			idx := record.Idx
			if cmd == DELETE {
				err := model.removeData(idx)
				if err != nil {
					return err
				}
			} else {
				data := record.Data
				err := model.putData(idx, data)
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

	return nil
}
