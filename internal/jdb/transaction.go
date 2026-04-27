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

const (
	INSERT = "insert"
	UPDATE = "update"
	DELETE = "delete"
)

const (
	PENDING     = "pending"
	ROLLED_BACK = "rolled_back"
	COMMITTED   = "committed"
)

type Transaction struct {
	model     *Model    `json:"-"`
	From      *From     `json:"from"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Command   string    `json:"command"`
	ID        string    `json:"id"`
	New       et.Json   `json:"new"`
	Old       et.Json   `json:"old"`
	Ttl       *Ttl      `json:"ttl"`
	Status    string    `json:"status"`
	tx        *Tx       `json:"-"`
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
	source, err := db.transaction.Source()
	if err != nil {
		return err
	}

	source.ForEach(func(idx string, src []byte) (bool, error) {
		var tx Tx
		if err := json.Unmarshal(src, &tx); err != nil {
			return false, err
		}

		tx.db = db
		if tx.Status == PENDING {
			err := tx.Rollback()
			if err != nil {
				db.putError("rollback", err)
				return true, err
			}
			_, err = source.Delete(idx)
			if err != nil {
				return true, err
			}
		}
		return true, nil
	}, false, 0, 0, 1)

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
	ID           string         `json:"id"`
	Transactions []*Transaction `json:"transactions"`
	Executions   []*Transaction `json:"executions"`
	Status       string         `json:"status"`
	db           *DB            `json:"-"`
}

/**
* GetTx: Returns the transaction
* @param tx *Tx
* @return *Tx
**/
func GetTx(db *DB, tx *Tx) *Tx {
	if tx == nil {
		now := timezone.Now()
		return &Tx{
			CreatedAt:    now,
			UpdatedAt:    now,
			ID:           reg.GenULID("tx"),
			Transactions: []*Transaction{},
			Executions:   []*Transaction{},
			Status:       PENDING,
			db:           db,
		}
	}
	tx.db = db
	return tx
}

/**
* save: Saves the transaction
* @return error
**/
func (s *Tx) save() error {
	if s.db == nil {
		return errors.New(msg.MSG_DB_IS_NIL)
	}

	ttl := s.db.config.TransactionTTL
	err := s.db.transaction.Put(s.ID, s, ttl)
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
func (s *Tx) Add(model *Model, command, idx string, old, new et.Json, ttl *Ttl) (*Tx, error) {
	now := timezone.Now()
	s.Transactions = append(s.Transactions, &Transaction{
		model:     model,
		From:      model.From(),
		CreatedAt: now,
		UpdatedAt: now,
		Command:   command,
		ID:        idx,
		New:       new,
		Old:       old,
		Ttl:       ttl,
		tx:        s,
	})
	err := s.SetStatus(PENDING)
	if err != nil {
		return s, err
	}
	return s, nil
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
			err := model.deleteObject(transaction.ID, transaction.New)
			if err != nil {
				return err
			}
			transaction.SetStatus(ROLLED_BACK)
		case UPDATE:
			err := model.putObject(transaction.ID, transaction.Old, transaction.Ttl)
			if err != nil {
				return err
			}
			transaction.SetStatus(ROLLED_BACK)
		case DELETE:
			err := model.putObject(transaction.ID, transaction.Old, transaction.Ttl)
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
* @return error
**/
func (s *Tx) Commit() error {
	for _, transaction := range s.Transactions {
		model := transaction.model
		if model == nil {
			return nil
		}

		s.Executions = append([]*Transaction{transaction}, s.Executions...)
		switch transaction.Command {
		case INSERT:
			err := model.putObject(transaction.ID, transaction.New, transaction.Ttl)
			if err != nil {
				err = s.Rollback()
				if err != nil {
					return err
				}
			}
			transaction.SetStatus(COMMITTED)
		case UPDATE:
			err := model.putObject(transaction.ID, transaction.New, transaction.Ttl)
			if err != nil {
				err = s.Rollback()
				if err != nil {
					return err
				}
			}
			transaction.SetStatus(COMMITTED)
		case DELETE:
			err := model.deleteObject(transaction.ID, transaction.Old)
			if err != nil {
				err = s.Rollback()
				if err != nil {
					return err
				}
			}
			transaction.SetStatus(COMMITTED)
		}
	}
	s.SetStatus(COMMITTED)
	return nil
}

/**
* Items: Returns all items in the transaction
* @param model *Model
* @return []et.Json
**/
func (s *Tx) Items(model *Model) []et.Json {
	result := []et.Json{}
	for _, transaction := range s.Transactions {
		if transaction.model.Table == model.Table {
			result = append(result, transaction.New)
		}
	}
	return result
}

/**
* Equal: Returns all primary keys where field == key.
* @param model *Model, field string, value any
* @return []et.Json, bool
**/
func (s *Tx) Equal(model *Model, field string, value any) ([]et.Json, bool) {
	items := s.Items(model)
	result, ok := et.
		From(items, "A").
		Where(et.Eq(field, value)).
		All()

	return result, ok
}

/**
* NotEqual: Returns all primary keys where field != key.
* @param field string, value any
* @return []et.Json, error
**/
func (s *Tx) NotEqual(model *Model, field string, value any) ([]et.Json, bool) {
	items := s.Items(model)
	result, ok := et.
		From(items, "A").
		Where(et.Neg(field, value)).
		All()

	return result, ok
}
