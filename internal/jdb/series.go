package jdb

import (
	"errors"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/timezone"
	"github.com/josefina/internal/msg"
)

type Status string

const (
	ACTIVE     Status = "active"
	ARCHIVED   Status = "archived"
	TO_DELETED Status = "deleted"
	EXPIRED    Status = "expired"
	CANCELLED  Status = "cancelled"
)

type Serie struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Status    Status    `json:"status"`
	ID        string    `json:"id"`
	Tag       string    `json:"tag"`
	Format    string    `json:"format"`
	Value     int64     `json:"value"`
}

func (s *Serie) ToJson() et.Json {
	return et.Json{
		"created_at": s.CreatedAt.Format(time.RFC3339),
		"updated_at": s.UpdatedAt.Format(time.RFC3339),
		"status":     s.Status,
		"id":         s.ID,
		"tag":        s.Tag,
		"format":     s.Format,
		"value":      s.Value,
	}
}

type Series struct {
	Series map[string]*Serie
	mu     sync.RWMutex
	store  *Model
}

func (s *Series) newSerie(key, tag, format string, value int64) (*Serie, error) {
	now := timezone.Now()
	result := &Serie{
		CreatedAt: now,
		UpdatedAt: now,
		Status:    ACTIVE,
		ID:        key,
		Tag:       tag,
		Format:    format,
		Value:     value,
	}

	err := s.saveSerie(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Series) saveSerie(serie *Serie) error {

	return nil
}

/**
* addSerie: Adds a serie to the cache
* @param serie *Serie
* @return error
**/
func (s *Series) addSerie(serie *Serie) error {
	s.mu.Lock()
	s.Series[serie.ID] = serie
	s.mu.Unlock()
	return nil
}

/**
* getSerie: Gets a serie from the cache
* @param id string
* @return (*Serie, error)
**/
func (s *Series) getSerie(id string) (*Serie, error) {
	s.mu.RLock()
	result, exists := s.Series[id]
	s.mu.RUnlock()
	if exists {
		return result, nil
	}

	exists, err := s.store.get(id, &result)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New(msg.MSG_SERIE_NOT_FOUND)
	}

	return s.Series[id], nil
}
