package vm

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

/**
* runTrigger
* @param trigger *Trigger, tx *Tx, old et.Json, new et.Json
* @return error
**/
// func (s *Cmd) runTrigger(trigger *Trigger, tx *Tx, old, new et.Json) error {
// 	model := s.model
// 	vm, ok := model.triggers[trigger.Name]
// 	if !ok {
// 		vm = newVm()
// 		model.triggers[trigger.Name] = vm
// 	}

// 	vm.Set("self", model)
// 	vm.Set("tx", tx)
// 	vm.Set("old", old)
// 	vm.Set("new", new)
// 	script := string(trigger.Definition)
// 	_, err := vm.Run(script)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }
