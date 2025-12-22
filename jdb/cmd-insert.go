package jdb

import "github.com/cgalvisleon/et/et"

/**
* insert
* @return et.Items, error
**/
func (s *Cmd) insert() (et.Items, error) {
	result := et.Items{}
	for _, new := range s.Data {
		s.New = new.Clone()
		for _, fn := range s.beforeInserts {
			err := fn(s.tx, et.Json{}, s.New)
			if err != nil {
				return et.Items{}, err
			}
		}

		if s.New.IsEmpty() {
			continue
		}

		result, err := s.DB.Command(s)
		if err != nil {
			return et.Items{}, err
		}

		if !result.Ok {
			continue
		}

		s.New = result.First().Result
		for _, fn := range s.afterInserts {
			err := fn(s.tx, et.Json{}, s.New)
			if err != nil {
				return et.Items{}, err
			}
		}

		result.Add(s.New)
	}

	return result, nil
}
