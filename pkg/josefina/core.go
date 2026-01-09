package josefina

func (s *Tennant) loadCore() error {
	db, err := s.getDb(packageName)
	if err != nil {
		return err
	}

	db.newModel("", "databases", 1)
	db.newModel("", "schemas", 1)
	db.newModel("", "models", 1)
	db.newModel("", "records", 1)
	db.newModel("", "references", 1)
	db.newModel("", "series", 1)
	db.newModel("", "users", 1)

	return nil
}
