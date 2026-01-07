package utilities

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/cgalvisleon/et/et"
)

/**
* Normalize
* @param input string
* @return string
**/
func Normalize(input string) string {
	// 1. Quitar espacios al inicio y final
	s := strings.TrimSpace(input)

	// 2. Reemplazar uno o más espacios por _
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, "_")

	// 3. Eliminar todo lo que no sea letra, número o _
	s = regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(s, "")

	// 4. Garantizar que no empiece con número
	s = regexp.MustCompile(`^[0-9]+`).ReplaceAllString(s, "")

	return s
}

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
