package server

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/server"
	v1 "github.com/josefina/internal/server/v1"
)

func New() (*server.Ettp, error) {
	port := envar.GetInt("PORT", 1377)
	result := server.New(v1.AppName, port)

	latest := v1.New()
	result.Mount("/", latest)
	result.Mount("/v1", latest)
	result.OnClose(v1.Close)

	return result, nil
}
