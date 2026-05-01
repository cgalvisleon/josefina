package jdb

import "github.com/cgalvisleon/et/et"

/**
* loadUsers: Load the users
* @return error
**/
func (s *Node) loadUsers() error {
	var err error
	s.users, err = s.catalog.Define(Define{
		Name:    "users",
		IsCore:  true,
		Version: 1,
		Fields: map[string]DefineField{
			"username": {
				Type:    TpText,
				Default: "",
			},
			"password": {
				Type:    TpText,
				Default: "",
			},
		},
		PrimaryKeys: []string{
			"username",
		},
		Required: []DefineIndex{
			{Name: "password"},
		},
	})
	if err != nil {
		return err
	}

	err = s.users.DefineIndexes("email", "password")
	if err != nil {
		return err
	}

	if err = s.users.Init(); err != nil {
		return err
	}

	return nil
}

/**
* GetUser
* @param username, password string
* @return (*et.Item, error)
**/
func (s *Node) GetUser(username, password string) (et.Item, error) {
	result, err := From(s.users).
		Where(Eq("email", username)).
		And(Eq("password", password)).
		First(nil)
	if err != nil {
		return et.Item{}, err
	}

	return result, nil
}

/**
* CreateUser
* @param username, password string
* @return (*et.Item, error)
**/
func (s *Node) CreateUser(username, password string) (et.Item, error) {
	result, err := s.users.
		Insert(et.Json{
			"email":    username,
			"password": password,
		}).
		One()
	if err != nil {
		return et.Item{}, err
	}

	return result, nil
}
