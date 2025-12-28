package store

// Prune implements garbage collection logic
func (s *FileStore) Prune() error {
	tag := "prune"
	s.metricStart(tag)
	defer s.metricEnd(tag, "completed")

	// TODO: implement prune logic
	// For now, just trigger a compaction to clean up tombstones
	err := s.Compact()
	if err != nil {
		return err
	}

	return nil
}
