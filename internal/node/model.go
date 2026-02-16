package node

import (
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/msg"
)

var (
	ErrorRecordNotFound      = errors.New(msg.MSG_RECORD_NOT_FOUND)
	ErrorPrimaryKeysNotFound = errors.New(msg.MSG_PRIMARY_KEYS_NOT_FOUND)
	ErrorModelNotFound       = errors.New(msg.MSG_MODEL_NOT_FOUND)
)

/**
* Insert: Inserts the model
* @param data et.Json
* @return *Cmd
**/
func Insert(m *catalog.Model, data et.Json) *Cmd {
	result := newCmd(m)
	result.Insert(data)
	return result
}

/**
* update: Updates the model
* @param data et.Json
* @return *Cmd
**/
func Update(m *catalog.Model, data et.Json) *Cmd {
	result := newCmd(m)
	result.Update(data)
	return result
}

/**
* Delete: Deletes the model
* @return *Cmd
**/
func Delete(m *catalog.Model) *Cmd {
	result := newCmd(m)
	result.Delete()
	return result
}

/**
* Upsert: Upserts the model
* @param data et.Json
* @return *Cmd
**/
func Upsert(m *catalog.Model, data et.Json) *Cmd {
	result := newCmd(m)
	result.Upsert(data)
	return result
}

/**
* Selects: Returns the select
* @param fields ...string
* @return *Wheres
**/
func Selects(m *catalog.Model, fields ...string) *Wheres {
	result := newWhere()
	result.SetOwner(m)
	for _, field := range fields {
		result.selects = append(result.selects, field)
	}
	return result
}

/**
* commit: Commits the Transaction
* @return error
**/
func Commit(tx *Tx) error {
	for i, tr := range tx.Transactions {
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
		err := tx.SetStatus(i, catalog.Processed)
		if err != nil {
			return err
		}
	}

	return nil
}
