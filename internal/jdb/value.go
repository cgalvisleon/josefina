package jdb

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type Value struct {
	Tp  TypeData
	Val any
	Str string
	Num float64
}

/**
* NewValue
* @param tp TypeData, val any
* @return (*Value, error)
**/
func NewValue(tp TypeData, val any) (*Value, error) {
	switch tp {
	case TpAny:
		return &Value{
			Tp:  tp,
			Val: val,
			Str: fmt.Sprintf("%v", val),
			Num: 0,
		}, nil

	case TpBytes:
		var bt []byte
		switch v := val.(type) {
		case []byte:
			bt = v
		case string:
			bt = []byte(v)
		default:
			var err error
			bt, err = json.Marshal(val)
			if err != nil {
				return nil, err
			}
		}
		return &Value{
			Tp:  tp,
			Val: bt,
			Str: string(bt),
			Num: 0,
		}, nil

	case TpInt, TpAutoIncrement:
		var n float64
		switch v := val.(type) {
		case int:
			n = float64(v)
		case int32:
			n = float64(v)
		case int64:
			n = float64(v)
		case float32:
			n = float64(v)
		case float64:
			n = v
		default:
			return nil, fmt.Errorf(msg.MSG_INVALID_TYPE)
		}
		return &Value{
			Tp:  tp,
			Val: val,
			Str: fmt.Sprintf("%d", int64(n)),
			Num: n,
		}, nil

	case TpFloat:
		var n float64
		switch v := val.(type) {
		case float64:
			n = v
		case float32:
			n = float64(v)
		case int:
			n = float64(v)
		case int64:
			n = float64(v)
		default:
			return nil, fmt.Errorf(msg.MSG_INVALID_TYPE)
		}
		return &Value{
			Tp:  tp,
			Val: val,
			Str: fmt.Sprintf("%g", n),
			Num: n,
		}, nil

	case TpKey, TpText, TpMemo, TpReference:
		return &Value{
			Tp:  tp,
			Val: val,
			Str: fmt.Sprintf("%v", val),
			Num: 0,
		}, nil

	case TpDateTime:
		var t time.Time
		switch v := val.(type) {
		case time.Time:
			t = v
		case string:
			var err error
			t, err = time.Parse(time.RFC3339Nano, v)
			if err != nil {
				t, err = time.Parse(time.RFC3339, v)
				if err != nil {
					return nil, fmt.Errorf(msg.MSG_INVALID_TYPE)
				}
			}
		default:
			return nil, fmt.Errorf(msg.MSG_INVALID_TYPE)
		}
		return &Value{
			Tp:  tp,
			Val: t,
			Str: t.Format(time.RFC3339Nano),
			Num: float64(t.UnixNano()),
		}, nil

	case TpBoolean:
		var b bool
		switch v := val.(type) {
		case bool:
			b = v
		case int:
			b = v != 0
		case float64:
			b = v != 0
		case string:
			b = v == "true" || v == "1"
		default:
			return nil, fmt.Errorf(msg.MSG_INVALID_TYPE)
		}
		num, str := 0.0, "false"
		if b {
			num, str = 1.0, "true"
		}
		return &Value{
			Tp:  tp,
			Val: b,
			Str: str,
			Num: num,
		}, nil

	case TpJson:
		var obj et.Json
		switch v := val.(type) {
		case et.Json:
			obj = v
		case map[string]interface{}:
			obj = et.Json(v)
		case []byte:
			if err := json.Unmarshal(v, &obj); err != nil {
				return nil, err
			}
		case string:
			if err := json.Unmarshal([]byte(v), &obj); err != nil {
				return nil, err
			}
		default:
			bt, err := json.Marshal(val)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(bt, &obj); err != nil {
				return nil, err
			}
		}
		str, err := json.Marshal(obj)
		if err != nil {
			return nil, err
		}
		return &Value{
			Tp:  tp,
			Val: obj,
			Str: string(str),
			Num: 0,
		}, nil

	case TpArrayJson:
		var arr []et.Json
		switch v := val.(type) {
		case []et.Json:
			arr = v
		case []byte:
			if err := json.Unmarshal(v, &arr); err != nil {
				return nil, err
			}
		case string:
			if err := json.Unmarshal([]byte(v), &arr); err != nil {
				return nil, err
			}
		default:
			bt, err := json.Marshal(val)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(bt, &arr); err != nil {
				return nil, err
			}
		}
		str, err := json.Marshal(arr)
		if err != nil {
			return nil, err
		}
		return &Value{
			Tp:  tp,
			Val: arr,
			Str: string(str),
			Num: 0,
		}, nil
	}

	return &Value{
		Tp:  tp,
		Val: val,
		Str: fmt.Sprintf("%v", val),
		Num: 0,
	}, nil
}

/**
* Value: Returns the native Go typed value for this field.
* @return any
**/
func (s *Value) Value() any {
	switch s.Tp {
	case TpKey, TpText, TpMemo, TpReference:
		return s.Str
	case TpInt, TpAutoIncrement:
		return int64(s.Num)
	case TpFloat:
		return s.Num
	case TpBoolean:
		return s.Num != 0
	case TpDateTime:
		if t, ok := s.Val.(time.Time); ok {
			return t
		}
		return s.Val
	case TpBytes:
		if bt, ok := s.Val.([]byte); ok {
			return bt
		}
		return s.Val
	case TpJson:
		if obj, ok := s.Val.(et.Json); ok {
			return obj
		}
		return s.Val
	case TpArrayJson:
		if arr, ok := s.Val.([]et.Json); ok {
			return arr
		}
		return s.Val
	default:
		return s.Val
	}
}
