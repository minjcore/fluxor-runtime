package fx

import (
	"reflect"
)

// FuncProvider wraps a function as a Provider
type FuncProvider struct {
	fn interface{}
}

// NewProvider creates a new function provider
func NewProvider(fn interface{}) *FuncProvider {
	return &FuncProvider{fn: fn}
}

// Provide calls the function and returns its result
func (p *FuncProvider) Provide() (interface{}, error) {
	fnValue := reflect.ValueOf(p.fn)
	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		return nil, &Error{Message: "provider must be a function"}
	}

	// Call function with no arguments (can be extended to support dependency injection)
	if fnType.NumIn() > 0 {
		return nil, &Error{Message: "provider function must take no arguments"}
	}

	results := fnValue.Call(nil)

	if len(results) == 0 {
		return nil, nil
	}

	if len(results) == 1 {
		return results[0].Interface(), nil
	}

	// Last result is error
	if err, ok := results[len(results)-1].Interface().(error); ok && err != nil {
		return results[0].Interface(), err
	}

	return results[0].Interface(), nil
}

// ValueProvider wraps a value as a Provider
type ValueProvider struct {
	value interface{}
}

// NewValueProvider creates a new value provider
func NewValueProvider(value interface{}) *ValueProvider {
	return &ValueProvider{value: value}
}

// Provide returns the wrapped value
func (p *ValueProvider) Provide() (interface{}, error) {
	return p.value, nil
}
