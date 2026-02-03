package jql

import "fmt"

/**
* Key
* @return string
**/
func modelKey(database, schema, name string) string {
	result := name
	if schema != "" {
		result = fmt.Sprintf("%s.%s", schema, result)
	}
	if database != "" {
		result = fmt.Sprintf("%s.%s", database, result)
	}
	return result
}
