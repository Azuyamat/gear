package command

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// Arg is a positional argument. Create one with NewStringArg and the other
// constructors, or with NewArg.
type Arg interface {
	Label() string
	Description() string
	Expected() ValueType
	IsOptional() bool
	IsVariadic() bool
	toArg() arg
}

type arg struct {
	label       string
	description string
	expected    ValueType
	validators  []validator
	choices     []string
	complete    CompleteFunc
	optional    bool
	variadic    bool
}

func (a arg) Label() string       { return a.label }
func (a arg) Description() string { return a.description }
func (a arg) Expected() ValueType { return a.expected }
func (a arg) IsOptional() bool    { return a.optional }
func (a arg) IsVariadic() bool    { return a.variadic }
func (a arg) toArg() arg          { return a }

// usageName is the label as help lists it, with "..." if it's variadic.
func (a arg) usageName() string {
	if a.variadic {
		return a.label + "..."
	}
	return a.label
}

// inUsageLine is how the argument appears in a usage line: <name> or [name].
func (a arg) inUsageLine() string {
	if a.optional {
		return "[" + a.usageName() + "]"
	}
	return "<" + a.usageName() + ">"
}

// parse converts one raw value and runs the argument's checks on it.
func (a arg) parse(raw string) (any, error) {
	if len(a.choices) > 0 && !slices.Contains(a.choices, raw) {
		return nil, fmt.Errorf("'%s' must be one of %s (got %q)", a.label, strings.Join(a.choices, ", "), raw)
	}
	value, err := parseValue(raw, a.expected)
	if err != nil {
		return nil, fmt.Errorf("'%s' %w", a.label, err)
	}
	if err := runValidators(a.validators, value); err != nil {
		return nil, fmt.Errorf("invalid value %q for '%s': %w", raw, a.label, err)
	}
	return value, nil
}

// TypedArg is an argument whose value is a T. Its methods return a changed
// copy, so they chain:
//
//	command.NewIntArg("count", "How many").AsOptional().ExtendValidators(positive)
type TypedArg[T any] struct {
	arg
}

// ExtendValidators adds checks that run on each parsed value. An error from
// one is reported as invalid input.
func (a TypedArg[T]) ExtendValidators(validators ...func(T) error) TypedArg[T] {
	a.validators = slices.Clip(a.validators)
	for _, v := range validators {
		a.validators = append(a.validators, toValidator(v))
	}
	return a
}

// AsOptional lets the argument be left out. Every argument after an optional
// one must be optional too.
func (a TypedArg[T]) AsOptional() TypedArg[T] {
	a.optional = true
	return a
}

// AsVariadic makes the argument take every remaining value, so it must be the
// last one. Read its values with ArgValue[[]T].
func (a TypedArg[T]) AsVariadic() TypedArg[T] {
	a.variadic = true
	return a
}

// Choices limits the argument to the given values, compared as typed.
func (a TypedArg[T]) Choices(choices ...string) TypedArg[T] {
	a.choices = slices.Clone(choices)
	return a
}

// Complete sets a function that suggests values for the argument during shell
// completion. Choices are suggested without one.
func (a TypedArg[T]) Complete(fn CompleteFunc) TypedArg[T] {
	a.complete = fn
	return a
}

// NewArg creates an argument of the given type whose value is read untyped.
// Prefer NewStringArg and the other typed constructors.
func NewArg(label, description string, expected ValueType, validators ...func(any) error) TypedArg[any] {
	return newTypedArg[any](label, description, expected).ExtendValidators(validators...)
}

func newTypedArg[T any](label, description string, expected ValueType) TypedArg[T] {
	return TypedArg[T]{arg{label: label, description: description, expected: expected}}
}

// NewStringArg creates an argument that takes any text.
func NewStringArg(label, description string) TypedArg[string] {
	return newTypedArg[string](label, description, ValueTypeString)
}

// NewIntArg creates an argument that takes a whole number, negative ones
// included.
func NewIntArg(label, description string) TypedArg[int] {
	return newTypedArg[int](label, description, ValueTypeInt)
}

// NewFloatArg creates an argument that takes a number.
func NewFloatArg(label, description string) TypedArg[float64] {
	return newTypedArg[float64](label, description, ValueTypeFloat)
}

// NewBoolArg creates an argument that takes true or false (or 1, 0, t, f and
// the other spellings strconv.ParseBool accepts).
func NewBoolArg(label, description string) TypedArg[bool] {
	return newTypedArg[bool](label, description, ValueTypeBool)
}

// NewDurationArg creates an argument that takes a duration such as 1h30m.
func NewDurationArg(label, description string) TypedArg[time.Duration] {
	return newTypedArg[time.Duration](label, description, ValueTypeDuration)
}

// checkArgs reports mistakes in how a command's arguments are set up.
func checkArgs(args []Arg) error {
	seen := map[string]bool{}
	var optional, variadic string
	for _, a := range args {
		s := a.toArg()
		switch {
		case s.label == "":
			return fmt.Errorf("an argument has no label")
		case seen[s.label]:
			return fmt.Errorf("two arguments are labelled %q", s.label)
		case variadic != "":
			return fmt.Errorf("argument %q comes after variadic argument %q, which must be last", s.label, variadic)
		case !s.optional && optional != "":
			return fmt.Errorf("required argument %q comes after optional argument %q", s.label, optional)
		}
		seen[s.label] = true
		if s.optional {
			optional = s.label
		}
		if s.variadic {
			variadic = s.label
		}
	}
	return nil
}
