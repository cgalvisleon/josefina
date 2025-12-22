package server

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
)

type Tx struct {
	Database string  `json:"database"`
	Id       int     `json:"id"`
	Query    et.Json `json:"query"`
	Result   et.Json `json:"result"`
}

/**
* Serialize
* @return []byte, error
**/
func (s *Tx) Serialize() ([]byte, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return bt, nil
}

/**
* ToJson
* @return et.Json
**/
func (s *Tx) ToJson() et.Json {
	bt, err := s.Serialize()
	if err != nil {
		return et.Json{}
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}
	}

	return result
}

type TxError struct {
	Database string `json:"database"`
	Id       int    `json:"id"`
	Error    []byte `json:"error"`
}

/**
* Serialize
* @return []byte, error
**/
func (s *TxError) Serialize() ([]byte, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return bt, nil
}

/**
* ToJson
* @return et.Json
**/
func (s *TxError) ToJson() et.Json {
	bt, err := s.Serialize()
	if err != nil {
		return et.Json{}
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}
	}

	return result
}

var transactions = make(map[int]*Tx)

func init() {
	transactions = make(map[int]*Tx)
}
