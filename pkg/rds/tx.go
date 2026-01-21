package rds

import (
	"slices"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
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
	transactions.definePrimaryKey(INDEX)
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

	err := transactions.putData(key, data)
	if err != nil {
		return "", err
	}

	return key, nil
}

type record struct {
	tx      *Tx
	Command Command `json:"command"`
	Idx     string  `json:"idx"`
	Data    et.Json `json:"data"`
	Status  Status  `json:"status"`
}

/**
* commit: Commits the transaction
**/
func (s *record) commit() error {
	s.Status = Processed
	return s.tx.save()
}

/**
* newTransaction: Creates a new transaction
* @param model *Model
* @return *transaction
**/
type transaction struct {
	tx      *Tx
	Model   *Model    `json:"model"`
	Records []*record `json:"records"`
}

/**
* getFrom: Gets the from
* @return *From
**/
func (s *transaction) toJson() et.Json {
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
			"host":     s.Model.Host,
			"version":  s.Model.Version,
		},
		"records": records,
	}
}

/**
* add: Adds data to the transaction
* @param cmd Command, idx string, data et.Json
* @return void
**/
func (s *transaction) add(cmd Command, idx string, data et.Json) error {
	item := &record{
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
* newTransaction: Creates a new transaction
* @param model *Model
* @return *transaction
**/
func newTransaction(tx *Tx, model *Model) *transaction {
	return &transaction{
		tx:      tx,
		Model:   model,
		Records: make([]*record, 0),
	}
}

type Tx struct {
	StartedAt    time.Time      `json:"startedAt"`
	EndedAt      time.Time      `json:"endedAt"`
	Session      string         `json:"session"`
	Id           string         `json:"id"`
	Transactions []*transaction `json:"transactions"`
}

/**
* getTx: Creates a new transaction
* @param tx *Tx
* @return *Tx
**/
func getTx(tx *Tx) (*Tx, bool) {
	if tx != nil {
		return tx, false
	}

	tx = &Tx{
		StartedAt:    timezone.Now(),
		EndedAt:      time.Time{},
		Id:           reg.GenULID("transaction"),
		Transactions: make([]*transaction, 0),
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
	debug := envar.GetBool("DEBUG", false)
	if debug {
		logs.Debug(data.ToString())
	}

	_, err := setTransaction(s.Id, data)
	if err != nil {
		return err
	}

	return nil
}

/**
* getTx: Gets the transaction
* @return error
**/
func (s *Tx) getRecors(name string) []*record {
	idx := slices.IndexFunc(s.Transactions, func(item *transaction) bool { return item.Model.Name == name })
	if idx == -1 {
		return []*record{}
	}

	tra := s.Transactions[idx]
	return tra.Records
}

/**
* add: Adds data to the transaction
* @param name string, data et.Json
**/
func (s *Tx) add(model *Model, cmd Command, key string, data et.Json) error {
	var tx *transaction
	idx := slices.IndexFunc(s.Transactions, func(t *transaction) bool { return t.Model.Name == model.Name })
	if idx == -1 {
		tx = newTransaction(s, model)
		s.Transactions = append(s.Transactions, tx)
	} else {
		tx = s.Transactions[idx]
	}

	return tx.add(cmd, key, data)
}

/**
* commit: Commits the transaction
* @return error
**/
func (s *Tx) commit() error {
	for _, tx := range s.Transactions {
		model := tx.Model
		for _, record := range tx.Records {
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
