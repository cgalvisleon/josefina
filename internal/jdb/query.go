package jdb

import (
	"sync"

	"github.com/cgalvisleon/et/et"
)

type queryJob struct {
	fn     func(*DB, []et.Json) ([]et.Json, error)
	params []et.Json
}

/**
* runQueryJobs: Runs a set of query jobs in parallel and merges their results.
* @param db *DB
* @param jobs []queryJob
* @return et.Items, error
**/
func runQueryJobs(db *DB, jobs []queryJob) (et.Items, error) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	var resErr error
	result := et.Items{Result: []et.Json{}}

	for _, job := range jobs {
		wg.Add(1)
		go func(fn func(*DB, []et.Json) ([]et.Json, error), params []et.Json) {
			defer wg.Done()

			items, err := fn(db, params)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if resErr == nil {
					resErr = err
				}
				return
			}
			result.Add(items...)
		}(job.fn, job.params)
	}
	wg.Wait()

	if resErr != nil {
		return et.Items{}, resErr
	}

	result.Ok = true
	result.Count = len(result.Result)

	return result, nil
}

func defineQuery(db *DB, params []et.Json) ([]et.Json, error) {
	return []et.Json{}, nil
}

func describeQuery(db *DB, params []et.Json) ([]et.Json, error) {
	return []et.Json{}, nil
}

func insertQuery(db *DB, params []et.Json) ([]et.Json, error) {
	return []et.Json{}, nil
}

func updateQuery(db *DB, params []et.Json) ([]et.Json, error) {
	return []et.Json{}, nil
}

func deleteQuery(db *DB, params []et.Json) ([]et.Json, error) {
	return []et.Json{}, nil
}

func upsertQuery(db *DB, params []et.Json) ([]et.Json, error) {
	return []et.Json{}, nil
}

func bulkQuery(db *DB, params []et.Json) ([]et.Json, error) {
	return []et.Json{}, nil
}

func execQuery(db *DB, params []et.Json) ([]et.Json, error) {
	return []et.Json{}, nil
}
