package command

// ValidatedArgs holds the arguments and flags of one run, parsed and checked.
// Read them with ArgValue and FlagValue.
type ValidatedArgs struct {
	args    map[string]any
	flags   map[string]any
	changed map[string]bool
}

func newValidatedArgs() *ValidatedArgs {
	return &ValidatedArgs{
		args:    map[string]any{},
		flags:   map[string]any{},
		changed: map[string]bool{},
	}
}

// ArgValue returns the argument labelled label as a T, or T's zero value if it
// wasn't given or isn't a T. A variadic argument's value is a slice, such as
// ArgValue[[]string](args, "files").
func ArgValue[T any](v ValidatedArgs, label string) T {
	t, _ := v.args[label].(T)
	return t
}

// FlagValue returns the flag named name as a T: the value the command line or
// the flag's environment variable gave it, or else its default. It returns T's
// zero value if there's no such flag or it isn't a T.
func FlagValue[T any](v ValidatedArgs, name string) T {
	t, _ := v.flags[name].(T)
	return t
}

// Has reports whether the argument labelled label was given. A variadic
// argument counts as given when it got at least one value.
func (v ValidatedArgs) Has(label string) bool {
	_, ok := v.args[label]
	return ok
}

// Changed reports whether the flag named name was set on the command line or
// from its environment variable, rather than left at its default.
func (v ValidatedArgs) Changed(name string) bool {
	return v.changed[name]
}
