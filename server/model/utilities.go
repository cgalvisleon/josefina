package model

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
)

/**
* Serialize
* @return []byte, error
**/
func ToSerialize(v any) ([]byte, error) {
	bt, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	return bt, nil
}

/**
* ToJson
* @return et.Json
**/
func ToJson(v any) et.Json {
	bt, err := ToSerialize(v)
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
