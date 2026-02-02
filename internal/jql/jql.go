package jql

import (
	"context"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/cache"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

/**
* Jql: Executes a query
* @param ctx context.Context, query et.Json
* @return []et.Json, error
**/
func Jql(ctx context.Context, query et.Json) ([]et.Json, error) {
	app := ctx.Value("app").(string)
	device := ctx.Value("device").(string)
	username := ctx.Value("username").(string)
	key := fmt.Sprintf("%s:%s:%s", app, device, username)
	_, exists := cache.GetStr(key)
	if !exists {
		return nil, msg.ERROR_CLIENT_NOT_AUTHENTICATION.Error()
	}

	return []et.Json{}, nil
}
