package node

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/msg"
)

var (
	ErrorRecordNotFound      = errors.New(msg.MSG_RECORD_NOT_FOUND)
	ErrorPrimaryKeysNotFound = errors.New(msg.MSG_PRIMARY_KEYS_NOT_FOUND)
	ErrorModelNotFound       = errors.New(msg.MSG_MODEL_NOT_FOUND)
)

type TriggerFunction func(tx *Tx, old, new et.Json) error

type Command string

const (
	INSERT Command = "insert"
	UPDATE Command = "update"
	DELETE Command = "delete"
	UPSERT Command = "upsert"
)

type Cmd struct {
	model         *catalog.Model    `json:"-"`
	data          et.Json           `json:"-"`
	command       Command           `json:"-"`
	wheres        *Wheres           `json:"-"`
	beforeInserts []TriggerFunction `json:"-"`
	afterInserts  []TriggerFunction `json:"-"`
	beforeUpdates []TriggerFunction `json:"-"`
	afterUpdates  []TriggerFunction `json:"-"`
	beforeDeletes []TriggerFunction `json:"-"`
	afterDeletes  []TriggerFunction `json:"-"`
	isDebug       bool              `json:"-"`
}

/**
* newCmd: Creates a new command
* @param model *Model, command Command
* @return *Cmd
**/
func newCmd(model *catalog.Model) *Cmd {
	result := &Cmd{
		model:         model,
		data:          et.Json{},
		wheres:        newWhere(),
		beforeInserts: make([]TriggerFunction, 0),
		afterInserts:  make([]TriggerFunction, 0),
		beforeUpdates: make([]TriggerFunction, 0),
		afterUpdates:  make([]TriggerFunction, 0),
		beforeDeletes: make([]TriggerFunction, 0),
		afterDeletes:  make([]TriggerFunction, 0),
	}

	return result
}

/**
* IsDebug: Returns the debug mode
* @return *Cmd
**/
func (s *Cmd) IsDebug() *Cmd {
	s.isDebug = true
	return s
}

/**
* BeforeInsertFn
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) BeforeInsertFn(fn TriggerFunction) *Cmd {
	s.beforeInserts = append(s.beforeInserts, fn)
	return s
}

/**
* AfterInsertFn
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) AfterInsertFn(fn TriggerFunction) *Cmd {
	s.afterInserts = append(s.afterInserts, fn)
	return s
}

/**
* BeforeUpdateFn
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) BeforeUpdateFn(fn TriggerFunction) *Cmd {
	s.beforeUpdates = append(s.beforeUpdates, fn)
	return s
}

/**
* AfterUpdateFn
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) AfterUpdateFn(fn TriggerFunction) *Cmd {
	s.afterUpdates = append(s.afterUpdates, fn)
	return s
}

/**
* BeforeDeleteFn
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) BeforeDeleteFn(fn TriggerFunction) *Cmd {
	s.beforeDeletes = append(s.beforeDeletes, fn)
	return s
}

/**
* AfterDeleteFn
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) AfterDeleteFn(fn TriggerFunction) *Cmd {
	s.afterDeletes = append(s.afterDeletes, fn)
	return s
}

/**
* runTrigger
* @param event catalog.EventTrigger, tx *Tx, old et.Json, new et.Json
* @return error
**/
func (s *Cmd) runTrigger(event catalog.Trigger, tx *Tx, old, new et.Json) error {
	model := s.model
	vm := NewVm()
	vm.Set("self", model)
	vm.Set("tx", tx)
	vm.Set("old", old)
	vm.Set("new", new)
	script := string(event.Definition)
	_, err := vm.Run(script)
	if err != nil {
		return err
	}

	return nil
}

/**
* executeInsert
* @param tx *Tx
* @return et.Json, error
**/
func (s *Cmd) executeInsert(tx *Tx) (et.Json, error) {
	model := s.model
	if model == nil {
		return nil, errors.New(msg.MSG_MODEL_IS_NIL)

	}

	if s.data.IsEmpty() {
		return nil, errors.New(msg.MSG_NOT_DATA)
	}

	new := s.data

	// Validate required fields
	for _, name := range model.Required {
		if _, ok := new[name]; !ok {
			return nil, fmt.Errorf(msg.MSG_FIELD_REQUIRED, name)
		}
	}

	// Validate unique fields
	for _, name := range model.Unique {
		if _, ok := new[name]; !ok {
			return nil, fmt.Errorf(msg.MSG_FIELD_REQUIRED, name)
		}
		source, err := model.Store(name)
		if err != nil {
			return nil, fmt.Errorf(msg.MSG_STORE_NOT_FOUND, name)
		}
		key := fmt.Sprintf("%v", new[name])
		if source.IsExist(key) {
			return nil, errors.New(msg.MSG_RECORD_EXISTS)
		}
	}

	// Validate foreign keys
	for name, detail := range model.ForeignKeys {
		if _, ok := new[name]; !ok {
			return nil, fmt.Errorf(msg.MSG_FIELD_REQUIRED, name)
		}

		fk, ok := detail.Keys[name]
		if !ok {
			return nil, fmt.Errorf(msg.MSG_FOREIGN_KEY_NOT_FOUND, name)
		}

		key := fmt.Sprintf("%v", new[name])
		exists, err := node.IsExisted(detail.To, fk, key)
		if err != nil {
			return nil, err
		}

		if !exists {
			return nil, fmt.Errorf(msg.MSG_VIOLATE_FOREIGN_KEY, name)
		}
	}

	idx := new.ValStr("", catalog.INDEX)
	if idx == "" {
		idx = model.GenKey()
		new[catalog.INDEX] = idx
	}

	// Run before insert trigger function
	for _, fn := range s.beforeInserts {
		err := fn(tx, et.Json{}, new)
		if err != nil {
			return nil, err
		}
	}

	// Insert data into indexes
	tx.AddTransaction(model.From, INSERT, idx, new)

	// Run after insert trigger function
	for _, fn := range s.afterInserts {
		err := fn(tx, et.Json{}, new)
		if err != nil {
			return nil, err
		}
	}

	return et.Json{}, nil
}

/**
* executeUpdate
* @param tx *Tx
* @return []et.Json, error
**/
func (s *Cmd) executeUpdate(tx *Tx) ([]et.Json, error) {
	model := s.model
	if model == nil {
		return nil, errors.New(msg.MSG_MODEL_IS_NIL)
	}

	s.wheres.SetOwner(model)
	items, err := s.wheres.Run(tx)
	if err != nil {
		return nil, err
	}

	result := []et.Json{}
	if len(items) == 0 {
		return result, nil
	}

	data := s.data

	add := func(item et.Json) {
		result = append(result, item)
	}

	for _, old := range items {
		// Get index
		idx := old.ValStr("", catalog.INDEX)
		if idx == "" {
			return nil, ErrorRecordNotFound
		}

		// Update data
		new := old.Clone()
		for k, v := range data {
			new[k] = v
		}

		// Run before update trigger function
		for _, fn := range s.beforeUpdates {
			err := fn(tx, old, new)
			if err != nil {
				return nil, err
			}
		}

		// Insert data into indexes
		tx.AddTransaction(model.From, UPDATE, idx, new)

		// Run after update trigger function
		for _, fn := range s.afterUpdates {
			err := fn(tx, old, new)
			if err != nil {
				return nil, err
			}
		}

		add(new)
	}

	return result, nil
}

/**
* executeDelete
* @param tx *Tx
* @return []et.Json, error
**/
func (s *Cmd) executeDelete(tx *Tx) ([]et.Json, error) {
	model := s.model
	if model == nil {
		return nil, errors.New(msg.MSG_MODEL_IS_NIL)
	}

	s.wheres.SetOwner(model)
	items, err := s.wheres.Run(tx)
	if err != nil {
		return nil, err
	}

	result := []et.Json{}
	if len(items) == 0 {
		return result, nil
	}

	add := func(item et.Json) {
		result = append(result, item)
	}

	for _, old := range items {
		// Get index
		idx := old.ValStr("", catalog.INDEX)
		if idx == "" {
			return nil, ErrorRecordNotFound
		}

		// Run before delete trigger function
		for _, fn := range s.beforeDeletes {
			err := fn(tx, old, et.Json{})
			if err != nil {
				return nil, err
			}
		}

		// Delete data from indexes
		tx.AddTransaction(model.From, DELETE, idx, old)

		// Run after delete trigger function
		for _, fn := range s.afterDeletes {
			err := fn(tx, old, et.Json{})
			if err != nil {
				return nil, err
			}
		}

		add(old)
	}

	return result, nil
}

/**
* executeUpsert
* @param tx *Tx
* @return []et.Json, error
**/
func (s *Cmd) executeUpsert(tx *Tx) ([]et.Json, error) {
	model := s.model
	if model == nil {
		return nil, errors.New(msg.MSG_MODEL_IS_NIL)
	}

	new := s.data

	exists := true
	for _, name := range model.PrimaryKeys {
		source, err := model.Store(name)
		if err != nil {
			return nil, ErrorPrimaryKeysNotFound
		}
		key := fmt.Sprintf("%v", new[name])
		if key == "" {
			return nil, ErrorPrimaryKeysNotFound
		}
		exists = source.IsExist(key)
		if !exists {
			break
		}
		s.wheres.Add(Eq(name, key))
	}

	if !exists {
		result, err := s.executeInsert(tx)
		if err != nil {
			return nil, err
		}
		return []et.Json{result}, nil
	}

	return s.executeUpdate(tx)
}

/**
* Execute: Executes the command
* @param tx *Tx
* @return []et.Json, error
**/
func (s *Cmd) Execute(tx *Tx) ([]et.Json, error) {
	tx, commit := GetTx(tx)
	tx.isDebug = s.isDebug
	result := []et.Json{}
	switch s.command {
	case INSERT:
		item, err := s.executeInsert(tx)
		if err != nil {
			return nil, err
		}

		result = append(result, item)
	case UPDATE:
		items, err := s.executeUpdate(tx)
		if err != nil {
			return nil, err
		}

		result = append(result, items...)
	case DELETE:
		items, err := s.executeDelete(tx)
		if err != nil {
			return nil, err
		}

		result = append(result, items...)
	case UPSERT:
		items, err := s.executeUpsert(tx)
		if err != nil {
			return nil, err
		}

		result = append(result, items...)
	}

	if commit {
		err := Commit(tx)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

/**
* Insert: Inserts the model
* @param new et.Json
* @return *Cmd
**/
func (s *Cmd) Insert(new et.Json) *Cmd {
	s.command = INSERT
	s.data = new
	return s
}

/**
* Update: Updates the model
* @param data et.Json, where *Wheres
* @return *Cmd
**/
func (s *Cmd) Update(data et.Json) *Cmd {
	s.command = UPDATE
	s.data = data
	return s
}

/**
* Delete: Deletes the model
* @return *Cmd
**/
func (s *Cmd) Delete() *Cmd {
	s.command = DELETE
	return s
}

/**
* Upsert: Upserts the model
* @param new et.Json
* @return *Cmd
**/
func (s *Cmd) Upsert(new et.Json) *Cmd {
	s.command = UPSERT
	s.data = new
	return s
}

/**
* Where
* @param con *Condition
* @return *Cmd
**/
func (s *Cmd) Where(con *Condition) *Cmd {
	s.wheres.Add(con)
	return s
}

/**
* And
* @param con *Condition
* @return *Cmd
**/
func (s *Cmd) And(con *Condition) *Cmd {
	s.wheres.And(con)
	return s
}

/**
* Or
* @param con *Condition
* @return *Cmd
**/
func (s *Cmd) Or(con *Condition) *Cmd {
	s.wheres.Or(con)
	return s
}

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
