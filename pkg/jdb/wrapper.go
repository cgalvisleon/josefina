package jdb

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/josefina/internal/core"
	"github.com/cgalvisleon/josefina/internal/mod"
	"github.com/cgalvisleon/josefina/internal/msg"
	"github.com/dop251/goja"
)

/**
* wrapperModel: Wraps the model
* @param vm *Vm
**/
func wrapperModel(vm *mod.Vm) {
	vm.Set("model", func(call goja.FunctionCall) goja.Value {
		args := call.Arguments
		if len(args) != 3 {
			panic(vm.NewGoError(fmt.Errorf(msg.MSG_ARG_REQUIRED, "database, schema, model")))
		}
		database := args[0].String()
		schema := args[1].String()
		name := args[2].String()
		var result *mod.Model
		exists, err := core.GetModel(&mod.From{
			Database: database,
			Schema:   schema,
			Name:     name,
		}, result)
		if err != nil {
			panic(vm.NewGoError(err))
		}

		if !exists {
			panic(vm.NewGoError(errors.New(msg.MSG_MODEL_NOT_EXISTS)))
		}

		return vm.ToValue(result)
	})
}
