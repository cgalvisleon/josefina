package v1

import (
	"context"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
)

type Controller struct {
}

/**
* Version
* @param ctx context.Context
* @return et.Json, error
**/
func (c *Controller) Version(ctx context.Context) (et.Json, error) {
	service := et.Json{
		"version": config.App.Version,
		"service": PackageName,
		"host":    "",
		"company": "",
		"web":     "",
	}

	return service, nil
}

/**
* Init
* @param ctx context.Context
**/
func (c *Controller) Init(ctx context.Context) {
	initCore()
}

/**
* Repository
**/
type Repository interface {
	Version(ctx context.Context) (et.Json, error)
	Init(ctx context.Context)
}
