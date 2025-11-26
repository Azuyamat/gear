package command

type Arg interface {
	Label() string
	Description() string
	Expected() ValueType
	IsOptional() bool
	IsVariadic() bool
	validate(value interface{}) error
	toArg() arg
}

type arg struct {
	label       string
	description string
	expected    ValueType
	validators  []validator
	optional    bool
	variadic    bool
}

func (a arg) Label() string {
	return a.label
}

func (a arg) Description() string {
	return a.description
}

func (a arg) Expected() ValueType {
	return a.expected
}

func (a arg) IsOptional() bool {
	return a.optional
}

func (a arg) IsVariadic() bool {
	return a.variadic
}

func (a arg) toArg() arg {
	return a
}

func NewArg(label string, description string, expected ValueType, validators ...validator) arg {
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

func (a arg) AsVariadic() arg {
	a.variadic = true
	return a
}

func (a arg) ExtendValidators(validators ...validator) arg {
	a.validators = append(a.validators, validators...)
	return a
}

type typedArg[T any] struct {
	arg
}

func (a typedArg[T]) ExtendValidators(validators ...func(T) error) typedArg[T] {
	for _, v := range validators {
		a.arg.validators = append(a.arg.validators, toValidator(v))
	}
	return a
}

func (a typedArg[T]) AsOptional() typedArg[T] {
	a.arg.optional = true
	return a
}

func (a typedArg[T]) AsVariadic() typedArg[T] {
	a.arg.variadic = true
	return a
}

func (a typedArg[T]) toArg() arg {
	return a.arg
}

func NewStringArg(label string, description string) typedArg[string] {
	return typedArg[string]{
		arg: NewArg(label, description, ValueTypeString, validateString),
	}
}

func NewIntArg(label string, description string) typedArg[int] {
	return typedArg[int]{
		arg: NewArg(label, description, ValueTypeInt, validateInt),
	}
}

func NewFloatArg(label string, description string) typedArg[float64] {
	return typedArg[float64]{
		arg: NewArg(label, description, ValueTypeFloat, validateFloat),
	}
}

func NewBoolArg(label string, description string) typedArg[bool] {
	return typedArg[bool]{
		arg: NewArg(label, description, ValueTypeBool, validateBool),
	}
}

func (a arg) validate(value interface{}) error {
	return runValidators(a.validators, value)
}
