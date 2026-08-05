package jdb

import (
	"errors"
	"sync"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/utility"
	"github.com/josefina/internal/msg"
)

type User struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
}

func (s *User) ToJson() et.Json {
	return et.Json{
		"created_at": s.CreatedAt.Format(time.RFC3339),
		"updated_at": s.UpdatedAt.Format(time.RFC3339),
		"id":         s.ID,
		"username":   s.Username,
	}
}

type Users struct {
	users map[string]*User
	mu    *sync.RWMutex
	store *Model
}

/**
* initUser: Initializes the user
* @return error
**/
func (s *Users) initUser() error {
	count, err := s.store.count()
	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	userAdmin := envar.GetStr("USER_ADMIN", "admin")
	passwordAdmin := envar.GetStr("PASSWORD_ADMIN", "admin")
	_, err = s.newUser(userAdmin, passwordAdmin)
	if err != nil {
		return err
	}

	return nil
}

/**
* newUser: Creates a new user
* @param username, password string
* @return error
**/
func (s *Users) newUser(username, password string) (*User, error) {
	hash, err := utility.HashSHA512(password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	result := &User{
		CreatedAt: now,
		UpdatedAt: now,
		ID:        reg.UUID(),
		Username:  username,
		Password:  hash,
	}

	err = s.saveUser(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* saveUser: Saves a user to the database
* @param user *User
* @return error
**/
func (s *Users) saveUser(user *User) error {
	s.addUser(user)
	return s.store.put(user.Username, user)
}

/**
* addUser: Adds a user to the cache
* @param id, username string
* @return error
**/
func (s *Users) addUser(user *User) {
	s.mu.Lock()
	s.users[user.Username] = user
	s.mu.Unlock()
}

/**
* GetUser
* @param username, password string
* @return (*et.Item, error)
**/
func (s *Users) getUser(username string) (*User, error) {
	s.mu.RLock()
	result, exists := s.users[username]
	s.mu.RUnlock()
	if exists {
		return result, nil
	}

	exists, err := s.store.get(username, &result)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New(msg.MSG_USER_NOT_FOUND)
	}

	return result, nil
}

/**
* loadCache: Loads the cache
* @return error
**/
func (s *DB) loadUsers() error {
	store, err := s.loadModel(sysSchema, "users", 1, true)
	if err != nil {
		return err
	}

	result := &Users{
		users: make(map[string]*User),
		mu:    &sync.RWMutex{},
		store: store,
	}

	err = result.initUser()
	if err != nil {
		return err
	}

	s.users = result

	return nil
}

/**
* getUser: Gets a user from the database
* @param username string
* @return (*User, error)
**/
func (s *DB) getUser(username string) (*User, error) {
	return s.users.getUser(username)
}

/**
* newUser: Creates a new user
* @param username, password string
* @return (*User, error)
**/
func (s *DB) newUser(username, password string) (*User, error) {
	return s.users.newUser(username, password)
}
