package main

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/store"
)

func main() {
	test, err := store.Open("./data", "test", store.ReadWrite)
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

	fmt.Println(test)
}
