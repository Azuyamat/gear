package command

import (
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// Flag is a named option. Create one with NewStringFlag and the other
// constructors, or with NewFlag.
type Flag interface {
	Name() string
	Shorthand() string
	Description() string
	Expected() ValueType
	DefaultValue() any
	toFlag() flag
}

type flag struct {
	name         string
	shorthand    string
	description  string
	expected     ValueType
	defaultValue any
	validators   []validator
	choices      []string
	complete     CompleteFunc
	env          string
	required     bool
	requires     []string
	negatable    bool
}

func (f flag) Name() string        { return f.name }
func (f flag) Shorthand() string   { return f.shorthand }
func (f flag) Description() string { return f.description }
func (f flag) Expected() ValueType { return f.expected }
func (f flag) DefaultValue() any   { return f.defaultValue }
func (f flag) toFlag() flag        { return f }

// takesValue reports whether the flag reads a value from the command line,
// which every flag but a bool one does.
func (f flag) takesValue() bool { return f.expected != ValueTypeBool }

// check reports mistakes in how a single flag is set up.
func (f flag) check() error {
	switch {
	case f.name == "":
		return fmt.Errorf("a flag has no name")
	case strings.HasPrefix(f.name, "-"):
		return fmt.Errorf("flag %q: leave the dashes out of the name", f.name)
	case strings.ContainsAny(f.name, "= \t"):
		return fmt.Errorf("flag %q: names can't contain '=' or spaces", f.name)
	case f.name == "help":
		return fmt.Errorf("--help is reserved for showing help")
	case utf8.RuneCountInString(f.shorthand) > 1:
		return fmt.Errorf("flag --%s: shorthand %q must be a single character", f.name, f.shorthand)
	case f.shorthand == "-" || f.shorthand == "=":
		return fmt.Errorf("flag --%s: %q can't be a shorthand", f.name, f.shorthand)
	case f.negatable && f.expected != ValueTypeBool:
		return fmt.Errorf("flag --%s: only bool flags can be negatable", f.name)
	}
	return nil
}

// TypedFlag is a flag whose value is a T. Its methods return a changed copy,
// so they chain:
//
//	command.NewStringFlag("token", "", "API token", "").Env("APP_TOKEN").Required()
type TypedFlag[T any] struct {
	flag
}

// ExtendValidators adds checks that run on the flag's value when it's set on
// the command line or from its environment variable. Defaults aren't checked.
func (f TypedFlag[T]) ExtendValidators(validators ...func(T) error) TypedFlag[T] {
	f.validators = slices.Clip(f.validators)
	for _, v := range validators {
		f.validators = append(f.validators, toValidator(v))
	}
	return f
}

// Choices limits the flag to the given values, compared as typed. Help lists
// them in place of the type.
func (f TypedFlag[T]) Choices(choices ...string) TypedFlag[T] {
	f.choices = slices.Clone(choices)
	return f
}

// Complete sets a function that suggests values for the flag during shell
// completion. Choices are suggested without one.
func (f TypedFlag[T]) Complete(fn CompleteFunc) TypedFlag[T] {
	f.complete = fn
	return f
}

// Env reads the flag from the environment variable name when it isn't on the
// command line. An empty variable counts as unset. A strings flag splits the
// variable's value on commas.
func (f TypedFlag[T]) Env(name string) TypedFlag[T] {
	f.env = name
	return f
}

// Required makes it an error to leave the flag unset. Setting it from its
// environment variable counts.
func (f TypedFlag[T]) Required() TypedFlag[T] {
	f.required = true
	return f
}

// Requires makes the flag valid only alongside the named flags.
func (f TypedFlag[T]) Requires(names ...string) TypedFlag[T] {
	f.requires = append(slices.Clip(f.requires), names...)
	return f
}

// Negatable adds a --no-<name> form that sets a bool flag to false. It panics
// on a flag of any other type when the flag is added to a command.
func (f TypedFlag[T]) Negatable() TypedFlag[T] {
	f.negatable = true
	return f
}

// NewFlag creates a flag of the given type whose value is read untyped.
// Prefer NewStringFlag and the other typed constructors.
func NewFlag(name, shorthand, description string, expected ValueType, defaultValue any, validators ...func(any) error) TypedFlag[any] {
	return newTypedFlag[any](name, shorthand, description, expected, defaultValue).ExtendValidators(validators...)
}

func newTypedFlag[T any](name, shorthand, description string, expected ValueType, defaultValue T) TypedFlag[T] {
	return TypedFlag[T]{flag{
		name:         name,
		shorthand:    shorthand,
		description:  description,
		expected:     expected,
		defaultValue: defaultValue,
	}}
}

// NewStringFlag creates a flag that takes any text. shorthand is a single
// character, or "" for none.
func NewStringFlag(name, shorthand, description, defaultValue string) TypedFlag[string] {
	return newTypedFlag(name, shorthand, description, ValueTypeString, defaultValue)
}

// NewIntFlag creates a flag that takes a whole number.
func NewIntFlag(name, shorthand, description string, defaultValue int) TypedFlag[int] {
	return newTypedFlag(name, shorthand, description, ValueTypeInt, defaultValue)
}

// NewFloatFlag creates a flag that takes a number.
func NewFloatFlag(name, shorthand, description string, defaultValue float64) TypedFlag[float64] {
	return newTypedFlag(name, shorthand, description, ValueTypeFloat, defaultValue)
}

// NewBoolFlag creates a flag that's true when given on its own. It also takes
// an explicit value with --name=false.
func NewBoolFlag(name, shorthand, description string, defaultValue bool) TypedFlag[bool] {
	return newTypedFlag(name, shorthand, description, ValueTypeBool, defaultValue)
}

// NewDurationFlag creates a flag that takes a duration such as 1h30m.
func NewDurationFlag(name, shorthand, description string, defaultValue time.Duration) TypedFlag[time.Duration] {
	return newTypedFlag(name, shorthand, description, ValueTypeDuration, defaultValue)
}

// NewStringsFlag creates a flag that can be given more than once, collecting
// each value into a list. Giving it at all replaces the default.
func NewStringsFlag(name, shorthand, description string, defaultValue []string) TypedFlag[[]string] {
	return newTypedFlag(name, shorthand, description, ValueTypeStrings, defaultValue)
}
