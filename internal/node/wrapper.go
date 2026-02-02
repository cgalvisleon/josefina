package node

import (
	"fmt"

	"github.com/cgalvisleon/josefina/internal/dbs"
	"github.com/cgalvisleon/josefina/pkg/msg"
	"github.com/dop251/goja"
)

/**
* wrapperModel: Wraps the model
* @param vm *Vm
**/
func wrapperModel(vm *dbs.Vm) {
	vm.Set("model", func(call goja.FunctionCall) goja.Value {
		args := call.Arguments
		if len(args) != 3 {
			panic(vm.NewGoError(fmt.Errorf(msg.MSG_ARG_REQUIRED, "database, schema, model")))
		}
		database := args[0].String()
		schema := args[1].String()
		model := args[2].String()
		result, err := node.GetModel(&From{
			Database: database,
			Schema:   schema,
			Name:     model,
		})
		if err != nil {
			panic(vm.NewGoError(err))
		}

		return vm.ToValue(result)
	})
}
