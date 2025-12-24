package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/josefina/server/store"
)

func main() {
	st, _ := store.Open("./data", "test", 1, true, 100)

	start := time.Now()
	var count int64

	// Write
	for {
		id := reg.GetULID("")
		err := st.Put(id, et.Json{
			"id": id,
			"ts": time.Now().UnixNano(),
		})
		if err != nil {
			panic(err)
		}

		count++
		if count == 10 {
			break
		}

		if count%10000 == 0 {
			fmt.Printf("writes=%d elapsed=%s\n", count, time.Since(start))
		}
	}

	st.Iterate(func(id string, data []byte) bool {
		result := et.Json{}
		err := json.Unmarshal(data, &result)
		if err != nil {
			panic(err)
		}

		logs.Debug("iterate:", result.ToString())
		return true
	})
}
