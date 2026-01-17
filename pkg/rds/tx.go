package rds

import (
	"fmt"
	"slices"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/josefina/pkg/store"
)

type record struct {
	cmd  Command
	idx  string
	data et.Json
}

type transaction struct {
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
		cmd:  cmd,
		idx:  idx,
		data: data,
	}
	s.records = append(s.records, item)
}

/**
* newTransaction: Creates a new transaction
* @param model *Model
* @return *transaction
**/
func newTransaction(model *Model) *transaction {
	return &transaction{
		model:   model,
		records: make([]*record, 0),
	}
}

type Tx struct {
	StartedAt    time.Time
	EndedAt      time.Time
	Id           string
	transactions []*transaction
}

func getTx(tx *Tx) {
	if tx != nil {
		return
	}

	tx = &Tx{
		StartedAt:    time.Now(),
		Id:           reg.GenULID("transaction"),
		transactions: make([]*transaction, 0),
	}
}

/**
* save: Saves the transaction
* @return error
**/
func (s *Tx) save() error {
	return nil
}

/**
* add: Adds data to the transaction
* @param name string, data et.Json
**/
func (s *Tx) add(model *Model, cmd Command, key string, data et.Json) {
	idx := slices.IndexFunc(s.transactions, func(t *transaction) bool { return t.model.Name == model.Name })
	if idx == -1 {
		tx := newTransaction(model)
		tx.add(cmd, key, data)
		s.transactions = append(s.transactions, tx)
		return
	}

	tx := s.transactions[idx]
	tx.add(cmd, key, data)
}

/**
* commit: Commits the transaction
* @return error
**/
func (s *Tx) commit() error {
	for _, tx := range s.transactions {
		model := tx.model
		for _, record := range tx.records {
			cmd := record.cmd
			idx := record.idx
			data := record.data
			for _, name := range model.Indexes {
				source := model.data[name]
				key := fmt.Sprintf("%v", data[name])
				if key == "" {
					continue
				}
				if cmd == DELETE {
					if name == INDEX {
						_, err := source.Delete(key)
						if err != nil {
							return err
						}
					} else {
						err := deleteIndex(source, key, idx)
						if err != nil {
							return err
						}
					}
				} else {
					if name == INDEX {
						err := source.Put(key, data)
						if err != nil {
							return err
						}
					} else {
						err := putIndex(source, key, idx)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}

	s.EndedAt = time.Now()
	return s.save()
}

/**
* putIndex
* @param store *store.FileStore, id string, idx any
* @return error
**/
func putIndex(store *store.FileStore, id string, idx any) error {
	result := map[string]bool{}
	exists, err := store.Get(id, result)
	if err != nil {
		return err
	}

	if !exists {
		result = map[string]bool{}
	}

	st := fmt.Sprintf("%v", idx)
	result[st] = true
	err = store.Put(id, result)
	if err != nil {
		return err
	}

	return nil
}

/**
* deleteIndex
* @param store *store.FileStore, id string, idx string
* @return error
**/
func deleteIndex(store *store.FileStore, id string, idx any) error {
	result := map[string]bool{}
	exists, err := store.Get(id, result)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	st := fmt.Sprintf("%v", idx)
	if _, ok := result[st]; !ok {
		return nil
	}

	delete(result, st)
	if len(result) == 0 {
		_, err := store.Delete(id)
		return err
	}

	err = store.Put(id, result)
	if err != nil {
		return err
	}

	return nil
}
