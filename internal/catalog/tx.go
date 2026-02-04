package catalog

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type Transaction struct {
	From    *From   `json:"from"`
	Command Command `json:"command"`
	Idx     string  `json:"idx"`
	Data    et.Json `json:"data"`
	Status  Status  `json:"status"`
}

/**
* newTransaction: Creates a new Transaction
* @param from *From, cmd Command, idx string, data et.Json, status Status
* @return *Transaction
**/
func newTransaction(from *From, cmd Command, idx string, data et.Json, status Status) *Transaction {
	return &Transaction{
		From:    from,
		Command: cmd,
		Idx:     idx,
		Data:    data,
		Status:  status,
	}
}

type Tx struct {
	StartedAt    time.Time           `json:"startedAt"`
	EndedAt      time.Time           `json:"endedAt"`
	ID           string              `json:"id"`
	Transactions []*Transaction      `json:"transactions"`
	onChange     func(et.Json) error `json:"-"`
	isDebug      bool                `json:"-"`
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
* Serialize
* @return []byte, error
**/
func (s *Tx) Serialize() ([]byte, error) {
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
	definition, err := s.Serialize()
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

func (s *Tx) SetOnChangeFn(fn func(et.Json) error) {
	s.onChange = fn
}

/**
* Save
* @return error
**/
func (s *Tx) change() error {
	s.EndedAt = timezone.Now()
	data, err := s.ToJson()
	if err != nil {
		return err
	}

	if s.isDebug {
		logs.Debug(data.ToString())
	}

	if s.onChange != nil {
		return s.onChange(data)
	}

	return nil
}

/**
* addTransaction: Adds data to the Transaction
* @param from *From, cmd Command, idx string, data et.Json
**/
func (s *Tx) addTransaction(from *From, cmd Command, idx string, data et.Json) error {
	transaction := newTransaction(from, cmd, idx, data, Pending)
	s.Transactions = append(s.Transactions, transaction)
	return s.change()
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
	return s.change()
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
			err := syn.removeObject(tr.From, idx)
			if err != nil {
				return err
			}
		} else {
			data := tr.Data
			err := syn.putObject(tr.From, idx, data)
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
