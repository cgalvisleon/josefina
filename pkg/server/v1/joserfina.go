package v1

import "github.com/cgalvisleon/josefina/pkg/josefina"

/**
* InitJosefina
* @return error
**/
func InitJosefina() error {
	josefina.Init()

	return nil
}
