package jdb

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

/**
* KeyType identifies the data type of an IndexKey.
**/
type KeyType uint8

const (
	KtString   KeyType = 1
	KtInt      KeyType = 2
	KtFloat    KeyType = 3
	KtBool     KeyType = 4
	KtDateTime KeyType = 5
)

/**
* IndexKey is a typed key for the B+ tree.
* Guarantees correct ordering per type: numeric for int/float/datetime/bool,
* lexicographic for strings.
**/
type IndexKey struct {
	tp  KeyType
	str string  // KtString
	num float64 // KtInt (int64 bits), KtFloat, KtBool (0/1), KtDateTime (UnixNano)
}

/**
* KeyString: Creates an IndexKey from a string value.
* @param v string
* @return IndexKey
**/
func KeyString(v string) IndexKey {
	return IndexKey{tp: KtString, str: v}
}

/**
* KeyInt: Creates an IndexKey from an int64 value.
* @param v int64
* @return IndexKey
**/
func KeyInt(v int64) IndexKey {
	return IndexKey{tp: KtInt, num: float64(v)}
}

/**
* KeyFloat: Creates an IndexKey from a float64 value.
* @param v float64
* @return IndexKey
**/
func KeyFloat(v float64) IndexKey {
	return IndexKey{tp: KtFloat, num: v}
}

/**
* KeyDateTime: Creates an IndexKey from a time.Time value.
* @param v time.Time
* @return IndexKey
**/
func KeyDateTime(v time.Time) IndexKey {
	return IndexKey{tp: KtDateTime, num: float64(v.UnixNano())}
}

/**
* KeyBool: Creates an IndexKey from a bool value.
* @param v bool
* @return IndexKey
**/
func KeyBool(v bool) IndexKey {
	if v {
		return IndexKey{tp: KtBool, num: 1}
	}
	return IndexKey{tp: KtBool, num: 0}
}

/**
* KeyFromAny: Builds an IndexKey from any JSON-unmarshalled value.
* Auto-detects type: whole float64 → KeyInt, RFC3339 string → KeyDateTime.
* @param v any
* @return IndexKey
**/
func KeyFromAny(v any) IndexKey {
	switch val := v.(type) {
	case string:
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			return KeyDateTime(t)
		}
		return KeyString(val)
	case float64:
		if val == math.Trunc(val) && !math.IsInf(val, 0) {
			return KeyInt(int64(val))
		}
		return KeyFloat(val)
	case int:
		return KeyInt(int64(val))
	case int32:
		return KeyInt(int64(val))
	case int64:
		return KeyInt(val)
	case bool:
		return KeyBool(val)
	case time.Time:
		return KeyDateTime(val)
	default:
		return KeyString(fmt.Sprintf("%v", val))
	}
}

/**
* Compare: Returns -1, 0 or 1.
* Keys of different types compare by their string representation (safe fallback).
* @param b IndexKey
* @return int
**/
func (a IndexKey) Compare(b IndexKey) int {
	if a.tp != b.tp {
		as, bs := a.String(), b.String()
		if as < bs {
			return -1
		}
		if as > bs {
			return 1
		}
		return 0
	}
	switch a.tp {
	case KtString:
		if a.str < b.str {
			return -1
		}
		if a.str > b.str {
			return 1
		}
		return 0
	default: // KtInt, KtFloat, KtBool, KtDateTime — all numeric
		if a.num < b.num {
			return -1
		}
		if a.num > b.num {
			return 1
		}
		return 0
	}
}

/**
* String: Returns a human-readable representation of the key.
* @return string
**/
func (k IndexKey) String() string {
	switch k.tp {
	case KtString:
		return k.str
	case KtInt:
		return fmt.Sprintf("%d", int64(k.num))
	case KtFloat:
		return fmt.Sprintf("%g", k.num)
	case KtBool:
		if k.num != 0 {
			return "true"
		}
		return "false"
	case KtDateTime:
		return time.Unix(0, int64(k.num)).UTC().Format(time.RFC3339Nano)
	}
	return ""
}

/**
* encodeKey: Serializes an IndexKey as a FileStore key: "<prefix>/<value>".
* @param key IndexKey
* @return string
**/
func encodeKey(key IndexKey) string {
	switch key.tp {
	case KtString:
		return "s/" + key.str
	case KtInt:
		return "i/" + strconv.FormatInt(int64(key.num), 10)
	case KtFloat:
		return "f/" + strconv.FormatFloat(key.num, 'g', -1, 64)
	case KtBool:
		if key.num != 0 {
			return "b/1"
		}
		return "b/0"
	case KtDateTime:
		return "d/" + strconv.FormatInt(int64(key.num), 10)
	}
	return "s/"
}

/**
* decodeKey: Reconstructs an IndexKey from a FileStore key string.
* @param s string
* @return IndexKey, error
**/
func decodeKey(s string) (IndexKey, error) {
	if len(s) < 2 || s[1] != '/' {
		return IndexKey{}, fmt.Errorf("invalid store key: %q", s)
	}
	tp, val := s[0], s[2:]
	switch tp {
	case 's':
		return KeyString(val), nil
	case 'i':
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return IndexKey{}, err
		}
		return KeyInt(n), nil
	case 'f':
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return IndexKey{}, err
		}
		return KeyFloat(f), nil
	case 'b':
		return KeyBool(val == "1"), nil
	case 'd':
		ns, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return IndexKey{}, err
		}
		return KeyDateTime(time.Unix(0, ns)), nil
	}
	return IndexKey{}, fmt.Errorf("unknown key type prefix: %c", tp)
}

// zero is the sentinel value that means "no bound" in Between/MoreEq/LessEq.
var zero IndexKey
