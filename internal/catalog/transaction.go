package catalog

import (
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
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
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Command   string    `json:"command"`
	ID        string    `json:"id"`
	New       et.Json   `json:"new"`
	Old       et.Json   `json:"old"`
	Status    string    `json:"status"`
	tx        *Tx       `json:"-"`
}

/**
* SetStatus: Sets the status of the transaction
* @param status string
**/
func (s *Transaction) SetStatus(status string) {
	now := timezone.Now()
	s.UpdatedAt = now
	s.Status = status
}

type Tx struct {
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	ID           string         `json:"id"`
	Transactions []*Transaction `json:"transactions"`
	Executions   []*Transaction `json:"executions"`
	Status       string         `json:"status"`
}

/**
* GetTx: Returns the transaction
* @param tx *Tx
* @return *Tx
**/
func GetTx(tx *Tx) *Tx {
	if tx == nil {
		now := timezone.Now()
		return &Tx{
			CreatedAt:    now,
			UpdatedAt:    now,
			ID:           reg.GenULID("tx"),
			Transactions: []*Transaction{},
			Executions:   []*Transaction{},
			Status:       PENDING,
		}
	}
	return tx
}

/**
* SetStatus: Sets the status of the transaction
* @param status string
**/
func (s *Tx) SetStatus(status string) {
	now := timezone.Now()
	s.UpdatedAt = now
	s.Status = status
}

/**
* Add: Adds a new transaction
* @param model *Model, command string, id string, old, new et.Json
* @return *Tx
**/
func (s *Tx) Add(model *Model, command, idx string, old, new et.Json) *Tx {
	now := timezone.Now()
	s.Transactions = append(s.Transactions, &Transaction{
		model:     model,
		CreatedAt: now,
		UpdatedAt: now,
		Command:   command,
		ID:        idx,
		New:       new,
		Old:       old,
		Status:    PENDING,
		tx:        s,
	})
	return s
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
			err := model.putObject(transaction.ID, transaction.Old)
			if err != nil {
				return err
			}
			transaction.SetStatus(ROLLED_BACK)
		case DELETE:
			err := model.putObject(transaction.ID, transaction.Old)
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
			err := model.putObject(transaction.ID, transaction.New)
			if err != nil {
				err = s.Rollback()
				if err != nil {
					return err
				}
			}
			transaction.SetStatus(COMMITTED)
		case UPDATE:
			err := model.putObject(transaction.ID, transaction.New)
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
