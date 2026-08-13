package jdb

import (
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/google/uuid"
)

type InstanceStatus string

const (
	InstanceStatusPending InstanceStatus = "pending"
	InstanceStatusRunning InstanceStatus = "running"
	InstanceStatusDone    InstanceStatus = "done"
	InstanceStatusFailed  InstanceStatus = "failed"
)

type Instance struct {
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DoneAt      time.Time      `json:"done_at"`
	ID          string         `json:"id"`
	Code        string         `json:"code"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Ctx         et.Json        `json:"ctx"`
	Params      et.Json        `json:"params"`
	Status      InstanceStatus `json:"status"`
}

func (s *Instance) ToJson() et.Json {
	return et.Json{
		"created_at":  s.CreatedAt.Format(time.RFC3339),
		"updated_at":  s.UpdatedAt.Format(time.RFC3339),
		"done_at":     s.DoneAt.Format(time.RFC3339),
		"id":          s.ID,
		"code":        s.Code,
		"title":       s.Title,
		"description": s.Description,
		"ctx":         s.Ctx,
		"params":      s.Params,
		"status":      s.Status,
	}
}

type Instances struct {
	instances map[string]*Instance
	mu        sync.RWMutex
	store     *Model
}

func (s *Instances) newInstance(title, description string, ctx et.Json) (*Instance, error) {
	now := timezone.Now()
	result := &Instance{
		CreatedAt:   now,
		UpdatedAt:   now,
		DoneAt:      time.Time{},
		ID:          reg.UUID(),
		Code:        uuid.New().String(),
		Title:       title,
		Description: description,
		Ctx:         ctx,
		Params:      et.Json{},
		Status:      InstanceStatusPending,
	}

	err := s.saveInstance(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
