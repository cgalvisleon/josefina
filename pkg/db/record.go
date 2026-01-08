package db

import (
	"encoding/json"
	"time"

	"github.com/cgalvisleon/et/et"
)

type Metadata struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int       `json:"version"`
}

type Record struct {
	Database string   `json:"database"`
	Model    string   `json:"model"`
	Id       string   `json:"id"`
	Metadata Metadata `json:"metadata"`
	Data     et.Json  `json:"data"`
}

/**
* Serialize
* @return []byte, error
**/
func (s *Record) Serialize() ([]byte, error) {
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
func (s *Record) ToJson() et.Json {
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
