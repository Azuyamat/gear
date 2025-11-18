package gear

import "fmt"

type ArgType string
type argValidator func(value interface{}) error

const (
	ArgTypeString ArgType = "string"
	ArgTypeInt    ArgType = "int"
	ArgTypeFloat  ArgType = "float"
	ArgTypeBool   ArgType = "bool"
)

type arg struct {
	label       string
	description string
	expected    ArgType
	validators  []argValidator
	optional    bool
}

func (a *arg) Label() string {
	return a.label
}

func (a *arg) Description() string {
	return a.description
}

func (a *arg) Expected() ArgType {
	return a.expected
}

func (a *arg) IsOptional() bool {
	return a.optional
}

func NewArg(label string, description string, expected ArgType, validators ...argValidator) arg {
	return arg{
		label:       label,
		description: description,
		expected:    expected,
		validators:  validators,
	}
}

func (a arg) AsOptional() arg {
	a.optional = true
	return a
}

func NewStringArg(label string, description string) arg {
	return NewArg(label, description, ArgTypeString, func(value interface{}) error {
		_, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
		return nil
	})
}

func NewIntArg(label string, description string) arg {
	return NewArg(label, description, ArgTypeInt, func(value interface{}) error {
		_, ok := value.(int)
		if !ok {
			return fmt.Errorf("expected int, got %T", value)
		}
		return nil
	})
}

func NewFloatArg(label string, description string) arg {
	return NewArg(label, description, ArgTypeFloat, func(value interface{}) error {
		_, ok := value.(float64)
		if !ok {
			return fmt.Errorf("expected float, got %T", value)
		}
		return nil
	})
}

func NewBoolArg(label string, description string) arg {
	return NewArg(label, description, ArgTypeBool, func(value interface{}) error {
		_, ok := value.(bool)
		if !ok {
			return fmt.Errorf("expected bool, got %T", value)
		}
		return nil
	})
}

func (a arg) ExtendValidators(validators ...argValidator) arg {
	a.validators = append(a.validators, validators...)
	return a
}

func (a arg) validate(value interface{}) error {
	for _, validator := range a.validators {
		if err := validator(value); err != nil {
			return err
		}
	}
	return nil
}
