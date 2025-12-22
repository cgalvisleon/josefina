package jdb

import (
	"database/sql"
	"time"

	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

type Tx struct {
	CreatedAt time.Time `json:"created_at"`
	EndAt     time.Time `json:"end_at"`
	Id        string    `json:"id"`
	Committed bool      `json:"committed"`
	Tx        *sql.Tx   `json:"-"`
}

/**
* NewTx
* @return *Tx
**/
func NewTx() *Tx {
	now := timezone.NowTime()
	return &Tx{
		CreatedAt: now,
		EndAt:     now,
		Id:        reg.TagULID("tx", ""),
	}
}

/**
* InitTx
* @param tx *Tx
* @return *Tx
**/
func InitTx(tx *Tx) *Tx {
	if tx.Tx == nil {
		tx = NewTx()
	}

	return tx
}

/**
* Begin
* @param db *sql.DB
* @return error
**/
func (s *Tx) Begin(db *sql.DB) error {
	if s.Tx != nil {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	s.Tx = tx

	return nil
}

/**
* Commit
* @return error
**/
func (s *Tx) Commit() error {
	if s.Tx == nil {
		return nil
	}

	if s.Committed {
		return nil
	}

	err := s.Tx.Commit()
	s.Committed = true
	s.EndAt = timezone.NowTime()

	return err
}

/**
* Rollback
* @return error
**/
func (s *Tx) Rollback() error {
	if s.Tx == nil {
		return nil
	}

	if s.Committed {
		return nil
	}

	err := s.Tx.Rollback()
	s.Committed = true
	s.EndAt = timezone.NowTime()

	return err
}
