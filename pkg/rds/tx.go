package rds

import (
	"fmt"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/josefina/pkg/store"
)

type transaction struct {
	model *Model
	cmd   Command
	idx   string
	data  et.Json
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
func (s *Tx) add(model *Model, cmd Command, idx string, data et.Json) {
	item := &transaction{
		model: model,
		cmd:   cmd,
		idx:   idx,
		data:  data,
	}
	s.transactions = append(s.transactions, item)
}

/**
* commit: Commits the transaction
* @return error
**/
func (s *Tx) commit() error {
	for _, tx := range s.transactions {
		model := tx.model
		idx := tx.idx
		data := tx.data
		for _, name := range model.Indexes {
			source := model.data[name]
			key := fmt.Sprintf("%v", data[name])
			if key == "" {
				continue
			}
			if tx.cmd == DELETE {
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
