package jdb

import "github.com/cgalvisleon/et/et"

/**
* SelectByJson
* @param query et.Json
* @return (et.Items, error)
**/
func (s *Model) SelectByJson(query et.Json) (et.Items, error) {
	from := query.MapStr("from")
	for _, v := range from {
		ql := From(s, v)
		selects := query.ArrayStr("select")
		ql.Select(selects...)
		wheres := query.ArrayJson("where")
		ql.WhereByJson(wheres)

		return ql.All()
	}

	return et.Items{}, nil
}

/**
* InsertByJson
* @param query et.Json
* @return (et.Items, error)
**/
func (s *Model) InsertByJson(query et.Json) (et.Items, error) {
	return et.Items{}, nil
}

/**
* UpdateByJson
* @param query et.Json
* @return (et.Items, error)
**/
func (s *Model) UpdateByJson(query et.Json) (et.Items, error) {
	return et.Items{}, nil
}

func (s *Model) UpsertByJson(query et.Json) (et.Items, error) {
	return et.Items{}, nil
}

/**
* DeleteByJson
* @param query et.Json
* @return (et.Items, error)
**/
func (s *Model) DeleteByJson(query et.Json) (et.Items, error) {
	return et.Items{}, nil
}

/**
* Query
* @param query et.Json
* @return (et.Items, error)
**/
func (s *Model) QueryByJson(query et.Json) (et.Items, error) {
	insert := query.Json("insert")
	if !insert.IsEmpty() {
		return s.InsertByJson(query)
	}

	update := query.Json("update")
	if !update.IsEmpty() {
		return s.UpdateByJson(query)
	}

	delete := query.Json("delete")
	if !delete.IsEmpty() {
		return s.DeleteByJson(query)
	}

	upsert := query.Json("upsert")
	if !upsert.IsEmpty() {
		return s.UpsertByJson(query)
	}

	return s.SelectByJson(query)
}
