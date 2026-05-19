package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// LoadProperties loads configuration from a properties file into a struct
// Supports dot-notation keys (e.g., "server.addr" maps to Server.Addr field)
// Supports duration strings, integers, booleans, and strings
// Fail-fast: Validates inputs before processing
func LoadProperties(path string, target interface{}) error {
	// Fail-fast: path cannot be empty
	if path == "" {
		return fmt.Errorf("fail-fast: properties file path cannot be empty")
	}

	// Fail-fast: target cannot be nil
	failfast.NotNil(target, "target")

	// Fail-fast: target must be a pointer to a struct
	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("fail-fast: target must be a pointer to a struct, got %s", val.Kind())
	}
	if val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("fail-fast: target must be a pointer to a struct, got pointer to %s", val.Elem().Kind())
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("fail-fast: properties file not found: %s", path)
	}

	// #nosec G304 -- path is provided by the caller (library function); callers should validate/lock down inputs if untrusted.
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read properties file %s: %w", path, err)
	}

	// Parse properties into map
	props := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle key=value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			props[key] = value
		}
	}

	// Apply properties to struct
	return applyPropertiesToStruct(props, target)
}

// getPropertyCaseInsensitive performs a case-insensitive lookup in the properties map
// First tries exact match, then tries case-insensitive match
func getPropertyCaseInsensitive(props map[string]string, key string) (string, bool) {
	// Try exact match first (fast path)
	if value, ok := props[key]; ok {
		return value, true
	}

	// Try case-insensitive match
	keyLower := strings.ToLower(key)
	for k, v := range props {
		if strings.ToLower(k) == keyLower {
			return v, true
		}
	}

	return "", false
}

// applyPropertiesToStruct applies properties map to struct using reflection
// Fail-fast: Validates inputs before processing
func applyPropertiesToStruct(props map[string]string, target interface{}) error {
	// Fail-fast: props cannot be nil
	if props == nil {
		return fmt.Errorf("fail-fast: properties map cannot be nil")
	}

	// Fail-fast: target cannot be nil
	failfast.NotNil(target, "target")

	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("fail-fast: target must be a pointer to a struct")
	}

	return applyPropertiesRecursive(props, "", val.Elem())
}

// applyPropertiesRecursive recursively applies properties to struct fields
func applyPropertiesRecursive(props map[string]string, prefix string, val reflect.Value) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Get config tag for section name, or use field name
		configTag := fieldType.Tag.Get("config")
		fieldName := fieldType.Name
		if configTag != "" {
			fieldName = configTag
		}

		// Build property key: prefix.fieldName (e.g., "server.addr")
		var propKey string
		if prefix == "" {
			propKey = strings.ToLower(fieldName)
		} else {
			propKey = prefix + "." + strings.ToLower(fieldName)
		}

		// Handle nested structs
		if field.Kind() == reflect.Struct {
			// Check if this is a time.Duration (special case)
			if field.Type() == reflect.TypeOf(time.Duration(0)) {
				if value, ok := getPropertyCaseInsensitive(props, propKey); ok {
					if err := setDurationField(field, value); err != nil {
						return fmt.Errorf("failed to set duration field %s: %w", propKey, err)
					}
				}
			} else {
				// Recursively apply to nested struct
				if err := applyPropertiesRecursive(props, propKey, field); err != nil {
					return err
				}
			}
			continue
		}

		// Handle pointers to structs
		if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
			if value, ok := getPropertyCaseInsensitive(props, propKey); ok && value != "" {
				if field.IsNil() {
					field.Set(reflect.New(field.Type().Elem()))
				}
				if err := applyPropertiesRecursive(props, propKey, field.Elem()); err != nil {
					return err
				}
			}
			continue
		}

		// Get property value (case-insensitive lookup)
		value, ok := getPropertyCaseInsensitive(props, propKey)
		if !ok || value == "" {
			continue // Skip if not found or empty
		}

		// Set field value based on type
		if err := setFieldFromString(field, value); err != nil {
			return fmt.Errorf("failed to set field %s from property %s: %w", fieldType.Name, propKey, err)
		}
	}

	return nil
}

// setFieldFromString sets a struct field value from string
// Fail-fast: Validates inputs before processing
func setFieldFromString(field reflect.Value, value string) error {
	// Fail-fast: value cannot be empty for non-optional fields
	if value == "" && field.Kind() != reflect.String {
		return fmt.Errorf("fail-fast: cannot set empty value to non-string field")
	}

	// Fail-fast: field must be settable
	if !field.CanSet() {
		return fmt.Errorf("fail-fast: field is not settable")
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Check if it's a duration string
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			return setDurationField(field, value)
		}
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer value: %s", value)
		}
		field.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer value: %s", value)
		}
		field.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float value: %s", value)
		}
		field.SetFloat(floatVal)
	case reflect.Bool:
		boolVal := strings.ToLower(value) == "true" || value == "1"
		field.SetBool(boolVal)
	case reflect.Slice:
		// For slices, split by comma
		parts := strings.Split(value, ",")
		sliceType := field.Type().Elem()
		slice := reflect.MakeSlice(field.Type(), len(parts), len(parts))
		for i, part := range parts {
			part = strings.TrimSpace(part)
			elem := reflect.New(sliceType).Elem()
			if err := setFieldFromString(elem, part); err != nil {
				return err
			}
			slice.Index(i).Set(elem)
		}
		field.Set(slice)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}

// setDurationField sets a time.Duration field from string
// Fail-fast: Validates inputs and fails immediately on invalid duration
func setDurationField(field reflect.Value, value string) error {
	// Fail-fast: value cannot be empty
	if value == "" {
		return fmt.Errorf("fail-fast: duration value cannot be empty")
	}

	// Fail-fast: field must be settable
	if !field.CanSet() {
		return fmt.Errorf("fail-fast: duration field is not settable")
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("fail-fast: invalid duration value %q: %w", value, err)
	}

	// Fail-fast: duration must be non-negative
	if duration < 0 {
		return fmt.Errorf("fail-fast: duration cannot be negative: %v", duration)
	}

	field.SetInt(int64(duration))
	return nil
}
