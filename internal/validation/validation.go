package validation

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// ValidateClientInput validates that a struct has all required fields filled
func ValidateClientInput(data interface{}) (bool, error) {
	if data == nil {
		return false, fmt.Errorf("data cannot be nil")
	}

	val := reflect.ValueOf(data)

	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return false, fmt.Errorf("data pointer cannot be nil")
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return false, fmt.Errorf("data must be a struct or pointer to struct")
	}

	valType := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := valType.Field(i)
		fieldValue := val.Field(i)

		if field.Name == "ID" {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Validate email without short-circuiting the rest of the fields
		if field.Name == "Email" || field.Name == "email" {
			if _, err := checkEmail(fieldValue.String()); err != nil {
				return false, err
			}
			continue
		}

		if !fieldValue.CanInterface() {
			return false, fmt.Errorf("field '%s' is not supported or not part of this form or request", jsonTag)
		}

		// Fixed: isFieldEmpty returns true when empty, so no negation needed
		if isFieldEmpty(fieldValue) {
			return false, fmt.Errorf("field '%s' is required but empty", jsonTag)
		}
	}

	return true, nil
}

// ValidateJSONFields validates that JSON data only contains allowed fields
func ValidateJSONFields(jsonData []byte, model interface{}) (bool, error) {
	if len(jsonData) == 0 {
		return false, fmt.Errorf("JSON data cannot be empty")
	}

	allowedFields := GetAllowedFields(model)

	var inputMap map[string]any
	if err := json.Unmarshal(jsonData, &inputMap); err != nil {
		return false, fmt.Errorf("invalid JSON format: %v", err)
	}

	for fieldName := range inputMap {
		if fieldName == "id" || fieldName == "ID" {
			continue
		}
		if !allowedFields[fieldName] {
			return false, fmt.Errorf("field '%s' is not supported or not part of this form or request", fieldName)
		}
	}

	return true, nil
}

// GetAllowedFields extracts all JSON field names from a struct
func GetAllowedFields(model interface{}) map[string]bool {
	allowed := make(map[string]bool)

	if model == nil {
		return allowed
	}

	valType := reflect.TypeOf(model)

	if valType.Kind() == reflect.Ptr {
		valType = valType.Elem()
	}

	if valType.Kind() != reflect.Struct {
		return allowed
	}

	// Fixed: use index-based loop, reflect.Type has no .Fields() iterator
	for i := 0; i < valType.NumField(); i++ {
		field := valType.Field(i)

		if field.Name == "ID" {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			tagParts := strings.Split(jsonTag, ",")
			mainTag := tagParts[0]
			if mainTag != "" {
				allowed[mainTag] = true
			}
		}
	}

	return allowed
}

// isFieldEmpty checks if a field value is empty/zero
func isFieldEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map, reflect.Chan:
		return v.Len() == 0
	case reflect.Struct:
		return isStructEmpty(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Invalid:
		return true
	default:
		if v.CanInterface() {
			return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
		}
		return false
	}
}

// isStructEmpty checks if all fields in a struct are empty
// Fixed: use index-based loop, reflect.Value has no .Fields() iterator
func isStructEmpty(v reflect.Value) bool {
	for i := 0; i < v.NumField(); i++ {
		fieldValue := v.Field(i)
		if !fieldValue.CanInterface() {
			continue
		}
		if !isFieldEmpty(fieldValue) {
			return false
		}
	}
	return true
}

func checkEmail(email string) (bool, error) {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return false, fmt.Errorf("invalid email address")
	}
	return true, nil
}
