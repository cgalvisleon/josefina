package jdb

func (s *Model) BeforeInsert(fn TriggerFunction) *Model {
	s.beforeInserts = append(s.beforeInserts, fn)
	return s
}

func (s *Model) BeforeUpdate(fn TriggerFunction) *Model {
	s.beforeUpdates = append(s.beforeUpdates, fn)
	return s
}

func (s *Model) BeforeDelete(fn TriggerFunction) *Model {
	s.beforeDeletes = append(s.beforeDeletes, fn)
	return s
}

func (s *Model) BeforeInsertOrUpdate(fn TriggerFunction) *Model {
	s.beforeInserts = append(s.beforeInserts, fn)
	s.beforeUpdates = append(s.beforeUpdates, fn)
	return s
}

func (s *Model) AfterInsert(fn TriggerFunction) *Model {
	s.afterInserts = append(s.afterInserts, fn)
	return s
}

func (s *Model) AfterUpdate(fn TriggerFunction) *Model {
	s.afterUpdates = append(s.afterUpdates, fn)
	return s
}

func (s *Model) AfterDelete(fn TriggerFunction) *Model {
	s.afterDeletes = append(s.afterDeletes, fn)
	return s
}

func (s *Model) AfterInsertOrUpdate(fn TriggerFunction) *Model {
	s.afterInserts = append(s.afterInserts, fn)
	s.afterUpdates = append(s.afterUpdates, fn)
	return s
}
