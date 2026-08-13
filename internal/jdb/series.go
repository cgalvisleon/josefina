package jdb

import (
	"errors"
	"fmt"
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
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	Status    Status       `json:"status"`
	ID        string       `json:"id"`
	Tag       string       `json:"tag"`
	Format    string       `json:"format"`
	Value     int64        `json:"value"`
	mu        sync.RWMutex `json:"-"`
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

/**
* inc: Increments the value of the serie
* @return int64, string
**/
func (s *Serie) inc() (int64, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Value++
	return s.Value, fmt.Sprintf(s.Format, s.Value)
}

/**
* dec: Decrements the value of the serie
* @return int64, string
**/
func (s *Serie) dec() (int64, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Value--
	return s.Value, fmt.Sprintf(s.Format, s.Value)
}

type Series struct {
	Series map[string]*Serie
	mu     sync.RWMutex
	store  *Model
}

/**
* newSerie: Creates a new serie
* @param key, tag, format string, value int64
* @return (*Serie, error)
**/
func (s *Series) newSerie(key, tag, format string, value int64) (*Serie, error) {
	now := timezone.Now()

	if format == "" {
		format = "%08d"
	}

	result := &Serie{
		CreatedAt: now,
		UpdatedAt: now,
		Status:    ACTIVE,
		ID:        fmt.Sprintf("%s-%s", key, tag),
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

/**
* saveSerie: Saves a serie to the database
* @param serie *Serie
* @return error
**/
func (s *Series) saveSerie(serie *Serie) error {
	s.addSerie(serie)
	return s.store.put(serie.ID, serie)
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

/**
* setSerieFormat: Sets the format of a serie
* @param key, tag, format string
* @return error
**/
func (s *Series) setSerieFormat(key, tag, format string) error {
	id := fmt.Sprintf("%s-%s", key, tag)
	serie, err := s.getSerie(id)
	if err != nil {
		return err
	}

	serie.Format = format
	err = s.saveSerie(serie)
	if err != nil {
		return err
	}

	return nil
}

/**
* setSerieValue: Sets the value of a serie
* @param key, tag string, value int64
* @return error
**/
func (s *Series) setSerieValue(key, tag string, value int64) error {
	id := fmt.Sprintf("%s-%s", key, tag)
	serie, err := s.getSerie(id)
	if err != nil {
		return err
	}

	serie.Value = value
	err = s.saveSerie(serie)
	if err != nil {
		return err
	}

	return nil
}

/**
* incSerie: Increments the value of a serie
* @param key, tag string
* @return string, error
**/
func (s *Series) incSerie(key, tag string) (string, error) {
	id := fmt.Sprintf("%s-%s", key, tag)
	serie, err := s.getSerie(id)
	if err != nil {
		return "", err
	}

	_, result := serie.inc()
	err = s.saveSerie(serie)
	if err != nil {
		return "", err
	}

	return result, nil
}

/**
* decSerie: Decrements the value of a serie
* @param id string
* @return string, error
**/
func (s *Series) decSerie(id string) (string, error) {
	serie, err := s.getSerie(id)
	if err != nil {
		return "", err
	}

	_, result := serie.dec()
	err = s.saveSerie(serie)
	if err != nil {
		return "", err
	}

	return result, nil
}

/**
* loadSeries: Loads the series from the database
* @return error
**/
func (s *DB) loadSeries() error {
	store, err := s.loadModel(sysSchema, "series", 1, true)
	if err != nil {
		return err
	}

	result := &Series{
		Series: make(map[string]*Serie),
		store:  store,
	}

	s.series = result

	return nil
}

/**
* getSerie: Gets a serie from the cache
* @param id string
* @return (*Serie, error)
**/
func (s *DB) getSerie(id string) (*Serie, error) {
	return s.series.getSerie(id)
}

/**
* newSerie: Creates a new serie
* @param key, tag, format string, value int64
* @return (*Serie, error)
**/
func (s *DB) newSerie(key, tag, format string, value int64) (*Serie, error) {
	return s.series.newSerie(key, tag, format, value)
}

/**
* setSerieFormat: Sets the format of a serie
* @param key, tag, format string
* @return error
**/
func (s *DB) setSerieFormat(key, tag, format string) error {
	return s.series.setSerieFormat(key, tag, format)
}

/**
* setSerieValue: Sets the value of a serie
* @param key, tag string, value int64
* @return error
**/
func (s *DB) setSerieValue(key, tag string, value int64) error {
	return s.series.setSerieValue(key, tag, value)
}

/**
* incSerie: Increments the value of a serie
* @param key, tag string
* @return string, error
**/
func (s *DB) incSerie(key, tag string) (string, error) {
	return s.series.incSerie(key, tag)
}

/**
* decSerie: Decrements the value of a serie
* @param id string
* @return string, error
**/
func (s *DB) decSerie(id string) (string, error) {
	return s.series.decSerie(id)
}
