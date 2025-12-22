package jdb

func (s *DB) initCore() error {
	if err := defineModel(s); err != nil {
		return err
	}

	return nil
}
