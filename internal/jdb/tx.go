package jdb

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type Cmd string

const (
	INSERT Cmd = "insert"
	UPDATE Cmd = "update"
	DELETE Cmd = "delete"
	UPSERT Cmd = "upsert"
)

const (
	PENDING     = "pending"
	ROLLED_BACK = "rolled_back"
	COMMITTED   = "committed"
)

type Transaction struct {
	Database  string    `json:"database"`
	Schema    string    `json:"schema"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Command   Cmd       `json:"command"`
	Idx       string    `json:"id"`
	New       et.Json   `json:"new"`
	Old       et.Json   `json:"old"`
	Status    string    `json:"status"`
	model     *Model    `json:"-"`
	tx        *Tx       `json:"-"`
}

/**
* load: Loads the transaction
* @param db *DB
* @return error
**/
func (s *Transaction) load(tx *Tx) error {
	model, err := tx.db.GetModel(s.Schema, s.Name)
	if err != nil {
		return err
	}
	s.model = model
	return nil
}

/**
* SetStatus: Sets the status of the transaction
* @param status string
* @return error
**/
func (s *Transaction) SetStatus(status string) error {
	now := timezone.Now()
	s.UpdatedAt = now
	s.Status = status
	return s.tx.save()
}

type Tx struct {
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	Idx          string         `json:"idx"`
	Transactions []*Transaction `json:"transactions"`
	Executions   []*Transaction `json:"executions"`
	Result       et.Json        `json:"result"`
	Status       string         `json:"status"`
	db           *DB            `json:"-"`
}

/**
* GetTx: Returns the transaction
* @param db *DB, tx *Tx
* @return *Tx
**/
func GetTx(db *DB, tx *Tx) *Tx {
	if tx == nil {
		idx := reg.GenULID("tx")
		now := timezone.Now()
		return &Tx{
			CreatedAt:    now,
			UpdatedAt:    now,
			Idx:          idx,
			Transactions: []*Transaction{},
			Executions:   []*Transaction{},
			Result:       et.Json{},
			Status:       PENDING,
			db:           db,
		}
	}
	tx.db = db
	return tx
}

/**
* load: Loads the transaction
* @param db *DB
* @return error
**/
func (s *Tx) load(db *DB) error {
	s.db = db
	for _, tx := range s.Transactions {
		err := tx.load(s)
		if err != nil {
			return err
		}
	}
	for _, tx := range s.Executions {
		err := tx.load(s)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* ToJson: Returns the transaction as JSON
* @return (et.Json, error)
**/
func (s *Tx) ToJson() (et.Json, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return et.Json{}, err
	}

	result := et.Json{}
	err = json.Unmarshal(bt, &result)
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
	if s.db == nil {
		return errors.New(msg.MSG_DB_IS_NIL)
	}

	jData, err := s.ToJson()
	if err != nil {
		return err
	}

	err = s.db.transaction.Put(s.Idx, jData)
	if err != nil {
		return err
	}

	return nil
}

/**
* SetDb: Sets the database
* @param db *DB
**/
func (s *Tx) SetDb(db *DB) {
	s.db = db
	for _, tx := range s.Transactions {
		tx.tx = s
	}
	for _, tx := range s.Executions {
		tx.tx = s
	}
}

/**
* SetStatus: Sets the status of the transaction
* @param status string
**/
func (s *Tx) SetStatus(status string) error {
	now := timezone.Now()
	s.UpdatedAt = now
	s.Status = status
	return s.save()
}

/**
* Add: Adds a new transaction
* @param model *Model, command string, id string, old, new et.Json, expiration time.Duration
* @return (*Tx, error)
**/
func (s *Tx) Add(model *Model, command Cmd, idx string, old, new et.Json) (*Transaction, error) {
	now := timezone.Now()
	result := &Transaction{
		Database:  model.Database,
		Schema:    model.Schema,
		Name:      model.Name,
		CreatedAt: now,
		UpdatedAt: now,
		Command:   command,
		Idx:       idx,
		New:       new,
		Old:       old,
		tx:        s,
		model:     model,
	}
	s.Transactions = append(s.Transactions, result)
	err := s.SetStatus(PENDING)
	if err != nil {
		return result, err
	}
	return result, nil
}

/**
* Rollback: Rolls back a transaction
* @return error
**/
func (s *Tx) Rollback() error {
	for _, transaction := range s.Executions {
		model := transaction.model
		if model == nil {
			return nil
		}

		switch transaction.Command {
		case INSERT:
			err := model.deleteObject(transaction.Idx, transaction.New)
			if err != nil {
				return err
			}
			transaction.SetStatus(ROLLED_BACK)
		case UPDATE:
			err := model.putObject(transaction.Idx, transaction.Old)
			if err != nil {
				return err
			}
			transaction.SetStatus(ROLLED_BACK)
		case DELETE:
			err := model.putObject(transaction.Idx, transaction.Old)
			if err != nil {
				return err
			}
			transaction.SetStatus(ROLLED_BACK)
		}
	}
	s.SetStatus(ROLLED_BACK)
	return nil
}

/**
* Commit: Commits a transaction
* @return (et.Item, error)
**/
func (s *Tx) Commit() (et.Item, error) {
	for _, transaction := range s.Transactions {
		model := transaction.model
		if model == nil {
			return et.Item{}, nil
		}

		s.Executions = append([]*Transaction{transaction}, s.Executions...)
		switch transaction.Command {
		case INSERT:
			err := model.putObject(transaction.Idx, transaction.New)
			if err != nil {
				err = s.Rollback()
				if err != nil {
					return et.Item{}, err
				}
			}
			transaction.SetStatus(COMMITTED)
		case UPDATE:
			err := model.putObject(transaction.Idx, transaction.New)
			if err != nil {
				err = s.Rollback()
				if err != nil {
					return et.Item{}, err
				}
			}
			transaction.SetStatus(COMMITTED)
		case DELETE:
			err := model.deleteObject(transaction.Idx, transaction.Old)
			if err != nil {
				err = s.Rollback()
				if err != nil {
					return et.Item{}, err
				}
			}
			transaction.SetStatus(COMMITTED)
		}
	}
	s.SetStatus(COMMITTED)
	return et.NewItem(s.Result), nil
}

/**
* Items: Returns all items in the transaction
* @param model *Model
* @return []et.Json
**/
func (s *Tx) Items(model *Model) []et.Json {
	result := []et.Json{}
	for _, transaction := range s.Transactions {
		if transaction.model.Key() == model.Key() {
			item := transaction.New
			item[INDEX] = transaction.Idx
			result = append(result, item)
		}
	}
	return result
}

/**
* loadTransaction: Loads the transaction model
* @param db *DB
* @return error
**/
func loadTransaction(db *DB) error {
	result, err := db.NewModel("", "transactions", true, 1)
	if err != nil {
		return err
	}

	err = result.Init()
	if err != nil {
		return err
	}

	db.transaction = result
	db.transaction.ForEachBt(func(idx string, src []byte) (bool, error) {
		var tx Tx
		if err := json.Unmarshal(src, &tx); err != nil {
			return false, err
		}

		err := tx.load(db)
		if err != nil {
			return false, err
		}
		if tx.Status == PENDING {
			err := tx.Rollback()
			if err != nil {
				_, er := db.putError(db.transaction.Key(), "rollback", tx.Idx, err)
				if er != nil {
					return true, er
				}
				return true, err
			}
			err = db.transaction.Remove(idx)
			if err != nil {
				_, er := db.putError(db.transaction.Key(), "rollback:delete", tx.Idx, err)
				if er != nil {
					return true, er
				}
				return true, err
			}
		}
		return true, nil
	}, false, 0, 0)

	return nil
}
