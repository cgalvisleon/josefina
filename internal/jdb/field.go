package jdb

import (
	"encoding/json"
	"fmt"
	"time"
	"unsafe"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/josefina/internal/msg"
)

/**
* Ttl: Holds a cached byte value together with its creation time and expiration duration.
**/
type Ttl struct {
	Value     []byte        `json:"value"`
	CreatedAt time.Time     `json:"created_at"`
	Duration  time.Duration `json:"duration"`
}

/**
* newTtl: Creates a new TTL entry serializing value to bytes.
* @param value any, duration time.Duration
* @return *Ttl, error
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

	result := &Ttl{
		Value:     bt,
		CreatedAt: timezone.Now(),
		Duration:  duration,
	}
	return result, nil
}

/**
* MemorySize: Returns the memory size of the TTL entry.
* @return uintptr
**/
func (s *Ttl) MemorySize() uintptr {
	return unsafe.Sizeof(*s) + uintptr(cap(s.Value))
}

/**
* Bytes: Returns the size of the TTL entry in bytes.
* @return int
**/
func (s *Ttl) Bytes() int {
	return int(s.MemorySize())
}

/**
* KB: Returns the size of the TTL entry in kilobytes.
* @return float64
**/
func (s *Ttl) KB() float64 {
	return float64(s.Bytes()) / 1024
}

/**
* MB: Returns the size of the TTL entry in megabytes.
* @return float64
**/
func (s *Ttl) MB() float64 {
	return float64(s.Bytes()) / 1024 / 1024
}

func (s *Ttl) GB() float64 {
	return float64(s.Bytes()) / 1024 / 1024 / 1024
}

/**
* IsExpired
* @return bool
**/
func (s *Ttl) IsExpired() bool {
	if s.Duration == 0 {
		return false
	}
	return timezone.Now().After(s.CreatedAt.Add(s.Duration))
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
	INDEX      string = "jid"
	STATUS     string = "status"
	VERSION    string = "version"
	CREATED_AT string = "created_at"
	UPDATED_AT string = "updated_at"
)

/**
* TypeField: Classifies the structural role of a field within a model.
**/
type TypeField string

/**
* Str: Returns the string representation of the TypeField.
* @return string
**/
func (s TypeField) Str() string {
	return string(s)
}

const (
	TpAtrib       TypeField = "atrib"
	TpDetail      TypeField = "detail"
	TpMaster      TypeField = "master"
	TpRollup      TypeField = "rollup"
	TpCalc        TypeField = "calc"
	TpAggregation TypeField = "aggregation"
)

/**
* TypeData: Describes the value type stored in a field.
**/
type TypeData string

/**
* Str: Returns the string representation of the TypeData.
* @return string
**/
func (s TypeData) Str() string {
	return string(s)
}

/**
* Default: Returns the default value of the TypeData.
* @return interface{}
**/
func (s TypeData) Default() interface{} {
	switch s {
	case TpAny:
		return "	"
	case TpBytes:
		return []byte{}
	case TpInt:
		return 0
	case TpFloat:
		return 0.0
	case TpAutoIncrement:
		return 0
	case TpKey:
		return ""
	case TpText:
		return ""
	case TpMemo:
		return ""
	case TpDateTime:
		return timezone.Now()
	case TpBoolean:
		return false
	case TpJson:
		return et.Json{}
	case TpArrayJson:
		return []et.Json{}
	case TpReference:
		return ""
	}
	return nil
}

const (
	TpAny           TypeData = "any"
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
	TpArrayJson     TypeData = "array_json"
	TpReference     TypeData = "reference"
)

/**
* GetTypeData: Returns the TypeData for the given string.
* @param tp string
* @return TypeData, bool
**/
func GetTypeData(tp string) (TypeData, bool) {
	types := map[string]TypeData{
		"any":            TpAny,
		"bytes":          TpBytes,
		"int":            TpInt,
		"float":          TpFloat,
		"auto_increment": TpAutoIncrement,
		"key":            TpKey,
		"text":           TpText,
		"memo":           TpMemo,
		"datetime":       TpDateTime,
		"boolean":        TpBoolean,
		"json":           TpJson,
		"array_json":     TpArrayJson,
		"reference":      TpReference,
	}
	result, ok := types[tp]
	if !ok {
		return TpAny, false
	}
	return result, true
}

/**
* TpIndex: Identifies the index implementation (hash or btree).
**/
type TpIndex string

/**
* Str: Returns the string representation of the TpIndex.
* @return string
**/
func (s TpIndex) Str() string {
	return string(s)
}

const (
	TpIndexBTree TpIndex = "btree"
	TpIndexHash  TpIndex = "hash"
)

type Index struct {
	Name string  `json:"name"`
	Tag  string  `json:"tag"`
	Type TpIndex `json:"type"`
}

/**
* newIndex
* @param name string, tp TpIndex
* @return *Index
**/
func newIndex(model *Model, name string, tp TpIndex) *Index {
	return &Index{
		Name: name,
		Tag:  fmt.Sprintf("%s.%s", model.Name, name),
		Type: tp,
	}
}

/**
* TypeAggregation: Identifies the aggregation function applied to a rollup field.
**/
type TypeAggregation string

/**
* Str: Returns the string representation of the TypeAggregation.
* @return string
**/
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

/**
* Value: Returns the value of the specified attribute from the given item.
* @param item et.Json
* @return interface{}
**/
func (s *Field) Value(item et.Json) interface{} {
	return item.Get(s.Name)
}

/**
* Detail: Represents a detail in the database
**/
type Detail struct {
	to              *Model            `json:"-"`                 // Target model
	bridge          *Model            `json:"-"`                 // Bridge model
	Keys            map[string]string `json:"key"`               // Keys
	ToKeys          map[string]string `json:"to_key"`            // To keys
	Selects         []string          `json:"select"`            // Selects
	OnDeleteCascade bool              `json:"on_delete_cascade"` // On delete cascade
	OnUpdateCascade bool              `json:"on_update_cascade"` // On update cascade
}

/**
* ToJson: Returns the JSON representation of the Detail.
* @return et.Json
**/
func (s *Detail) ToJson() et.Json {
	to := et.Json{}
	if s.to != nil {
		to = et.Json{
			"database": s.to.Database,
			"schema":   s.to.Schema,
			"name":     s.to.Name,
		}
	}

	bridge := et.Json{}
	if s.bridge != nil {
		bridge = et.Json{
			"database": s.bridge.Database,
			"schema":   s.bridge.Schema,
			"name":     s.bridge.Name,
		}
	}

	return et.Json{
		"to":                to,
		"bridge":            bridge,
		"keys":              s.Keys,
		"select":            s.Selects,
		"on_delete_cascade": s.OnDeleteCascade,
		"on_update_cascade": s.OnUpdateCascade,
	}
}

/**
* load: Loads the detail from the JSON definition
* @param def et.Json
* @return error
**/
func (s *Detail) load(def et.Json) error {
	database := def.Str("database")
	schema := def.Str("schema")
	name := def.Str("name")
	db, err := GetDb(database)
	if err != nil {
		return err
	}

	to, err := db.GetModel(schema, name)
	if err != nil {
		return err
	}

	bridge, err := db.GetModel(schema, name)
	if err != nil {
		return err
	}

	s.to = to
	s.bridge = bridge
	return nil
}

/**
* newDetail
* @param to *Model, keys map[string]string, select []string, onDeleteCascade, onUpdateCascade bool
* @return *Detail
**/
func newDetail(to *Model, keys map[string]string, selects []string, onDeleteCascade, onUpdateCascade bool) *Detail {
	return &Detail{
		to:              to,
		Keys:            keys,
		Selects:         selects,
		OnDeleteCascade: onDeleteCascade,
		OnUpdateCascade: onUpdateCascade,
	}
}

func newMaster(to *Model, bridge *Model, keys map[string]string, toKeys map[string]string, selects []string) *Detail {
	return &Detail{
		to:              to,
		bridge:          bridge,
		Keys:            keys,
		ToKeys:          toKeys,
		Selects:         selects,
		OnDeleteCascade: true,
		OnUpdateCascade: true,
	}
}
