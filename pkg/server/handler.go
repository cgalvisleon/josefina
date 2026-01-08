package server

import (
	"github.com/cgalvisleon/et/et"
)

/**
* version
* @return et.Json, error
**/
func version() (et.Json, error) {
	service := et.Json{
		"version": Version,
		"service": PackageName,
		"host":    Hostname,
		"company": "",
		"web":     "",
	}

	return service, nil
}
