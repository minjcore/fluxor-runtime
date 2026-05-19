package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// RequiredFields validates that required fields are not empty
// Fail-fast: Validates inputs before processing
func RequiredFields(fields ...string) Validator {
	// Fail-fast: must have at least one field
	if len(fields) == 0 {
		panic(fmt.Errorf("fail-fast: RequiredFields must have at least one field"))
	}

	// Fail-fast: no field can be empty
	for i, field := range fields {
		if field == "" {
			panic(fmt.Errorf("fail-fast: RequiredFields field at index %d cannot be empty", i))
		}
	}

	return ValidatorFunc(func(config interface{}) error {
		// Fail-fast: config cannot be nil
		failfast.NotNil(config, "config")

		val := reflect.ValueOf(config)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		if val.Kind() != reflect.Struct {
			return fmt.Errorf("fail-fast: config must be a struct, got %s", val.Kind())
		}

		missing := make([]string, 0)

		for _, fieldName := range fields {
			// Support nested field paths
			fieldVal := getNestedField(val, fieldName)
			if !fieldVal.IsValid() {
				return fmt.Errorf("field %s not found in config struct", fieldName)
			}

			if isEmpty(fieldVal) {
				missing = append(missing, fieldName)
			}
		}

		if len(missing) > 0 {
			return fmt.Errorf("required fields are missing: %s", strings.Join(missing, ", "))
		}

		return nil
	})
}

// isEmpty checks if a reflect.Value is empty (zero value)
func isEmpty(val reflect.Value) bool {
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

// RangeValidator validates that a numeric field is within a range
// Supports nested fields using dot notation (e.g., "Database.MaxConns")
// Fail-fast: Validates inputs before processing
func RangeValidator(fieldName string, min, max float64) Validator {
	// Fail-fast: fieldName cannot be empty
	if fieldName == "" {
		panic(fmt.Errorf("fail-fast: RangeValidator fieldName cannot be empty"))
	}

	// Fail-fast: min must be <= max
	if min > max {
		panic(fmt.Errorf("fail-fast: RangeValidator min (%f) must be <= max (%f)", min, max))
	}

	return ValidatorFunc(func(config interface{}) error {
		// Fail-fast: config cannot be nil
		failfast.NotNil(config, "config")

		val := reflect.ValueOf(config)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		// Support nested field paths (e.g., "Database.MaxConns")
		fieldVal := getNestedField(val, fieldName)
		if !fieldVal.IsValid() {
			return fmt.Errorf("field %s not found", fieldName)
		}

		var numVal float64
		switch fieldVal.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			numVal = float64(fieldVal.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			numVal = float64(fieldVal.Uint())
		case reflect.Float32, reflect.Float64:
			numVal = fieldVal.Float()
		default:
			return fmt.Errorf("field %s is not numeric", fieldName)
		}

		if numVal < min || numVal > max {
			return fmt.Errorf("field %s value %f is out of range [%f, %f]", fieldName, numVal, min, max)
		}

		return nil
	})
}

// getNestedField gets a field value, supporting nested paths with dot notation
func getNestedField(val reflect.Value, fieldPath string) reflect.Value {
	parts := strings.Split(fieldPath, ".")
	current := val

	for _, part := range parts {
		if current.Kind() == reflect.Ptr {
			current = current.Elem()
		}
		if current.Kind() != reflect.Struct {
			return reflect.Value{}
		}
		current = current.FieldByName(part)
		if !current.IsValid() {
			return reflect.Value{}
		}
	}
	return current
}

// StringLengthValidator validates that a string field has a specific length range
// Supports nested fields using dot notation
// Fail-fast: Validates inputs before processing
func StringLengthValidator(fieldName string, minLen, maxLen int) Validator {
	// Fail-fast: fieldName cannot be empty
	if fieldName == "" {
		panic(fmt.Errorf("fail-fast: StringLengthValidator fieldName cannot be empty"))
	}

	// Fail-fast: minLen must be >= 0
	if minLen < 0 {
		panic(fmt.Errorf("fail-fast: StringLengthValidator minLen (%d) must be >= 0", minLen))
	}

	// Fail-fast: maxLen must be >= minLen
	if maxLen < minLen {
		panic(fmt.Errorf("fail-fast: StringLengthValidator maxLen (%d) must be >= minLen (%d)", maxLen, minLen))
	}

	return ValidatorFunc(func(config interface{}) error {
		// Fail-fast: config cannot be nil
		failfast.NotNil(config, "config")

		val := reflect.ValueOf(config)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		fieldVal := getNestedField(val, fieldName)
		if !fieldVal.IsValid() {
			return fmt.Errorf("field %s not found", fieldName)
		}

		if fieldVal.Kind() != reflect.String {
			return fmt.Errorf("field %s is not a string", fieldName)
		}

		strVal := fieldVal.String()
		length := len(strVal)

		if length < minLen || length > maxLen {
			return fmt.Errorf("field %s length %d is out of range [%d, %d]", fieldName, length, minLen, maxLen)
		}

		return nil
	})
}

// OneOfValidator validates that a field value is one of the allowed values
// Supports nested fields using dot notation
// Fail-fast: Validates inputs before processing
func OneOfValidator(fieldName string, allowedValues ...interface{}) Validator {
	// Fail-fast: fieldName cannot be empty
	if fieldName == "" {
		panic(fmt.Errorf("fail-fast: OneOfValidator fieldName cannot be empty"))
	}

	// Fail-fast: must have at least one allowed value
	if len(allowedValues) == 0 {
		panic(fmt.Errorf("fail-fast: OneOfValidator must have at least one allowed value"))
	}

	return ValidatorFunc(func(config interface{}) error {
		// Fail-fast: config cannot be nil
		failfast.NotNil(config, "config")

		val := reflect.ValueOf(config)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		// Support nested field paths
		fieldVal := getNestedField(val, fieldName)
		if !fieldVal.IsValid() {
			return fmt.Errorf("field %s not found", fieldName)
		}

		fieldInterface := fieldVal.Interface()

		for _, allowed := range allowedValues {
			if reflect.DeepEqual(fieldInterface, allowed) {
				return nil
			}
		}

		return fmt.Errorf("field %s value %v is not one of allowed values: %v", fieldName, fieldInterface, allowedValues)
	})
}

// ValidateFromTags validates configuration using struct tags
// Supports validate:"required,range:1-100,min:10,max:1000" format
// Fail-fast: Validates inputs before processing
func ValidateFromTags(target interface{}) error {
	// Fail-fast: target cannot be nil
	failfast.NotNil(target, "target")

	val := reflect.ValueOf(target)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("fail-fast: target must be a struct, got %s", val.Kind())
	}

	return validateFromTagsRecursive("", val)
}

// validateFromTagsRecursive recursively validates struct fields using tags
func validateFromTagsRecursive(prefix string, val reflect.Value) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		// Get validate tag
		validateTag := fieldType.Tag.Get("validate")
		if validateTag == "" {
			// No validation tag, check nested structs
			if field.Kind() == reflect.Struct {
				fieldPrefix := fieldType.Name
				if prefix != "" {
					fieldPrefix = prefix + "." + fieldType.Name
				}
				if err := validateFromTagsRecursive(fieldPrefix, field); err != nil {
					return err
				}
			}
			continue
		}

		// Build full field path
		fieldPath := fieldType.Name
		if prefix != "" {
			fieldPath = prefix + "." + fieldType.Name
		}

		// Parse validation rules
		rules := strings.Split(validateTag, ",")
		for _, rule := range rules {
			rule = strings.TrimSpace(rule)
			if err := applyValidationRule(fieldPath, field, rule); err != nil {
				return err
			}
		}

		// Validate nested structs
		if field.Kind() == reflect.Struct {
			fieldPrefix := fieldPath
			if err := validateFromTagsRecursive(fieldPrefix, field); err != nil {
				return err
			}
		}
	}

	return nil
}

// applyValidationRule applies a single validation rule to a field
func applyValidationRule(fieldPath string, field reflect.Value, rule string) error {
	if rule == "required" {
		if isEmpty(field) {
			return fmt.Errorf("field %s is required", fieldPath)
		}
		return nil
	}

	// Parse range:min-max
	if strings.HasPrefix(rule, "range:") {
		rangeStr := strings.TrimPrefix(rule, "range:")
		parts := strings.Split(rangeStr, "-")
		if len(parts) != 2 {
			return fmt.Errorf("invalid range format for field %s: %s", fieldPath, rule)
		}

		var min, max float64
		var err error
		if min, err = parseFloat(parts[0]); err != nil {
			return fmt.Errorf("invalid min value for field %s: %s", fieldPath, parts[0])
		}
		if max, err = parseFloat(parts[1]); err != nil {
			return fmt.Errorf("invalid max value for field %s: %s", fieldPath, parts[1])
		}

		validator := RangeValidator(fieldPath, min, max)
		return validator.Validate(field.Interface())
	}

	// Parse min:value
	if strings.HasPrefix(rule, "min:") {
		minStr := strings.TrimPrefix(rule, "min:")
		min, err := parseFloat(minStr)
		if err != nil {
			return fmt.Errorf("invalid min value for field %s: %s", fieldPath, minStr)
		}
		validator := RangeValidator(fieldPath, min, 1e10) // Large max
		return validator.Validate(field.Interface())
	}

	// Parse max:value
	if strings.HasPrefix(rule, "max:") {
		maxStr := strings.TrimPrefix(rule, "max:")
		max, err := parseFloat(maxStr)
		if err != nil {
			return fmt.Errorf("invalid max value for field %s: %s", fieldPath, maxStr)
		}
		validator := RangeValidator(fieldPath, -1e10, max) // Small min
		return validator.Validate(field.Interface())
	}

	// Parse oneof:value1,value2,value3
	if strings.HasPrefix(rule, "oneof:") {
		oneofStr := strings.TrimPrefix(rule, "oneof:")
		values := strings.Split(oneofStr, ",")
		allowedValues := make([]interface{}, len(values))
		for i, v := range values {
			allowedValues[i] = strings.TrimSpace(v)
		}
		validator := OneOfValidator(fieldPath, allowedValues...)
		return validator.Validate(field.Interface())
	}

	return nil
}

// parseFloat parses a float from string, handling duration strings
func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	// Try parsing as duration first
	if duration, err := time.ParseDuration(s); err == nil {
		return float64(duration), nil
	}
	// Try parsing as regular float
	return strconv.ParseFloat(s, 64)
}
