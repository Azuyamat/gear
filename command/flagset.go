package command

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// flagSet is the flags one command accepts: its own, then the global ones.
type flagSet struct {
	flags       []flag
	global      map[string]bool
	byName      map[string]flag
	byShorthand map[string]flag
	negated     map[string]flag // "no-x" -> x
	exclusive   [][]string
}

func newFlagSet(own, global []Flag, exclusive [][]string) (*flagSet, error) {
	fs := &flagSet{
		global:      map[string]bool{},
		byName:      map[string]flag{},
		byShorthand: map[string]flag{},
		negated:     map[string]flag{},
		exclusive:   exclusive,
	}
	for _, f := range own {
		if err := fs.add(f.toFlag(), false); err != nil {
			return nil, err
		}
	}
	for _, f := range global {
		if err := fs.add(f.toFlag(), true); err != nil {
			return nil, err
		}
	}
	return fs, nil
}

func mustFlagSet(own, global []Flag, exclusive [][]string) *flagSet {
	fs, err := newFlagSet(own, global, exclusive)
	if err != nil {
		panic("gear: " + err.Error())
	}
	return fs
}

func (fs *flagSet) add(f flag, global bool) error {
	if err := f.check(); err != nil {
		return err
	}
	spellings := []string{f.name}
	if f.negatable {
		spellings = append(spellings, "no-"+f.name)
	}
	for _, s := range spellings {
		if other, ok := fs.longOwner(s); ok {
			return fs.clash("--"+s, other, f, global)
		}
	}
	if other, ok := fs.byShorthand[f.shorthand]; ok && f.shorthand != "" {
		return fs.clash("-"+f.shorthand, other, f, global)
	}

	fs.flags = append(fs.flags, f)
	fs.global[f.name] = global
	fs.byName[f.name] = f
	if f.shorthand != "" {
		fs.byShorthand[f.shorthand] = f
	}
	if f.negatable {
		fs.negated["no-"+f.name] = f
	}
	return nil
}

func (fs *flagSet) longOwner(spelling string) (flag, bool) {
	if f, ok := fs.byName[spelling]; ok {
		return f, true
	}
	f, ok := fs.negated[spelling]
	return f, ok
}

func (fs *flagSet) clash(spelling string, existing, f flag, global bool) error {
	describe := func(f flag, global bool) string {
		if global {
			return "global flag --" + f.name
		}
		return "flag --" + f.name
	}
	if existing.name == f.name && fs.global[existing.name] == global {
		return fmt.Errorf("%s is defined twice", describe(f, global))
	}
	return fmt.Errorf("%s is used by both %s and %s", spelling, describe(existing, fs.global[existing.name]), describe(f, global))
}

// checkRefs reports Requires and MutuallyExclusive names that aren't flags.
// It's separate from newFlagSet because a command's own flags can refer to
// global ones, which are only known once the tree is complete.
func (fs *flagSet) checkRefs() error {
	for _, f := range fs.flags {
		for _, r := range f.requires {
			if _, ok := fs.byName[r]; !ok {
				return fmt.Errorf("flag --%s requires --%s, which doesn't exist", f.name, r)
			}
		}
	}
	for _, group := range fs.exclusive {
		if len(group) < 2 {
			return fmt.Errorf("MutuallyExclusive needs at least two flags, got %v", group)
		}
		for _, name := range group {
			if _, ok := fs.byName[name]; !ok {
				return fmt.Errorf("MutuallyExclusive names --%s, which doesn't exist", name)
			}
		}
	}
	return nil
}

// parse reads the flags in args into v and returns the positional arguments.
// When interspersed is false it stops at the first positional one, returning
// it and everything after it unread. It returns ErrHelp when --help or -h is
// given.
func (fs *flagSet) parse(args []string, v *ValidatedArgs, interspersed bool) ([]string, error) {
	var positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		var err error
		switch {
		case a == "--":
			return append(positional, args[i+1:]...), nil
		case strings.HasPrefix(a, "--"):
			i, err = fs.parseLong(a[2:], args, i, v)
		case len(a) > 1 && a[0] == '-' && !fs.isNegativeNumber(a):
			i, err = fs.parseShort(a[1:], args, i, v)
		case !interspersed:
			return append(positional, args[i:]...), nil
		default:
			positional = append(positional, a)
		}
		if err != nil {
			return nil, err
		}
	}
	return positional, nil
}

// isNegativeNumber reports whether a is a value like -1 or -2.5 rather than a
// flag. It's a flag if a digit shorthand claims its first character.
func (fs *flagSet) isNegativeNumber(a string) bool {
	if _, taken := fs.byShorthand[a[1:2]]; taken {
		return false
	}
	_, err := strconv.ParseFloat(a, 64)
	return err == nil && (a[1] == '.' || a[1] >= '0' && a[1] <= '9')
}

// parseLong handles --name, --name=value and --name value, where args[i] is
// the flag. It returns the index of the last argument it used.
func (fs *flagSet) parseLong(body string, args []string, i int, v *ValidatedArgs) (int, error) {
	name, value, hasValue := strings.Cut(body, "=")
	f, ok := fs.byName[name]
	if !ok {
		if f, ok := fs.negated[name]; ok {
			if hasValue {
				return i, fmt.Errorf("--%s doesn't take a value", name)
			}
			return i, fs.set(v, f, "false")
		}
		if name == "help" {
			return i, ErrHelp
		}
		return i, fs.unknown("--" + name)
	}
	switch {
	case hasValue:
	case !f.takesValue():
		value = "true"
	case i+1 < len(args):
		i++
		value = args[i]
	default:
		return i, fmt.Errorf("flag --%s needs a value", name)
	}
	return i, fs.set(v, f, value)
}

// parseShort handles -x, -x value, -xvalue, -x=value and runs of bool flags
// such as -abc, where args[i] is the flag. It returns the index of the last
// argument it used.
func (fs *flagSet) parseShort(body string, args []string, i int, v *ValidatedArgs) (int, error) {
	for k, r := range body {
		s := string(r)
		f, ok := fs.byShorthand[s]
		if !ok {
			if s == "h" {
				return i, ErrHelp
			}
			if len(body) > len(s) {
				return i, fmt.Errorf("unknown flag -%s in -%s", s, body)
			}
			return i, fs.unknown("-" + s)
		}
		rest := body[k+utf8.RuneLen(r):]
		if !f.takesValue() {
			if value, ok := strings.CutPrefix(rest, "="); ok {
				return i, fs.set(v, f, value)
			}
			if err := fs.set(v, f, "true"); err != nil {
				return i, err
			}
			continue
		}
		value := strings.TrimPrefix(rest, "=")
		if rest == "" {
			if i+1 >= len(args) {
				return i, fmt.Errorf("flag -%s needs a value", s)
			}
			i++
			value = args[i]
		}
		return i, fs.set(v, f, value)
	}
	return i, nil
}

func (fs *flagSet) set(v *ValidatedArgs, f flag, raw string) error {
	if len(f.choices) > 0 && !slices.Contains(f.choices, raw) {
		return fmt.Errorf("--%s must be one of %s (got %q)", f.name, strings.Join(f.choices, ", "), raw)
	}
	value, err := parseValue(raw, f.expected)
	if err != nil {
		return fmt.Errorf("--%s %w", f.name, err)
	}
	if f.expected == ValueTypeStrings {
		var list []string
		if v.changed[f.name] {
			list, _ = v.flags[f.name].([]string)
		}
		value = append(slices.Clip(list), value.(string))
	}
	v.flags[f.name] = value
	v.changed[f.name] = true
	return nil
}

func (fs *flagSet) unknown(spelling string) error {
	msg := "unknown flag " + spelling
	if name, ok := strings.CutPrefix(spelling, "--"); ok {
		var names []string
		for _, f := range fs.flags {
			names = append(names, f.name)
		}
		names = append(names, "help")
		if s := suggest(name, names); len(s) > 0 {
			msg += " (did you mean --" + s[0] + "?)"
		}
	}
	return errors.New(msg)
}

// finish fills in what the command line left unset, from environment
// variables and then defaults, and checks the rules that span the whole
// command line: required flags, validators, Requires and MutuallyExclusive.
func (fs *flagSet) finish(v *ValidatedArgs) error {
	for _, f := range fs.flags {
		if v.changed[f.name] {
			continue
		}
		if raw := os.Getenv(f.env); f.env != "" && raw != "" {
			if err := fs.setFromEnv(v, f, raw); err != nil {
				return err
			}
			continue
		}
		if f.defaultValue != nil {
			v.flags[f.name] = f.defaultValue
		}
	}

	for _, f := range fs.flags {
		if !v.changed[f.name] {
			if f.required {
				return fmt.Errorf("required flag --%s isn't set%s", f.name, envHint(f))
			}
			continue
		}
		if err := runValidators(f.validators, v.flags[f.name]); err != nil {
			return fmt.Errorf("invalid value for --%s: %w", f.name, err)
		}
		for _, r := range f.requires {
			if !v.changed[r] {
				return fmt.Errorf("--%s can only be used with --%s", f.name, r)
			}
		}
	}

	for _, group := range fs.exclusive {
		var set []string
		for _, name := range group {
			if v.changed[name] {
				set = append(set, "--"+name)
			}
		}
		if len(set) > 1 {
			return fmt.Errorf("%s can't be used together", joinAnd(set))
		}
	}
	return nil
}

func (fs *flagSet) setFromEnv(v *ValidatedArgs, f flag, raw string) error {
	values := []string{raw}
	if f.expected == ValueTypeStrings {
		values = strings.Split(raw, ",")
	}
	for _, value := range values {
		if err := fs.set(v, f, strings.TrimSpace(value)); err != nil {
			return fmt.Errorf("$%s: %w", f.env, err)
		}
	}
	return nil
}

func envHint(f flag) string {
	if f.env == "" {
		return ""
	}
	return " (or set $" + f.env + ")"
}

// joinAnd joins items as "a, b and c".
func joinAnd(items []string) string {
	if len(items) < 2 {
		return strings.Join(items, "")
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}
