package jdb

import "github.com/cgalvisleon/et/et"

func (s *Cmd) delete() (et.Items, error) {
	if s.Model == nil {
		return et.Items{}, nil
	}

	from := s.Model
	current, err := from.
		Current(s.Wheres).
		All()
	if err != nil {
		return et.Items{}, err
	}

	result := et.Items{}
	for _, old := range current.Result {
		for _, fn := range s.beforeDeletes {
			err := fn(s.tx, old, s.New)
			if err != nil {
				return et.Items{}, err
			}
		}

		result, err := s.DB.Command(s)
		if err != nil {
			return et.Items{}, err
		}

		if !result.Ok {
			continue
		}

		old = result.First().Result
		for _, fn := range s.afterDeletes {
			err := fn(s.tx, old, s.New)
			if err != nil {
				return et.Items{}, err
			}
		}

		result.Add(old)
	}

	return result, nil
}
