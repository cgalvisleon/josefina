package jdb

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/msg"
	"github.com/dop251/goja"
)

type Vm struct {
	*goja.Runtime
	ctx et.Json
}

/**
* NewVm
* Create a new vm
**/
func NewVm() *Vm {
	result := &Vm{
		Runtime: goja.New(),
		ctx:     et.Json{},
	}

	wrapperConsole(result)
	wrapperFetch(result)
	wrapperToJson(result)
	wrapperToString(result)
	wrapperModel(result)
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
* wrapperConsole: Wraps the console
* @param vm *Vm
**/
func wrapperConsole(vm *Vm) {
	vm.Set("console", map[string]interface{}{
		"log": func(args ...interface{}) {
			kind := "Log"
			logs.Log(kind, args...)
		},
		"debug": func(args ...interface{}) {
			logs.Debug(args...)
		},
		"info": func(args ...interface{}) {
			logs.Info(args...)
		},
		"error": func(args string) {
			logs.Error(errors.New(args))
		},
	})
}

/**
* wrapperFetch: Wraps the fetch
* @param vm *Vm
**/
func wrapperFetch(vm *Vm) {
	vm.Set("fetch", func(call goja.FunctionCall) goja.Value {
		args := call.Arguments
		if len(args) != 4 {
			panic(vm.NewGoError(fmt.Errorf(msg.MSG_ARG_REQUIRED, "method, url, headers, body")))
		}
		method := args[0].String()
		url := args[1].String()
		headers := args[2].Export().(map[string]interface{})
		body := args[3].Export().(map[string]interface{})
		result, status := request.Fetch(method, url, headers, body)
		if status.Code != 200 {
			panic(vm.NewGoError(errors.New(status.Message)))
		}
		if !status.Ok {
			panic(vm.NewGoError(fmt.Errorf("error al hacer la peticion: %s", status.Message)))
		}

		return vm.ToValue(result)
	})
}

/**
* wrapperToJson: Wraps the json
* @param vm *Vm
**/
func wrapperToJson(vm *Vm) {
	vm.Set("toJson", func(call goja.FunctionCall) goja.Value {
		args := call.Arguments
		if len(args) != 1 {
			panic(vm.NewGoError(fmt.Errorf(msg.MSG_ARG_REQUIRED, "value")))
		}
		value := args[0].Export()

		switch v := value.(type) {
		case map[string]interface{}:
			return vm.ToValue(et.Json(v).ToString())
		case et.Json:
			return vm.ToValue(v.ToString())
		default:
			return vm.ToValue(et.Json{})
		}
	})
}

/**
* wrapperToString: Wraps the to string
* @param vm *Vm
**/
func wrapperToString(vm *Vm) {
	vm.Set("toString", func(call goja.FunctionCall) goja.Value {
		args := call.Arguments
		if len(args) != 1 {
			panic(vm.NewGoError(fmt.Errorf(msg.MSG_ARG_REQUIRED, "value")))
		}
		value := args[0].Export()

		switch v := value.(type) {
		case map[string]interface{}:
			return vm.ToValue(et.Json(v).ToString())
		case et.Json:
			return vm.ToValue(v.ToString())
		default:
			return vm.ToValue(et.Json{})
		}
	})
}

/**
* wrapperModel: Wraps the model
* @param vm *Vm
**/
func wrapperModel(vm *Vm) {
	vm.Set("getModel", func(call goja.FunctionCall) goja.Value {
		args := call.Arguments
		if len(args) != 3 {
			panic(vm.NewGoError(fmt.Errorf(msg.MSG_ARG_REQUIRED, "database, schema, model")))
		}
		database := args[0].String()
		schema := args[1].String()
		name := args[2].String()
		result, exists := node.GetModel(&catalog.From{
			Database: database,
			Schema:   schema,
			Name:     name,
		})
		if !exists {
			panic(vm.NewGoError(errors.New(msg.MSG_MODEL_NOT_FOUND)))
		}

		return vm.ToValue(result)
	})
}
