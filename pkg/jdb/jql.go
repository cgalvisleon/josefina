package jdb

import (
	"context"
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/cache"
	"github.com/cgalvisleon/josefina/internal/jql"
	"github.com/cgalvisleon/josefina/internal/msg"
)

/**
* JQuery: Executes a query
* @param ctx context.Context, query et.Json
* @return et.Items, error
**/
func JQuery(ctx context.Context, query et.Json) (et.Items, error) {
	app := ctx.Value("app").(string)
	device := ctx.Value("device").(string)
	username := ctx.Value("username").(string)
	key := fmt.Sprintf("%s:%s:%s", app, device, username)
	_, exists := cache.GetStr(key)
	if !exists {
		return et.Items{}, errors.New(msg.MSG_CLIENT_NOT_AUTHENTICATION)
	}

	ql, err := jql.ToQl(query)
	if err != nil {
		return et.Items{}, err
	}

	result, err := ql.Run()
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}
