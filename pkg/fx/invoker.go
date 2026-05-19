package fx

import (
	"reflect"
)

// FuncInvoker wraps a function as an Invoker
type FuncInvoker struct {
	fn interface{}
}

// NewInvoker creates a new function invoker
func NewInvoker(fn interface{}) *FuncInvoker {
	return &FuncInvoker{fn: fn}
}

// Invoke calls the function with dependencies from the map
func (i *FuncInvoker) Invoke(deps map[reflect.Type]interface{}) error {
	fnValue := reflect.ValueOf(i.fn)
	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		return &Error{Message: "invoker must be a function"}
	}

	// Build arguments from dependencies
	args := make([]reflect.Value, fnType.NumIn())
	for j := 0; j < fnType.NumIn(); j++ {
		argType := fnType.In(j)

		// Fix Bug 3: Special handling for map[reflect.Type]interface{} parameter
		// Functions like setupApplication expect the entire deps map as a parameter
		if argType.Kind() == reflect.Map {
			keyType := argType.Key()
			elemType := argType.Elem()
			// Check if this is map[reflect.Type]interface{}
			if keyType == reflect.TypeOf((*reflect.Type)(nil)).Elem() &&
				elemType == reflect.TypeOf((*interface{})(nil)).Elem() {
				// This is map[reflect.Type]interface{} - pass the deps map directly
				args[j] = reflect.ValueOf(deps)
				continue
			}
		}

		// Try to find dependency
		if dep, ok := deps[argType]; ok {
			args[j] = reflect.ValueOf(dep)
		} else {
			// Try pointer type (using PointerTo instead of deprecated PtrTo)
			ptrType := reflect.PointerTo(argType)
			if dep, ok := deps[ptrType]; ok {
				args[j] = reflect.ValueOf(dep).Elem()
			} else {
				return &Error{Message: "dependency not found for type: " + argType.String()}
			}
		}
	}

	results := fnValue.Call(args)

	// Check for error return
	if len(results) > 0 {
		if err, ok := results[len(results)-1].Interface().(error); ok {
			return err
		}
	}

	return nil
}
