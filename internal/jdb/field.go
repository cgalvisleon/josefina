package jdb

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type Ttl struct {
	Value     []byte        `json:"value"`
	CreatedAt time.Time     `json:"created_at"`
	Duration  time.Duration `json:"duration"`
}

/**
* newTtl
* @param duration time.Duration
* @return *Ttl
**/
func newTtl(value any, duration time.Duration) (*Ttl, error) {
	bt, ok := value.([]byte)
	if !ok {
		var err error
		bt, err = json.Marshal(value)
		if err != nil {
			return nil, err
		}
	}

	return &Ttl{
		Value:     bt,
		CreatedAt: timezone.Now(),
		Duration:  duration,
	}, nil
}

/**
* IsExpired
* @return bool
**/
func (s *Ttl) IsExpired() bool {
	if s.Duration == 0 {
		return false
	}
	return time.Now().After(s.CreatedAt.Add(s.Duration))
}

/**
* GetExpiresAt
* @return time.Time
**/
func (s *Ttl) GetExpiresAt() time.Time {
	return s.CreatedAt.Add(s.Duration)
}

const (
	ID         string = "id"
	INDEX      string = "_idx"
	TTL        string = "_ttl"
	STATUS     string = "status"
	VERSION    string = "version"
	PROJECT_ID string = "project_id"
	TENANT_ID  string = "tenant_id"
	CREATED_AT string = "created_at"
	UPDATED_AT string = "updated_at"
)

type TypeField string

func (s TypeField) Str() string {
	return string(s)
}

const (
	TpAtrib       TypeField = "atrib"
	TpDetail      TypeField = "detail"
	TpRollup      TypeField = "rollup"
	TpCalc        TypeField = "calc"
	TpAggregation TypeField = "aggregation"
)

type TypeData string

func (s TypeData) Str() string {
	return string(s)
}

const (
	TpBytes         TypeData = "bytes"
	TpInt           TypeData = "int"
	TpFloat         TypeData = "float"
	TpAutoIncrement TypeData = "auto_increment"
	TpKey           TypeData = "key"
	TpText          TypeData = "text"
	TpMemo          TypeData = "memo"
	TpDateTime      TypeData = "datetime"
	TpBoolean       TypeData = "boolean"
	TpJson          TypeData = "json"
	TpModel         TypeData = "model"
)

type TypeAggregation string

func (s TypeAggregation) Str() string {
	return string(s)
}

/**
* GetAggregation
* @param tp string
* @return TypeAggregation
**/
func GetAggregation(tp string) TypeAggregation {
	aggregation := map[string]TypeAggregation{
		"count": TpCount,
		"sum":   TpSum,
		"avg":   TpAvg,
		"max":   TpMax,
		"min":   TpMin,
		"exp":   TpExp,
	}

	result, ok := aggregation[tp]
	if !ok {
		return TpExp
	}
	return result
}

const (
	TpCount TypeAggregation = "count"
	TpSum   TypeAggregation = "sum"
	TpAvg   TypeAggregation = "avg"
	TpMax   TypeAggregation = "max"
	TpMin   TypeAggregation = "min"
	TpExp   TypeAggregation = "exp"
)

type Field struct {
	from         *Model      `json:"-"`
	Name         string      `json:"name"`
	TypeField    TypeField   `json:"type_field"`
	TypeData     TypeData    `json:"type_data"`
	DefaultValue interface{} `json:"default_value"`
}

/**
* newField: Creates a new field
* @param from *Model, name string, tpField TypeField, tpData TypeData, defaultValue interface{}
* @return *Field, error
**/
func newField(from *Model, name string, tpField TypeField, tpData TypeData, defaultValue interface{}) (*Field, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, name)
	}

	return &Field{
		from:         from,
		Name:         name,
		TypeField:    tpField,
		TypeData:     tpData,
		DefaultValue: defaultValue,
	}, nil
}
