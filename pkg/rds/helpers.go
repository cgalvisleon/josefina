package rds

import (
	"reflect"
	"strings"
	"time"
)

/**
* numberToFloat64: Converts a number to float64
* @param v any
* @return float64, reflect.Kind, bool
**/
func numberToFloat64(v any) (float64, reflect.Kind, bool) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return 0, reflect.Invalid, false
	}

	// Si llega un puntero, opcionalmente lo resolvemos
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return 0, reflect.Invalid, false
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), rv.Kind(), true

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return float64(rv.Uint()), rv.Kind(), true

	case reflect.Float32, reflect.Float64:
		return rv.Float(), rv.Kind(), true

	default:
		return 0, rv.Kind(), false
	}
}

/**
* numberToInt64: Converts a number to int64
* @param v any
* @return int64, bool
**/
func numberToInt64(v any) (int64, bool) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return 0, false
	}

	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return 0, false
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int(), true
	default:
		return 0, false
	}
}

/**
* isSignedIntKind
* @param k reflect.Kind
* @return bool
**/
func isSignedIntKind(k reflect.Kind) bool {
	return k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 || k == reflect.Int32 || k == reflect.Int64
}

/**
* isUnsignedIntKind
* @param k reflect.Kind
* @return bool
**/
func isUnsignedIntKind(k reflect.Kind) bool {
	return k == reflect.Uint || k == reflect.Uint8 || k == reflect.Uint16 || k == reflect.Uint32 || k == reflect.Uint64 || k == reflect.Uintptr
}

/**
* matchLikeStar: Matches a string with a pattern
* @param value string, pattern string
* @return bool
**/
func matchLikeStar(value, pattern string) bool {
	// "*" = match todo
	if pattern == "*" {
		return true
	}

	starts := strings.HasPrefix(pattern, "*")
	ends := strings.HasSuffix(pattern, "*")

	core := strings.Trim(pattern, "*")

	// si es "" después de trim (ej: "**") => match todo
	if core == "" {
		return true
	}

	switch {
	// *abc*
	case starts && ends:
		return strings.Contains(value, core)

	// *abc
	case starts && !ends:
		return strings.HasSuffix(value, core)

	// abc*
	case !starts && ends:
		return strings.HasPrefix(value, core)

	// abc (sin comodín)
	default:
		return value == pattern
	}
}

/**
* equalsAny: Compares two values
* @param a any, b any
* @return bool, error
**/
func equalsAny(a, b any) (bool, error) {
	// time.Time
	if ta, ok := a.(time.Time); ok {
		tb, ok := b.(time.Time)
		if !ok {
			return false, nil
		}
		return ta.Equal(tb), nil
	}

	// string
	if sa, ok := a.(string); ok {
		sb, ok := b.(string)
		if !ok {
			return false, nil
		}
		return sa == sb, nil
	}

	// numbers (usa tu helper numberToFloat64 del paso anterior)
	af, _, okA := numberToFloat64(a)
	if okA {
		bf, _, okB := numberToFloat64(b)
		if !okB {
			return false, nil
		}
		return af == bf, nil
	}

	// fallback: solo para tipos comparables
	ra := reflect.ValueOf(a)
	rb := reflect.ValueOf(b)

	if !ra.IsValid() || !rb.IsValid() {
		return false, nil
	}

	// si no son comparables, no se puede hacer ==
	if !ra.Type().Comparable() || !rb.Type().Comparable() {
		return false, nil
	}

	// si son tipos distintos pero comparables, no son iguales
	if ra.Type() != rb.Type() {
		return false, nil
	}

	return ra.Interface() == rb.Interface(), nil
}
