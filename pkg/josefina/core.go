package josefina

func (s *Tennant) loadCore() error {
	db, err := s.getDb(packageName)
	if err != nil {
		return err
	}

	db.newModel("", "databases", false, 1)
	db.newModel("", "schemas", false, 1)
	db.newModel("", "models", false, 1)
	db.newModel("", "records", false, 1)
	db.newModel("", "references", false, 1)
	db.newModel("", "series", false, 1)
	db.newModel("", "users", false, 1)

	return nil
}
