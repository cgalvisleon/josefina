package jdb

import (
	"fmt"

	"github.com/cgalvisleon/josefina/internal/jql"
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
		model := args[2].String()
		result, err := jql.GetModel(database, schema, model)
		if err != nil {
			panic(vm.NewGoError(err))
		}

		return vm.ToValue(result)
	})
}
