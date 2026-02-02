package jql

import (
	"context"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/cache"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

/**
* JQuery: Executes a query
* @param ctx context.Context, query et.Json
* @return et.Items, error
**/
func Jquery(ctx context.Context, query et.Json) (et.Items, error) {
	app := ctx.Value("app").(string)
	device := ctx.Value("device").(string)
	username := ctx.Value("username").(string)
	key := fmt.Sprintf("%s:%s:%s", app, device, username)
	_, exists := cache.GetStr(key)
	if !exists {
		return et.Items{}, msg.ERROR_CLIENT_NOT_AUTHENTICATION.Error()
	}

	ql, err := getQl(query)
	if err != nil {
		return et.Items{}, err
	}

	result, err := ql.run()
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}
