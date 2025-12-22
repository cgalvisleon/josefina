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
func (s *Model) Query(query et.Json) (et.Items, error) {
	return et.Items{}, nil
}
