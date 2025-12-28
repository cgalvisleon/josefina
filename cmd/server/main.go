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
	st, _ := store.Open("./data", "metadata", "models", false)

	putData(st, 10001)
	// delete(st, "01KDE96HC1AMV2HZA8AEJZ9K72")
	// readData(st)
	// compact(st)
	// getData(st, "01KDEB4086CEGVD1G3FA69CMBB")
	logs.Logf("test", "Memory usage: %.2f MB", st.UseMemory())
	st.Close()
	logs.Log("test", "Finished:", st.ToString())
}

/**
* getData
* @param st *store.FileStore, id string
* @return et.Json
**/
func getData(st *store.FileStore, id string) et.Json {
	var result et.Json
	err := st.Get(id, &result)
	if err != nil {
		logs.Logf("test", "Error getting record %s: %v\n", id, err)
	} else {
		logs.Logf("test", "Retrieved record %s: %v\n", id, result.ToString())
	}

	return result
}

/**
* putData
* @param st *store.FileStore, limit int64
* @return
**/
func putData(st *store.FileStore, limit int64) {
	start := time.Now()
	var n int64

	// Write
	n = 0
	for {
		id := reg.GenULID(st.Name) //fmt.Sprintf("record_%d", n)
		_, err := st.Put(id, et.Json{
			"id": id,
			"ts": time.Now().UnixNano(),
		})
		if err != nil {
			panic(err)
		}

		n++
		if n >= limit {
			break
		}

		if n%10000 == 0 {
			fmt.Printf("writes=%d elapsed=%s\n", n, time.Since(start))
		}
	}
}

/**
* delete
* @param st *store.FileStore, id string
* @return
**/
func delete(st *store.FileStore, id string) {
	err, existed := st.Delete(id)
	if err != nil {
		logs.Logf("test", "Error deleting record %s: %v\n", id, err)
	} else if existed {
		logs.Logf("test", "Successfully deleted record %s\n", id)
	} else {
		logs.Logf("test", "Record %s not found\n", id)
	}
}

/**
* readData
* @param st *store.FileStore
* @return
**/
func readData(st *store.FileStore) {
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

/**
* compact
* @param st *store.FileStore
* @return
**/
func compact(st *store.FileStore) {
	err := st.Compact()
	if err != nil {
		logs.Logf("test", "Error compacting store: %v\n", err)
	} else {
		logs.Logf("test", "Successfully compacted store\n")
	}
}
