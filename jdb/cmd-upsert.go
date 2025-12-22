package jdb

import "github.com/cgalvisleon/et/et"

func (s *Cmd) upsert() (et.Items, error) {
	if s.Model == nil {
		return et.Items{}, nil
	}

	from := s.Model
	data := s.Data[0]
	exists, err := from.
		WhereByPrimaryKeys(data).
		SetDebug(s.IsDebug).
		ItExists()
	if err != nil {
		return et.Items{}, err
	}

	if exists {
		s.Type = TypeUpdate
		s.WhereByPrimaryKeys(data)
		return s.update()
	}

	s.Type = TypeInsert
	return s.insert()
}
