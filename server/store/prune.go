package store

// Prune implements garbage collection logic
func (s *FileStore) Prune() error {
	tag := "prune"
	s.metricStart(tag)
	defer s.metricEnd(tag, "completed")

	err := s.Compact()
	if err != nil {
		return err
	}

	err = s.RebuildIndexes()
	if err != nil {
		return err
	}

	return nil
}
