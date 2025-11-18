package command

import (
	"fmt"
	"strconv"
)

type ValueType string

const (
	ValueTypeString ValueType = "string"
	ValueTypeInt    ValueType = "int"
	ValueTypeFloat  ValueType = "float"
	ValueTypeBool   ValueType = "bool"
)

type validator func(value interface{}) error

func toValidator[T any](typedValidator func(T) error) validator {
	return func(value interface{}) error {
		typedValue, ok := value.(T)
		if !ok {
			return fmt.Errorf("expected %T, got %T", *new(T), value)
		}
		return typedValidator(typedValue)
	}
}

func parseValue(value string, expectedType ValueType) (interface{}, error) {
	switch expectedType {
	case ValueTypeString:
		return value, nil
	case ValueTypeInt:
		return strconv.Atoi(value)
	case ValueTypeFloat:
		return strconv.ParseFloat(value, 64)
	case ValueTypeBool:
		if value == "" {
			return true, nil
		}
		return strconv.ParseBool(value)
	default:
		return nil, fmt.Errorf("unsupported value type: %s", expectedType)
	}
}

func validateString(value interface{}) error {
	_, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", value)
	}
	return nil
}

func validateInt(value interface{}) error {
	_, ok := value.(int)
	if !ok {
		return fmt.Errorf("expected int, got %T", value)
	}
	return nil
}

func validateFloat(value interface{}) error {
	_, ok := value.(float64)
	if !ok {
		return fmt.Errorf("expected float, got %T", value)
	}
	return nil
}

func validateBool(value interface{}) error {
	_, ok := value.(bool)
	if !ok {
		return fmt.Errorf("expected bool, got %T", value)
	}
	return nil
}

func runValidators(validators []validator, value interface{}) error {
	for _, v := range validators {
		if err := v(value); err != nil {
			return err
		}
	}
	return nil
}
