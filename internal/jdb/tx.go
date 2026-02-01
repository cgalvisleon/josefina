package jdb

import (
	"errors"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

type Transaction struct {
	From    *From   `json:"from"`
	Command Command `json:"command"`
	Idx     string  `json:"idx"`
	Data    et.Json `json:"data"`
	Status  Status  `json:"status"`
}

/**
* getFrom: Gets the from
* @return *From
**/
func (s *Transaction) toJson() et.Json {
	return et.Json{
		"from":    s.From,
		"command": s.Command,
		"idx":     s.Idx,
		"data":    s.Data,
		"status":  s.Status,
	}
}

/**
* newTransaction: Creates a new Transaction
* @param model *Model
* @return *Transaction
**/
func newTransaction(model *Model, cmd Command, idx string, data et.Json, status Status) *Transaction {
	return &Transaction{
		From:    model.From,
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
	onSave       OnSave         `json:"-"`
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
		ID:           id,
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
		"id":           s.ID,
		"transactions": transactions,
	}
}

/**
* SetOnSave
* @param onSave OnSave
**/
func (s *Tx) SetOnSave(onSave OnSave) {
	s.onSave = onSave
}

/**
* Save
* @return error
**/
func (s *Tx) Save() error {
	s.EndedAt = timezone.Now()
	data := s.toJson()
	if s.isDebug {
		logs.Debug(data.ToString())
	}

	if s.onSave != nil {
		err := s.onSave(s.ID, data)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* add: Adds data to the Transaction
* @param name string, data et.Json
**/
func (s *Tx) add(model *Model, cmd Command, idx string, data et.Json) error {
	transaction := newTransaction(model, cmd, idx, data, Pending)
	s.Transactions = append(s.Transactions, transaction)
	return s.Save()
}

/**
* setStatus: Sets the status of a transaction
* @param idx int, status Status
* @return error
**/
func (s *Tx) setStatus(idx int, status Status) error {
	tr := s.Transactions[idx]
	if tr == nil {
		return errors.New(msg.MSG_TRANSACTION_NOT_FOUND)
	}

	tr.Status = status
	s.Transactions[idx] = tr
	return s.Save()
}

/**
* getRecors: Returns the records for the from
* @param from *From
* @return []et.Json
**/
func (s *Tx) getRecors(from *From) []et.Json {
	result := []et.Json{}
	for _, transaction := range s.Transactions {
		if transaction.From == from {
			result = append(result, transaction.Data)
		}
	}
	return result
}

/**
* commit: Commits the Transaction
* @return error
**/
func (s *Tx) commit() error {
	for i, tr := range s.Transactions {
		cmd := tr.Command
		idx := tr.Idx
		if cmd == DELETE {
			err := node.RemoveObject(tr.From, idx)
			if err != nil {
				return err
			}
		} else {
			data := tr.Data
			err := node.PutObject(tr.From, idx, data)
			if err != nil {
				return err
			}
		}
		err := s.setStatus(i, Processed)
		if err != nil {
			return err
		}
	}

	return nil
}
