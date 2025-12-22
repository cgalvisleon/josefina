package store

/**
* Sync
* @return error
**/
func (s *FileStore) Sync() error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if s.active == nil || s.active.file == nil {
		return nil
	}

	return s.active.file.Sync()
}
