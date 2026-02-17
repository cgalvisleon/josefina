package node

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/msg"
)

var transactions *catalog.Model

/**
* initTransactions: Initializes the transactions model
* @return error
**/
func (s *Node) initTransactions() error {
	if transactions != nil {
		return nil
	}

	db, err := s.coreDb()
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
* @param tx *Tx
* @return error
**/
func (s *Node) setTransaction(tx *Tx) error {
	err := s.initTransactions()
	if err != nil {
		return err
	}

	data, err := tx.ToJson()
	if err != nil {
		return err
	}

	err = transactions.PutObject(tx.ID, data)
	if err != nil {
		return err
	}

	return nil
}

type Transaction struct {
	From    *catalog.From `json:"from"`
	Command Command       `json:"command"`
	Idx     string        `json:"idx"`
	Data    et.Json       `json:"data"`
	Status  Status        `json:"status"`
}

/**
* newTransaction: Creates a new Transaction
* @param from *From, cmd Command, idx string, data et.Json, status Status
* @return *Transaction
**/
func newTransaction(from *catalog.From, cmd Command, idx string, data et.Json, status Status) *Transaction {
	return &Transaction{
		From:    from,
		Command: cmd,
		Idx:     idx,
		Data:    data,
		Status:  status,
	}
}

type Tx struct {
	StartedAt    time.Time      `json:"startedAt"`
	EndedAt      time.Time      `json:"endedAt"`
	ID           string         `json:"id"`
	Transactions []*Transaction `json:"transactions"`
	isDebug      bool           `json:"-"`
}

/**
* GetTx: Returns the Transaction for the session
* @param tx *Tx
* @return (*Tx, bool)
**/
func GetTx(tx *Tx) (*Tx, bool) {
	if tx != nil {
		return tx, false
	}

	id := reg.GenULID("transaction")
	tx = &Tx{
		StartedAt:    timezone.Now(),
		EndedAt:      time.Time{},
		ID:           id,
		Transactions: make([]*Transaction, 0),
	}
	return tx, true
}

/**
* serialize
* @return []byte, error
**/
func (s *Tx) serialize() ([]byte, error) {
	result, err := json.Marshal(s)
	if err != nil {
		return []byte{}, err
	}

	return result, nil
}

/**
* ToJson
* @return et.Json, error
**/
func (s *Tx) ToJson() (et.Json, error) {
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
* Save
* @return error
**/
func (s *Tx) change() error {
	s.EndedAt = timezone.Now()
	if s.isDebug {
		data, err := s.ToJson()
		if err != nil {
			return err
		}
		logs.Debug(data.ToString())
	}

	return setTransaction(s)
}

/**
* AddTransaction: Adds data to the Transaction
* @param from *From, cmd Command, idx string, data et.Json
**/
func (s *Tx) AddTransaction(from *catalog.From, cmd Command, idx string, data et.Json) error {
	transaction := newTransaction(from, cmd, idx, data, Pending)
	s.Transactions = append(s.Transactions, transaction)
	return s.change()
}

/**
* SetStatus: Sets the status of a transaction
* @param idx int, status Status
* @return error
**/
func (s *Tx) SetStatus(idx int, status Status) error {
	tr := s.Transactions[idx]
	if tr == nil {
		return errors.New(msg.MSG_TRANSACTION_NOT_FOUND)
	}

	tr.Status = status
	s.Transactions[idx] = tr
	return s.change()
}

/**
* getRecors: Returns the records for the from
* @param from *catalog.From
* @return []et.Json
**/
func (s *Tx) getRecors(from *catalog.From) []et.Json {
	result := []et.Json{}
	for _, transaction := range s.Transactions {
		if transaction.From == from {
			result = append(result, transaction.Data)
		}
	}
	return result
}
