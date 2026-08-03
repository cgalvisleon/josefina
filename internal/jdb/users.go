package jdb

import (
	"errors"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/msg"
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
		"password":   s.Password,
	}
}

type Users struct {
	users map[string]*User
	mu    *sync.RWMutex
	store *Model
}

/**
* loadCache: Loads the cache
* @return *Cache, error
**/
func (s *DB) loadUsers() (*Users, error) {
	store, err := s.loadModel("", "users", 1, true)
	if err != nil {
		return nil, err
	}

	return &Users{
		users: make(map[string]*User),
		mu:    &sync.RWMutex{},
		store: store,
	}, nil
}

/**
* setUser: Sets a user in the cache
* @param id, username string
* @return error
**/
func (s *Users) setUser(user *User) error {
	s.mu.Lock()
	s.users[user.ID] = user
	s.mu.Unlock()

	return nil
}

/**
* GetUser
* @param username, password string
* @return (*et.Item, error)
**/
func (s *Users) getUser(id string) (*User, error) {
	s.mu.RLock()
	user, exists := s.users[id]
	s.mu.RUnlock()
	if !exists {
		return nil, errors.New(msg.MSG_USER_NOT_FOUND)
	}

	return user, nil
}
