package main

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/josefina/internal/store"
)

func main() {
	test, err := store.Open("./data/collections", "./data/wald", "test", store.ReadWrite)
	if err != nil {
		panic(err)
	}

	_, _, err = test.Put("1", et.Json{
		"name": "Uno",
	})
	if err != nil {
		panic(err)
	}

	_, _, err = test.Put("2", et.Json{
		"name": "Dos",
	})
	if err != nil {
		panic(err)
	}

	test.ForEach(func(idx string, data []byte) (bool, error) {
		var item et.Json
		err := json.Unmarshal(data, &item)
		if err != nil {
			return false, err
		}

		logs.Debug(item.ToString())
		return true, nil
	}, true, 0, 0)
}
