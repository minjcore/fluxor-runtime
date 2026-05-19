package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// ApplyDefaults applies default values from struct tags to a configuration struct
// Supports "default" tag: `default:"value"`
// Fail-fast: Validates inputs before processing
func ApplyDefaults(target interface{}) error {
	// Fail-fast: target cannot be nil
	failfast.NotNil(target, "target")

	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("fail-fast: target must be a pointer to a struct, got %s", val.Kind())
	}
	if val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("fail-fast: target must be a pointer to a struct, got pointer to %s", val.Elem().Kind())
	}

	return applyDefaultsRecursive(val.Elem())
}

// applyDefaultsRecursive recursively applies defaults to struct fields
func applyDefaultsRecursive(val reflect.Value) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Get default tag
		defaultTag := fieldType.Tag.Get("default")
		if defaultTag == "" {
			// No default tag, check nested structs
			if field.Kind() == reflect.Struct {
				if err := applyDefaultsRecursive(field); err != nil {
					return err
				}
			} else if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
				if field.IsNil() {
					field.Set(reflect.New(field.Type().Elem()))
				}
				if err := applyDefaultsRecursive(field.Elem()); err != nil {
					return err
				}
			}
			continue
		}

		// Only apply default if field is zero value
		if !isZeroValue(field) {
			// Field already has a value, check nested structs
			if field.Kind() == reflect.Struct {
				if err := applyDefaultsRecursive(field); err != nil {
					return err
				}
			} else if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
				if field.IsNil() {
					field.Set(reflect.New(field.Type().Elem()))
				}
				if err := applyDefaultsRecursive(field.Elem()); err != nil {
					return err
				}
			}
			continue
		}

		// Apply default value
		if err := setFieldFromDefault(field, defaultTag); err != nil {
			return fmt.Errorf("failed to set default for field %s: %w", fieldType.Name, err)
		}

		// If it's a struct, recursively apply defaults
		if field.Kind() == reflect.Struct {
			if err := applyDefaultsRecursive(field); err != nil {
				return err
			}
		} else if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
			if err := applyDefaultsRecursive(field.Elem()); err != nil {
				return err
			}
		}
	}

	return nil
}

// isZeroValue checks if a reflect.Value is zero value
func isZeroValue(val reflect.Value) bool {
	switch val.Kind() {
	case reflect.String:
		return val.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return val.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return val.Float() == 0
	case reflect.Bool:
		return !val.Bool()
	case reflect.Slice, reflect.Map, reflect.Array:
		return val.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return val.IsNil()
	default:
		return false
	}
}

// setFieldFromDefault sets a struct field value from default tag string
// Fail-fast: Validates inputs and fails immediately on invalid defaults
func setFieldFromDefault(field reflect.Value, defaultValue string) error {
	// Fail-fast: defaultValue cannot be empty for non-optional fields
	if defaultValue == "" && field.Kind() != reflect.String {
		return fmt.Errorf("fail-fast: default value cannot be empty for non-string field")
	}

	// Fail-fast: field must be settable
	if !field.CanSet() {
		return fmt.Errorf("fail-fast: field is not settable")
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(defaultValue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Check if it's a duration
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			duration, err := time.ParseDuration(defaultValue)
			if err != nil {
				return fmt.Errorf("fail-fast: invalid duration default %q: %w", defaultValue, err)
			}
			// Fail-fast: duration must be non-negative
			if duration < 0 {
				return fmt.Errorf("fail-fast: duration default cannot be negative: %v", duration)
			}
			field.SetInt(int64(duration))
		} else {
			intVal, err := strconv.ParseInt(defaultValue, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid integer default: %s", defaultValue)
			}
			field.SetInt(intVal)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(defaultValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer default: %s", defaultValue)
		}
		field.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(defaultValue, 64)
		if err != nil {
			return fmt.Errorf("invalid float default: %s", defaultValue)
		}
		field.SetFloat(floatVal)
	case reflect.Bool:
		boolVal := strings.ToLower(defaultValue) == "true" || defaultValue == "1"
		field.SetBool(boolVal)
	default:
		return fmt.Errorf("unsupported field type for default: %s", field.Kind())
	}

	return nil
}
