package command

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// ExecutableCommand is a command that runs a handler.
type ExecutableCommand struct {
	*baseCommand

	handler   HandlerFunc
	args      []Arg
	flags     []Flag
	exclusive [][]string
}

func defaultHandler(ctx *Context, _ ValidatedArgs) error {
	return fmt.Errorf("no handler defined for command %q", ctx.Path())
}

// NewExecutableCommand creates a command that runs a handler, set with
// Handler.
func NewExecutableCommand(label, description string) *ExecutableCommand {
	return &ExecutableCommand{
		baseCommand: newBaseCommand(label, description),
		handler:     defaultHandler,
	}
}

// Handler sets the function the command runs.
func (c *ExecutableCommand) Handler(handler HandlerFunc) *ExecutableCommand {
	c.handler = handler
	return c
}

// Args adds positional arguments, in order. It panics if they break the
// rules: one variadic argument at most, and only last, and no required
// argument after an optional one.
func (c *ExecutableCommand) Args(args ...Arg) *ExecutableCommand {
	c.args = append(c.args, args...)
	if err := checkArgs(c.args); err != nil {
		mustNot(fmt.Errorf("command %q: %w", c.label, err))
	}
	return c
}

// Flags adds flags. It panics if two flags share a name or shorthand.
func (c *ExecutableCommand) Flags(flags ...Flag) *ExecutableCommand {
	c.flags = append(c.flags, flags...)
	if _, err := newFlagSet(c.flags, nil, nil); err != nil {
		mustNot(fmt.Errorf("command %q: %w", c.label, err))
	}
	return c
}

// MutuallyExclusive makes it an error to set more than one of the named
// flags. Call it once for each such group.
func (c *ExecutableCommand) MutuallyExclusive(names ...string) *ExecutableCommand {
	c.exclusive = append(c.exclusive, slices.Clone(names))
	return c
}

// Long sets a longer description, shown at the top of the command's own help
// in place of the one-line description. Line breaks are kept, and lines are
// wrapped to fit the terminal.
func (c *ExecutableCommand) Long(text string) *ExecutableCommand {
	c.long = text
	return c
}

// Example adds examples to the command's help, shown as written. An example
// can span several lines, for instance a comment and then the command.
func (c *ExecutableCommand) Example(examples ...string) *ExecutableCommand {
	c.examples = append(c.examples, examples...)
	return c
}

// Group puts the command under a titled section in its parent's help, such as
// "Setup commands". Sections appear in the order their first command was
// added; ungrouped commands come last.
func (c *ExecutableCommand) Group(title string) *ExecutableCommand {
	c.groupTitle = title
	return c
}

// Aliases adds other names the command can be run by.
func (c *ExecutableCommand) Aliases(aliases ...string) *ExecutableCommand {
	c.aliases = append(c.aliases, aliases...)
	return c
}

// Hidden leaves the command out of help, suggestions and completions. It
// still runs.
func (c *ExecutableCommand) Hidden() *ExecutableCommand {
	c.hidden = true
	return c
}

// Deprecated hides the command like Hidden, and prints message as a warning
// whenever it runs.
func (c *ExecutableCommand) Deprecated(message string) *ExecutableCommand {
	c.deprecated = message
	return c
}

func (c *ExecutableCommand) subcommands() *group { return nil }

func (c *ExecutableCommand) check(path string, global []Flag) error {
	if err := checkArgs(c.args); err != nil {
		return fmt.Errorf("command %q: %w", path, err)
	}
	fs, err := newFlagSet(c.flags, global, c.exclusive)
	if err == nil {
		err = fs.checkRefs()
	}
	if err != nil {
		return fmt.Errorf("command %q: %w", path, err)
	}
	return nil
}

func (c *ExecutableCommand) run(inv *invocation, args []string, v *ValidatedArgs) error {
	return c.execute(inv, args, v, c)
}

// execute parses args and runs the handler. page is the command whose help
// and usage are shown, which is the root when c is the root's own handler.
func (c *ExecutableCommand) execute(inv *invocation, args []string, v *ValidatedArgs, page Command) error {
	fs := mustFlagSet(c.flags, inv.global, c.exclusive)
	positional, err := fs.parse(args, v, true)
	if errors.Is(err, ErrHelp) {
		inv.printHelp(page)
		return ErrHelp
	}
	if err == nil {
		err = fs.finish(v)
	}
	if err == nil {
		err = c.bindArgs(positional, v)
	}
	if err != nil {
		return inv.usageError(page, err)
	}

	err = c.handler(newContext(inv, page), *v)
	var usage *UsageError
	if errors.As(err, &usage) && usage.Command == "" {
		usage.Command = inv.pathString()
	}
	return err
}

func (c *ExecutableCommand) bindArgs(raw []string, v *ValidatedArgs) error {
	for i, a := range c.args {
		spec := a.toArg()
		if spec.variadic {
			rest := raw[min(i, len(raw)):]
			if len(rest) == 0 {
				if !spec.optional {
					return fmt.Errorf("missing argument <%s>", spec.usageName())
				}
				return nil
			}
			values := make([]any, len(rest))
			for j, r := range rest {
				value, err := spec.parse(r)
				if err != nil {
					return err
				}
				values[j] = value
			}
			v.args[spec.label] = typedSlice(spec.expected, values)
			return nil
		}
		if i >= len(raw) {
			if !spec.optional {
				return fmt.Errorf("missing argument <%s>", spec.label)
			}
			continue
		}
		value, err := spec.parse(raw[i])
		if err != nil {
			return err
		}
		v.args[spec.label] = value
	}
	if len(raw) > len(c.args) {
		return fmt.Errorf("unexpected argument %q", raw[len(c.args)])
	}
	return nil
}

func (c *ExecutableCommand) usage(inv *invocation) string {
	parts := []string{inv.pathString()}
	for _, a := range c.args {
		parts = append(parts, a.toArg().inUsageLine())
	}
	return strings.Join(append(parts, "[flags]"), " ")
}

func (c *ExecutableCommand) help(inv *invocation) Help {
	return buildHelp(inv, c.baseCommand, []string{c.usage(inv)}, nil, c.args, c.flags)
}
