package jdb

import "github.com/cgalvisleon/et/et"

func (s *Cmd) update() (et.Items, error) {
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
		new := old.Clone()
		for k, v := range s.Data[0] {
			new[k] = v
		}

		for _, fn := range s.beforeUpdates {
			err := fn(s.tx, old, new)
			if err != nil {
				return et.Items{}, err
			}
		}

		for k, v := range new {
			v1 := old[k]
			if !et.EqualJSON(v1, v) {
				s.New[k] = v
			}
		}

		if s.New.IsEmpty() {
			result.Add(new)
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
		for _, fn := range s.afterUpdates {
			err := fn(s.tx, old, s.New)
			if err != nil {
				return et.Items{}, err
			}
		}

		result.Add(s.New)
	}

	return result, nil
}
