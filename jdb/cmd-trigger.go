package jdb

/**
* BeforeInsert
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) BeforeInsert(fn TriggerFunction) *Cmd {
	s.beforeInserts = append(s.beforeInserts, fn)
	return s
}

/**
* BeforeUpdate
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) BeforeUpdate(fn TriggerFunction) *Cmd {
	s.beforeUpdates = append(s.beforeUpdates, fn)
	return s
}

/**
* BeforeDelete
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) BeforeDelete(fn TriggerFunction) *Cmd {
	s.beforeDeletes = append(s.beforeDeletes, fn)
	return s
}

/**
* AfterInsert
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) AfterInsert(fn TriggerFunction) *Cmd {
	s.afterInserts = append(s.afterInserts, fn)
	return s
}

/**
* AfterUpdate
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) AfterUpdate(fn TriggerFunction) *Cmd {
	s.afterUpdates = append(s.afterUpdates, fn)
	return s
}

/**
* AfterDelete
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) AfterDelete(fn TriggerFunction) *Cmd {
	s.afterDeletes = append(s.afterDeletes, fn)
	return s
}

/**
* BeforeInsertOrUpdate
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) BeforeInsertOrUpdate(fn TriggerFunction) *Cmd {
	s.beforeInserts = append(s.beforeInserts, fn)
	s.beforeUpdates = append(s.beforeUpdates, fn)
	return s
}

/**
* AfterInsertOrUpdate
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) AfterInsertOrUpdate(fn TriggerFunction) *Cmd {
	s.afterInserts = append(s.afterInserts, fn)
	s.afterUpdates = append(s.afterUpdates, fn)
	return s
}
