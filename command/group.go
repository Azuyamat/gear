package command

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// group is the list of children shared by RootCommand and Subcommand. It keeps
// them in the order they were added, which is the order help lists them in.
type group struct {
	children []Command
}

func (g *group) subcommands() *group { return g }

func (g *group) add(owner string, c Command) {
	if c == nil {
		panic(fmt.Sprintf("gear: %q was given a nil command", owner))
	}
	for _, name := range c.base().names() {
		mustNot(checkName(name))
		if g.find(name) != nil {
			panic(fmt.Sprintf("gear: %q has two commands named %q", owner, name))
		}
	}
	g.children = append(g.children, c)
}

func (g *group) find(name string) Command {
	for _, c := range g.children {
		if slices.Contains(c.base().names(), name) {
			return c
		}
	}
	return nil
}

func (g *group) visible() []Command {
	var out []Command
	for _, c := range g.children {
		if c.base().listed() {
			out = append(out, c)
		}
	}
	return out
}

func (g *group) check(path string, global []Flag) error {
	seen := map[string]bool{}
	for _, c := range g.children {
		for _, name := range c.base().names() {
			if err := checkName(name); err != nil {
				return fmt.Errorf("command %q: %w", path, err)
			}
			if seen[name] {
				return fmt.Errorf("command %q has two commands named %q", path, name)
			}
			seen[name] = true
		}
		if err := c.check(path+" "+c.Label(), global); err != nil {
			return err
		}
	}
	return nil
}

// dispatch reads the flags in front of the first word of args, then runs the
// child that word names. self is the group's own command, for help and
// errors. fallback, if not nil, runs when args name no child.
func (g *group) dispatch(inv *invocation, self Command, args []string, v *ValidatedArgs, fallback *ExecutableCommand) error {
	var own []Flag
	var exclusive [][]string
	if fallback != nil {
		own, exclusive = fallback.flags, fallback.exclusive
	}
	fs := mustFlagSet(own, inv.global, exclusive)
	rest, err := fs.parse(args, v, false)
	if errors.Is(err, ErrHelp) {
		inv.printHelp(self)
		return ErrHelp
	}
	if err != nil {
		return inv.usageError(self, err)
	}

	if len(rest) == 0 {
		if fallback != nil {
			return fallback.execute(inv, nil, v, self)
		}
		inv.printHelp(self)
		return ErrHelp
	}

	name := rest[0]
	if child := g.find(name); child != nil {
		childInv := inv.sub(child.Label())
		for flagName := range v.changed {
			if !fs.global[flagName] {
				return inv.usageError(self, fmt.Errorf("--%s can't be used with %q", flagName, childInv.pathString()))
			}
		}
		if msg := child.base().deprecated; msg != "" {
			fmt.Fprintf(inv.err, "Command %q is deprecated: %s\n", childInv.pathString(), msg)
		}
		return child.run(childInv, rest[1:], v)
	}
	if name == "help" {
		return g.help(inv, self, rest[1:])
	}
	if fallback != nil && len(fallback.args) > 0 {
		return fallback.execute(inv, rest, v, self)
	}
	return inv.usageError(self, g.unknown(inv, name))
}

// help prints the help of the command at path below self, for "app help a b".
func (g *group) help(inv *invocation, self Command, path []string) error {
	cmd, cmdInv := self, inv
	for _, name := range path {
		var child Command
		if sub := cmd.subcommands(); sub != nil {
			child = sub.find(name)
		}
		if child == nil {
			var err error = fmt.Errorf("%q has no subcommands", cmdInv.pathString())
			if sub := cmd.subcommands(); sub != nil {
				err = sub.unknown(cmdInv, name)
			}
			return cmdInv.usageError(cmd, err)
		}
		cmd, cmdInv = child, cmdInv.sub(child.Label())
	}
	cmdInv.printHelp(cmd)
	return ErrHelp
}

func (g *group) unknown(inv *invocation, name string) error {
	var names []string
	for _, c := range g.visible() {
		names = append(names, c.base().names()...)
	}
	msg := fmt.Sprintf("unknown command %q for %q", name, inv.pathString())
	if s := suggest(name, names); len(s) > 0 {
		quoted := make([]string, len(s))
		for i, n := range s {
			quoted[i] = fmt.Sprintf("%q", n)
		}
		msg += fmt.Sprintf(" (did you mean %s?)", strings.Join(quoted, " or "))
	}
	return errors.New(msg)
}
