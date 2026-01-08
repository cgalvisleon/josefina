package server

import (
	"context"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
)

/**
* version
* @param ctx context.Context
* @return et.Json, error
**/
func (s *Router) version(ctx context.Context) (et.Json, error) {
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
* init
* @param ctx context.Context
**/
func (s *Router) init(ctx context.Context) {

}
