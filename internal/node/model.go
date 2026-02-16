package node

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/catalog"
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
func Commit(tx *catalog.Tx) error {
	for i, tr := range tx.Transactions {
		cmd := tr.Command
		idx := tr.Idx
		if cmd == catalog.DELETE {
			err := RemoveObject(tr.From, idx)
			if err != nil {
				return err
			}
		} else {
			data := tr.Data
			err := PutObject(tr.From, idx, data)
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
