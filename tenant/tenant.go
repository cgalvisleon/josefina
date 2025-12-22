package tenant

import (
	"fmt"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/jdb"
)

const (
	MSG_TENANT_NOT_FOUND = "tenant not found"
)

var (
	ErrTenantNotFound = fmt.Errorf(MSG_TENANT_NOT_FOUND)
)

type Tenant struct {
	DB     *jdb.DB               `json:"db"`
	Models map[string]*jdb.Model `json:"models"`
}

/**
* newTenant
* @param db *jdb.DB
* @return *Tenant
**/
func newTenant(db *jdb.DB) *Tenant {
	return &Tenant{
		DB:     db,
		Models: make(map[string]*jdb.Model),
	}
}

/**
* ToJson
* @return et.Json
**/
func (s *Tenant) ToJson() et.Json {
	result := et.Json{
		"database": s.DB.Name,
		"models":   s.Models,
	}

	return result
}

var tenants map[string]*Tenant

func init() {
	tenants = make(map[string]*Tenant)
}

/**
* save
* @param tenantId string
**/
func save(tenantId string) {
	tenant, ok := tenants[tenantId]
	if !ok {
		return
	}

	scr := tenant.ToJson()
	cache.ObjetSet("tenant", tenantId, scr)
}

/**
* Delete
* @param tenantId string
**/
func Delete(tenantId string) {
	cache.ObjetDelete("tenant", tenantId)
}

/**
* GetDb
* @param tenantId string
* @return (*DB, error)
**/
func GetDb(tenantId string) (*jdb.DB, bool) {
	if _, ok := tenants[tenantId]; !ok {
		return nil, false
	}

	return tenants[tenantId].DB, true
}

/**
* GetModel
* @param tenantId string, name string
* @return (*Model, error)
**/
func GetModel(tenantId string, name string) (*jdb.Model, bool) {
	tenant, ok := tenants[tenantId]
	if !ok {
		return nil, false
	}

	if _, ok := tenant.Models[name]; !ok {
		return nil, false
	}

	return tenant.Models[name], true
}

/**
* LoadDb
* @param tenantId string, dbName string, connection et.Json
* @return (*DB, error)
**/
func LoadDb(tenantId, dbName string, connection et.Json) (*jdb.DB, error) {
	tenant, ok := tenants[tenantId]
	if ok {
		return tenant.DB, nil
	}

	driver := connection.String("driver")
	db, err := jdb.ConnectTo(tenantId, dbName, driver, true, connection)
	if err != nil {
		return nil, err
	}

	tenants[tenantId] = newTenant(db)
	save(tenantId)
	return db, nil
}

/**
* LoadModel
* @param tenantId string, model *Model
* @return (*Model, error)
**/
func LoadModel(tenantId string, model *jdb.Model) (*jdb.Model, error) {
	tenant, ok := tenants[tenantId]
	if !ok {
		return nil, ErrTenantNotFound
	}

	tenant.Models[model.Name] = model

	save(tenantId)
	return model, nil
}
