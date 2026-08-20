package domain

import (
	"encoding/json"
	"fmt"
	contract "github.com/example/event-contract-quality-gateway/internal/contract/domain"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// TypeIssue describes a payload value that does not satisfy its contract
// field. It is intentionally independent from Rule results so contract
// failures and business-quality failures can be distinguished in telemetry.
type TypeIssue struct {
	Field    string `json:"field"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Reason   string `json:"reason"`
}

// ValidatePayload checks primitive JSON and protobuf-compatible scalar types,
// required fields and enum constraints. The gateway accepts map[string]any
// after decoding JSON, so this function normalizes Go's json.Number and
// integer representations before checking them.
func ValidatePayload(fields map[string]any, schemaFields []contract.Field) []TypeIssue {
	issues := make([]TypeIssue, 0)
	for _, field := range schemaFields {
		value, exists := fields[field.Name]
		if !exists || value == nil {
			if field.Required {
				issues = append(issues, TypeIssue{Field: field.Name, Expected: field.Type, Actual: "null", Reason: "required field is missing"})
			}
			continue
		}
		if !matchesType(value, field.Type) {
			issues = append(issues, TypeIssue{Field: field.Name, Expected: field.Type, Actual: valueType(value), Reason: "value has incompatible type"})
			continue
		}
		if len(field.Enum) > 0 && !enumContains(field.Enum, fmt.Sprint(value)) {
			issues = append(issues, TypeIssue{Field: field.Name, Expected: strings.Join(field.Enum, ","), Actual: fmt.Sprint(value), Reason: "value is not in enum"})
		}
	}
	return issues
}

func matchesType(value any, fieldType string) bool {
	t := strings.ToLower(strings.TrimSpace(fieldType))
	switch t {
	case "string", "text", "uuid", "date", "datetime", "timestamp":
		return isString(value)
	case "bool", "boolean":
		_, ok := value.(bool)
		return ok
	case "int", "int8", "int16", "int32", "int64", "integer":
		return isInteger(value)
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return isInteger(value) && numericValue(value) >= 0
	case "float", "float32", "float64", "number", "decimal":
		return isNumber(value)
	case "object", "map", "json":
		kind := reflect.ValueOf(value).Kind()
		return kind == reflect.Map || kind == reflect.Struct
	case "array", "list", "repeated":
		kind := reflect.ValueOf(value).Kind()
		return kind == reflect.Array || kind == reflect.Slice
	case "bytes", "binary":
		return isString(value) || isByteSlice(value)
	default:
		// Unknown custom types are kept permissive for schema evolution. The
		// schema parser remains responsible for rejecting malformed type names.
		return true
	}
}

func enumContains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func isString(value any) bool {
	_, ok := value.(string)
	return ok
}

func isByteSlice(value any) bool {
	_, ok := value.([]byte)
	return ok
}

func isInteger(value any) bool {
	switch typed := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, json.Number:
		if number, ok := value.(json.Number); ok {
			_, err := strconv.ParseInt(number.String(), 10, 64)
			return err == nil
		}
		return reflect.ValueOf(typed).IsValid()
	case float32:
		return !math.IsNaN(float64(typed)) && float32(int64(typed)) == typed
	case float64:
		return !math.IsNaN(typed) && float64(int64(typed)) == typed
	default:
		return false
	}
}

func isNumber(value any) bool {
	switch typed := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, json.Number:
		if number, ok := value.(json.Number); ok {
			_, err := strconv.ParseFloat(number.String(), 64)
			return err == nil
		}
		return !math.IsNaN(reflect.ValueOf(typed).Convert(reflect.TypeOf(float64(0))).Float())
	default:
		return false
	}
}

func numericValue(value any) float64 {
	switch typed := value.(type) {
	case json.Number:
		v, _ := typed.Float64()
		return v
	case float32:
		return float64(typed)
	case float64:
		return typed
	default:
		return reflect.ValueOf(value).Convert(reflect.TypeOf(float64(0))).Float()
	}
}

func valueType(value any) string {
	if value == nil {
		return "null"
	}
	return reflect.TypeOf(value).String()
}
