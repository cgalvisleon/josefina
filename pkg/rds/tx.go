package rds

import (
	"encoding/json"
	"slices"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

type record struct {
	tx      *Tx
	command Command
	idx     string
	data    et.Json
	status  Status
}

/**
* commit: Commits the transaction
* @return error
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
func (s *transaction) add(cmd Command, idx string, data et.Json) {
	item := &record{
		tx:      s.tx,
		command: cmd,
		idx:     idx,
		data:    data,
		status:  Pending,
	}
	s.records = append(s.records, item)
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
* getTx: Gets the transaction
* @param tx *Tx
* @return *Tx
**/
func getTx(tx *Tx) *Tx {
	if tx != nil {
		return tx
	}

	tx = &Tx{
		startedAt:    timezone.Now(),
		endedAt:      time.Time{},
		id:           reg.GenULID("transaction"),
		transactions: make([]*transaction, 0),
	}
	tx.save()
	return tx
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
* ToJson
* @return et.Json, error
**/
func (s Tx) ToJson() (et.Json, error) {
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
	data, err := s.ToJson()
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
		tx.add(cmd, key, data)
		s.transactions = append(s.transactions, tx)
		return s.save()
	}

	tx := s.transactions[idx]
	tx.add(cmd, key, data)
	return s.save()
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
	return s.save()
}
