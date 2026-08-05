package jdb

import (
	"errors"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/josefina/internal/msg"
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

/**
* defineQuery: Executes a define query
* @param db *DB, params []et.Json
* @return []et.Json, error
**/
func defineQuery(db *DB, params []et.Json) ([]et.Json, error) {
	result := []et.Json{}
	for _, param := range params {
		for name := range param {
			define := param.Json(name)
			switch name {
			case "database":
				model, err := db.server.defineDatabase(define)
				if err != nil {
					return []et.Json{}, err
				}
				result = append(result, model)
			case "user":
				database := define.Str("database")
				dbp, exists := db.server.getDb(database)
				if !exists {
					return []et.Json{}, errors.New(msg.MSG_DB_NOT_FOUND)
				}
				model, err := dbp.defineUser(define)
				if err != nil {
					return []et.Json{}, err
				}
				result = append(result, model)
			case "schema":
				database := define.Str("database")
				dbp, exists := db.server.getDb(database)
				if !exists {
					return []et.Json{}, errors.New(msg.MSG_DB_NOT_FOUND)
				}
				model, err := dbp.defineSchema(define)
				if err != nil {
					return []et.Json{}, err
				}
				result = append(result, model)
			case "model":
				database := define.Str("database")
				dbp, exists := db.server.getDb(database)
				if !exists {
					return []et.Json{}, errors.New(msg.MSG_DB_NOT_FOUND)
				}
				model, err := dbp.defineModel(define)
				if err != nil {
					return []et.Json{}, err
				}
				result = append(result, model)
			}
		}
	}
	return result, nil
}

/**
* describeQuery: Executes a describe query
* @param db *DB, params []et.Json
* @return []et.Json, error
**/
func describeQuery(db *DB, params []et.Json) ([]et.Json, error) {
	result := []et.Json{}
	for _, param := range params {
		for name := range param {
			define := param.Json(name)
			switch name {
			case "database":
				model, err := db.server.describeDatabase(define)
				if err != nil {
					return []et.Json{}, err
				}
				result = append(result, model)
			case "user":
				model, err := db.describeUser(define)
				if err != nil {
					return []et.Json{}, err
				}
				result = append(result, model)
			case "schema":
				model, err := db.describeSchema(define)
				if err != nil {
					return []et.Json{}, err
				}
				result = append(result, model)
			case "model":
				model, err := db.describeModel(define)
				if err != nil {
					return []et.Json{}, err
				}
				result = append(result, model)
			}
		}
	}
	return result, nil
}

/**
* insertQuery: Executes a insert query
* @param db *DB, params []et.Json
* @return []et.Json, error
**/
func insertQuery(db *DB, params []et.Json) ([]et.Json, error) {
	result := []et.Json{}
	for _, param := range params {
		items, err := db.insertQuery(param)
		if err != nil {
			return []et.Json{}, err
		}
		result = append(result, items...)
	}
	return result, nil
}

/**
* updateQuery: Executes a update query
* @param db *DB, params []et.Json
* @return []et.Json, error
**/
func updateQuery(db *DB, params []et.Json) ([]et.Json, error) {
	result := []et.Json{}
	for _, param := range params {
		items, err := db.updateQuery(param)
		if err != nil {
			return []et.Json{}, err
		}
		result = append(result, items...)
	}
	return result, nil
}

/**
* deleteQuery: Executes a delete query
* @param db *DB, params []et.Json
* @return []et.Json, error
**/
func deleteQuery(db *DB, params []et.Json) ([]et.Json, error) {
	result := []et.Json{}
	for _, param := range params {
		items, err := db.deleteQuery(param)
		if err != nil {
			return []et.Json{}, err
		}
		result = append(result, items...)
	}
	return result, nil
}

/**
* upsertQuery: Executes a upsert query
* @param db *DB, params []et.Json
* @return []et.Json, error
**/
func upsertQuery(db *DB, params []et.Json) ([]et.Json, error) {
	result := []et.Json{}
	for _, param := range params {
		items, err := db.upsertQuery(param)
		if err != nil {
			return []et.Json{}, err
		}
		result = append(result, items...)
	}
	return result, nil
}

/**
* bulkQuery: Executes a bulk query
* @param db *DB, params []et.Json
* @return []et.Json, error
**/
func bulkQuery(db *DB, params []et.Json) ([]et.Json, error) {
	result := []et.Json{}
	for _, param := range params {
		items, err := db.bulkQuery(param)
		if err != nil {
			return []et.Json{}, err
		}
		result = append(result, items...)
	}
	return result, nil
}

/**
* execQuery: Executes a exec query
* @param db *DB, params []et.Json
* @return []et.Json, error
**/
func execQuery(db *DB, params []et.Json) ([]et.Json, error) {
	return []et.Json{}, nil
}
