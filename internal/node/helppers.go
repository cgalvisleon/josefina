package node

import (
	"slices"

	"github.com/cgalvisleon/et/et"
)

/**
* Select
* @param keys []string, object et.Json
* @return et.Json
**/
func Select(keys []string, object et.Json) et.Json {
	result := et.Json{}
	for _, key := range keys {
		val, ok := object[key]
		if ok {
			result[key] = val
		}
	}

	return result
}

/**
* Hidden
* @param keys []string, object et.Json
* @return et.Json
**/
func Hidden(keys []string, object et.Json) et.Json {
	result := et.Json{}
	for key, value := range object {
		if slices.Contains(keys, key) {
			continue
		}
		result[key] = value
	}

	return result
}
