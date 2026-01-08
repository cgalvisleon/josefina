package http

import (
	"github.com/cgalvisleon/et/server"
	v1 "github.com/cgalvisleon/josefina/internal/services/v1"
	pkg "github.com/cgalvisleon/josefina/pkg/server"
)

func New() (*server.Ettp, error) {
	result, err := server.New(pkg.PackageName)
	if err != nil {
		return nil, err
	}

	latest := v1.New()
	result.Mount("/", latest)
	result.Mount("/v1", latest)
	result.OnClose(v1.Close)

	return result, nil
}
