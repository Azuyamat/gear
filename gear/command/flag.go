package command

type Flag interface {
	Name() string
	Shorthand() string
	Description() string
	Expected() ValueType
	DefaultValue() interface{}
	validate(value interface{}) error
	toFlag() flag
}

type flag struct {
	name         string
	shorthand    string
	description  string
	expected     ValueType
	validators   []validator
	defaultValue interface{}
}

func (f flag) Name() string {
	return f.name
}

func (f flag) Shorthand() string {
	return f.shorthand
}

func (f flag) Description() string {
	return f.description
}

func (f flag) Expected() ValueType {
	return f.expected
}

func (f flag) DefaultValue() interface{} {
	return f.defaultValue
}

func (f flag) toFlag() flag {
	return f
}

func NewFlag(name string, shorthand string, description string, expected ValueType, defaultValue interface{}, validators ...validator) flag {
	return flag{
		name:         name,
		shorthand:    shorthand,
		description:  description,
		expected:     expected,
		defaultValue: defaultValue,
		validators:   validators,
	}
}

func (f flag) ExtendValidators(validators ...validator) flag {
	f.validators = append(f.validators, validators...)
	return f
}

type typedFlag[T any] struct {
	flag
}

func (f typedFlag[T]) ExtendValidators(validators ...func(T) error) typedFlag[T] {
	for _, v := range validators {
		f.flag.validators = append(f.flag.validators, toValidator(v))
	}
	return f
}

func (f typedFlag[T]) toFlag() flag {
	return f.flag
}

func NewStringFlag(name string, shorthand string, description string, defaultValue string) typedFlag[string] {
	return typedFlag[string]{
		flag: NewFlag(name, shorthand, description, ValueTypeString, defaultValue, validateString),
	}
}

func NewIntFlag(name string, shorthand string, description string, defaultValue int) typedFlag[int] {
	return typedFlag[int]{
		flag: NewFlag(name, shorthand, description, ValueTypeInt, defaultValue, validateInt),
	}
}

func NewFloatFlag(name string, shorthand string, description string, defaultValue float64) typedFlag[float64] {
	return typedFlag[float64]{
		flag: NewFlag(name, shorthand, description, ValueTypeFloat, defaultValue, validateFloat),
	}
}

func NewBoolFlag(name string, shorthand string, description string, defaultValue bool) typedFlag[bool] {
	return typedFlag[bool]{
		flag: NewFlag(name, shorthand, description, ValueTypeBool, defaultValue, validateBool),
	}
}

func (f flag) validate(value interface{}) error {
	return runValidators(f.validators, value)
}
