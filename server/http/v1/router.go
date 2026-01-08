package v1

import (
	"context"
	"net/http"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/middleware"
	"github.com/go-chi/chi/v5"
)

var PackageName = "apps"
var PathApi = envar.GetStr("PATH_API", "/api")

type Router struct {
	Repository Repository
}

func (s *Router) Routes() http.Handler {
	// defaultHost := strs.Format("http://%s", envar.GetStr("HOST", "localhost"))
	// var host = strs.Format("%s:%d", envar.GetStr("HOST", defaultHost), envar.GetInt("PORT", 3300))

	r := chi.NewRouter()

	ctx := context.Background()
	s.Repository.Init(ctx)
	middleware.SetServiceName(PackageName)

	logs.Logf(PackageName, "Router version:%s", config.App.Version)

	return r
}
