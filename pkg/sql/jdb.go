package sql

import (
	"context"
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/core"
	"github.com/cgalvisleon/josefina/internal/msg"
)

var (
	appName string = "josefina"
	version string = "0.0.1"
	node    *core.Node
)

/**
* HelpCheck: Returns the help check
* @return et.Item
**/
func HelpCheck() et.Item {
	if !node.started {
		return et.Item{
			Ok: false,
			Result: et.Json{
				"status":  false,
				"message": "josefina is not started",
			},
		}
	}

	return et.Item{
		Ok:     true,
		Result: node.toJson(),
	}
}

/**
* authenticate: Authenticates a user
* @param ctx context.Context
* @return error
**/
func authenticate(ctx context.Context) error {
	app := ctx.Value("app").(string)
	device := ctx.Value("device").(string)
	username := ctx.Value("username").(string)
	key := fmt.Sprintf("%s:%s:%s", app, device, username)
	_, exists, err := cache.GetStr(key)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(msg.MSG_CLIENT_NOT_AUTHENTICATION)
	}

	return nil
}

/**
* Query: Executes a query
* @param ctx context.Context, query string
* @return et.Items, error
**/
func Query(ctx context.Context, sql string, args ...any) (et.Items, error) {
	err := authenticate(ctx)
	if err != nil {
		return et.Items{}, err
	}

	items, err := query(sql, args...)
	if err != nil {
		return et.Items{}, err
	}

	result := et.Items{}
	result.Add(items...)
	return result, nil
}

/**
* JQuery: Executes a query
* @param ctx context.Context, query et.Json
* @return et.Items, error
**/
func JQuery(ctx context.Context, query et.Json) (et.Items, error) {
	err := authenticate(ctx)
	if err != nil {
		return et.Items{}, err
	}

	items, err := jquery(query)
	if err != nil {
		return et.Items{}, err
	}

	result := et.Items{}
	result.Add(items...)
	return result, nil
}
