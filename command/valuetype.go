package command

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// ValueType names the kind of value an argument or flag holds.
type ValueType string

const (
	ValueTypeString   ValueType = "string"
	ValueTypeInt      ValueType = "int"
	ValueTypeFloat    ValueType = "float"
	ValueTypeBool     ValueType = "bool"
	ValueTypeDuration ValueType = "duration"
	// ValueTypeStrings is a list of strings. A flag of this type can be given
	// more than once, and each use adds to the list.
	ValueTypeStrings ValueType = "strings"
)

type validator func(value any) error

func toValidator[T any](typed func(T) error) validator {
	return func(value any) error {
		t, ok := value.(T)
		if !ok {
			return fmt.Errorf("expected %T, got %T", *new(T), value)
		}
		return typed(t)
	}
}

func runValidators(validators []validator, value any) error {
	for _, v := range validators {
		if err := v(value); err != nil {
			return err
		}
	}
	return nil
}

// parseValue converts raw to the type t names. Its errors say what was
// wanted but not what was being parsed, so callers put the name in front:
// "'count' must be a whole number".
func parseValue(raw string, t ValueType) (any, error) {
	switch t {
	case ValueTypeString, ValueTypeStrings:
		return raw, nil
	case ValueTypeInt:
		n, err := strconv.Atoi(raw)
		if errors.Is(err, strconv.ErrRange) {
			return nil, fmt.Errorf("is out of range (got %q)", raw)
		}
		if err != nil {
			return nil, fmt.Errorf("must be a whole number (got %q)", raw)
		}
		return n, nil
	case ValueTypeFloat:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("must be a number (got %q)", raw)
		}
		return f, nil
	case ValueTypeBool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("must be true or false (got %q)", raw)
		}
		return b, nil
	case ValueTypeDuration:
		d, err := time.ParseDuration(raw)
		if err != nil {
			return nil, fmt.Errorf("must be a duration such as 30s, 5m or 1h30m (got %q)", raw)
		}
		return d, nil
	default:
		return nil, fmt.Errorf("has unsupported type %q", t)
	}
}

// typedSlice turns the values of a variadic argument into a slice of their
// own type, such as []string, so ArgValue can return it.
func typedSlice(t ValueType, values []any) any {
	switch t {
	case ValueTypeString, ValueTypeStrings:
		return convertSlice[string](values)
	case ValueTypeInt:
		return convertSlice[int](values)
	case ValueTypeFloat:
		return convertSlice[float64](values)
	case ValueTypeBool:
		return convertSlice[bool](values)
	case ValueTypeDuration:
		return convertSlice[time.Duration](values)
	default:
		return values
	}
}

func convertSlice[T any](values []any) []T {
	out := make([]T, len(values))
	for i, v := range values {
		out[i], _ = v.(T)
	}
	return out
}

func isZero(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice {
		return rv.Len() == 0
	}
	return rv.IsZero()
}

func formatDefault(v any) string {
	switch v := v.(type) {
	case string:
		return strconv.Quote(v)
	case []string:
		return strings.Join(v, ",")
	default:
		return fmt.Sprint(v)
	}
}
