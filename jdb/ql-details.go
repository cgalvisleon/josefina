package jdb

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

/**
* getDetailsTx
* @param tx *Tx, data et.Json
* @return
**/
func (s *Ql) getDetailsTx(tx *Tx, data et.Json) {
	for name, detail := range s.Details {
		to := detail.To
		conditions := WhereByKeys(data, detail.Keys)
		items, err := From(to, "A").
			Select(detail.Select...).
			WhereByConditions(conditions).
			LimitTx(tx, detail.Page, detail.Rows)
		if err != nil {
			logs.Error(err)
			continue
		}

		data[name] = items.Result
	}
}

/**
* getRollupsTx
* @param tx *Tx, data et.Json
* @return
**/
func (s *Ql) getRollupsTx(tx *Tx, data et.Json) {
	for name, rollup := range s.Rollups {
		to := rollup.To
		conditions := WhereByKeys(data, rollup.Keys)
		items, err := From(to, "A").
			Select(rollup.Select...).
			WhereByConditions(conditions).
			LimitTx(tx, rollup.Page, rollup.Rows)
		if err != nil {
			logs.Error(err)
			continue
		}

		item := items.First().Result
		if len(item) == 1 {
			for _, v := range item {
				data[name] = v
			}
		} else {
			data[name] = item
		}
	}
}

/**
* getCallsTx
* @param tx *Tx, data et.Json
* @return
**/
func (s *Ql) getCallsTx(tx *Tx, data et.Json) {
	for _, call := range s.Calcs {
		call(tx, data)
	}
}
