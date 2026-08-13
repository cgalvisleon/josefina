package jdb

import (
	"errors"
	"sync"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/josefina/internal/msg"
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
	Error       string         `json:"error"`
	Result      et.Json        `json:"result"`
	Status      InstanceStatus `json:"status"`
	AuditLog    []et.Json      `json:"audit_log"`
	isChanged   bool           `json:"-"`
	owner       *Instances     `json:"-"`
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
		"error":       s.Error,
		"result":      s.Result,
		"status":      s.Status,
		"audit_log":   s.AuditLog,
	}
}

/**
* addAuditLog
* @param sessionName string, action string
**/
func (s *Instance) addAuditLog(sessionName string, action string) {
	if s.AuditLog == nil {
		s.AuditLog = make([]et.Json, 0)
	}

	now := timezone.Now()
	s.UpdatedAt = now
	s.AuditLog = append(s.AuditLog, et.Json{
		"created_at":   now,
		"session_name": sessionName,
		"action":       action,
	})
	maxAuditLog := envar.GetInt("MAX_AUDIT_LOG", 1000)
	if len(s.AuditLog) > maxAuditLog {
		s.AuditLog = s.AuditLog[len(s.AuditLog)-maxAuditLog:]
	}
	s.isChanged = true
}

/**
* setStatus
* @param status InstanceStatus
**/
func (s *Instance) setStatus(sessionName string, status InstanceStatus) {
	s.Status = status
	s.addAuditLog(sessionName, "set status to "+string(status))
}

/**
* setParams
* @param params et.Json
**/
func (s *Instance) setParams(sessionName string, params et.Json) {
	for key, value := range params {
		s.Params[key] = value
	}
	s.addAuditLog(sessionName, "set params to "+params.String())
}

/**
* setError
* @param error string
**/
func (s *Instance) setError(sessionName string, err error) {
	s.Error = err.Error()
	s.addAuditLog(sessionName, "set error to "+s.Error)
}

/**
* setResult
* @param result et.Json
**/
func (s *Instance) setResult(sessionName string, result et.Json) {
	s.Result = result
	s.addAuditLog(sessionName, "set result to "+result.String())
}

/**
* save: Saves the instance to the database
* @return error
**/
func (s *Instance) save() error {
	if !s.isChanged {
		return nil
	}

	if s.owner == nil {
		return errors.New(msg.MSG_INSTANCE_DONT_HAVE_OWNER)
	}

	return s.owner.saveInstance(s)
}

type Instances struct {
	instances map[string]*Instance
	mu        sync.RWMutex
	store     *Model
	db        *DB
}

/**
* newInstance: Creates a new instance
* @param title, description string, ctx et.Json
* @return (*Instance, error)
**/
func (s *Instances) newInstance(id, title, description string, ctx et.Json) (*Instance, error) {
	now := timezone.Now()
	code, err := s.db.incSerie("instance", "code")
	if err != nil {
		return nil, err
	}

	if id == "" {
		id = reg.UUID()
	}

	result := &Instance{
		CreatedAt:   now,
		UpdatedAt:   now,
		DoneAt:      time.Time{},
		ID:          id,
		Code:        code,
		Title:       title,
		Description: description,
		Ctx:         ctx,
		Params:      et.Json{},
		Result:      et.Json{},
		Status:      InstanceStatusPending,
		AuditLog:    []et.Json{},
		owner:       s,
	}

	err = s.saveInstance(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* saveInstance: Saves an instance to the database
* @param instance *Instance
* @return error
**/
func (s *Instances) saveInstance(instance *Instance) error {
	s.addInstance(instance)
	return s.store.put(instance.ID, instance)
}

/**
* addInstance: Adds an instance to the cache
* @param instance *Instance
**/
func (s *Instances) addInstance(instance *Instance) {
	s.mu.Lock()
	s.instances[instance.ID] = instance
	s.mu.Unlock()
}

/**
* getInstance: Gets an instance from the cache
* @param id string
* @return (*Instance, error)
**/
func (s *Instances) getInstance(id string) (*Instance, error) {
	s.mu.RLock()
	result, exists := s.instances[id]
	s.mu.RUnlock()
	if exists {
		return result, nil
	}

	exists, err := s.store.get(id, &result)
	if err != nil {
		return nil, err
	}

	result.owner = s

	if !exists {
		return nil, errors.New(msg.MSG_INSTANCE_NOT_FOUND)
	}

	return result, nil
}

/**
* loadInstances: Loads the instances
* @return error
**/
func (s *DB) loadInstances() error {
	store, err := s.newModel(sysSchema, "instances", 1, true)
	if err != nil {
		return err
	}

	if err := store.Init(); err != nil {
		return err
	}

	result := &Instances{
		instances: make(map[string]*Instance),
		store:     store,
		db:        s,
	}

	s.instances = result

	return nil
}

/**
* newInstance: Creates a new instance
* @param id, title, description string, ctx et.Json
* @return (*Instance, error)
**/
func (s *DB) newInstance(id, title, description string, ctx et.Json) (*Instance, error) {
	return s.instances.newInstance(id, title, description, ctx)
}

/**
* getInstance: Gets an instance from the database
* @param id string
* @return (*Instance, error)
**/
func (s *DB) getInstance(id string) (*Instance, error) {
	return s.instances.getInstance(id)
}
