package dbs

import (
	"github.com/cgalvisleon/et/et"
	"github.com/dop251/goja"
)

type Vm struct {
	*goja.Runtime
	ctx et.Json
}

/**
* New
* Create a new vm
**/
func newVm() *Vm {
	result := &Vm{
		Runtime: goja.New(),
		ctx:     et.Json{},
	}

	wrapperConsole(result)
	wrapperFetch(result)
	wrapperToJson(result)
	wrapperToString(result)
	return result
}

/**
* Run
* @param script string
* @return goja.Value, error
**/
func (s *Vm) Run(script string) (goja.Value, error) {
	if script == "" {
		return nil, nil
	}

	result, err := s.RunString(script)
	if err != nil {
		return nil, err
	}

	return result, nil
}
