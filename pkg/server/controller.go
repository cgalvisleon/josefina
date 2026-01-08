package server

import (
	"context"

	"github.com/cgalvisleon/et/et"
)

/**
* version
* @param ctx context.Context
* @return et.Json, error
**/
func version(ctx context.Context) (et.Json, error) {
	service := et.Json{
		"version": Version,
		"service": PackageName,
		"host":    Hostname,
		"company": "",
		"web":     "",
	}

	return service, nil
}
