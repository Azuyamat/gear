package command

import (
	stdcontext "context"
	"errors"
	"fmt"
	"io"
	"os"
)

// RootCommand is the top of a command tree: the program itself.
type RootCommand struct {
	*baseCommand
	group

	self        *ExecutableCommand // the root's own handler, if it has one
	globalFlags []Flag
	version     string
	out         io.Writer
	err         io.Writer
	helpStyle   HelpStyle
	helpFunc    func(w io.Writer, h Help)
}

// NewRootCommand creates the top of a command tree. label is the program's
// name, as help shows it.
func NewRootCommand(label, description string) *RootCommand {
	return &RootCommand{
		baseCommand: newBaseCommand(label, description),
		out:         os.Stdout,
		err:         os.Stderr,
	}
}

// AddChild adds a command below the root. It panics if the command's label or
// one of its aliases is already taken.
func (c *RootCommand) AddChild(command Command) *RootCommand {
	c.group.add(c.label, command)
	return c
}

// GlobalFlags adds flags that every command in the tree accepts. They can
// also go before the command name, as in "app --verbose build". It panics if
// two global flags share a name or shorthand.
func (c *RootCommand) GlobalFlags(flags ...Flag) *RootCommand {
	c.globalFlags = append(c.globalFlags, flags...)
	if _, err := newFlagSet(nil, c.globalFlags, nil); err != nil {
		mustNot(fmt.Errorf("command %q: %w", c.label, err))
	}
	return c
}

// Long sets a longer description, shown at the top of the command's own help
// in place of the one-line description. Line breaks are kept, and lines are
// wrapped to fit the terminal.
func (c *RootCommand) Long(text string) *RootCommand {
	c.long = text
	return c
}

// Example adds examples to the command's help, shown as written. An example
// can span several lines, for instance a comment and then the command.
func (c *RootCommand) Example(examples ...string) *RootCommand {
	c.examples = append(c.examples, examples...)
	return c
}

// Version sets the version that --version prints.
func (c *RootCommand) Version(version string) *RootCommand {
	c.version = version
	return c
}

// Output sets where help and handler output go (out, by default os.Stdout)
// and where usage hints and warnings go (err, by default os.Stderr).
func (c *RootCommand) Output(out, err io.Writer) *RootCommand {
	c.out, c.err = out, err
	return c
}

// Handler gives the root a handler of its own, which runs when the command
// line names no child command. Without one, the root prints its help.
func (c *RootCommand) Handler(handler HandlerFunc) *RootCommand {
	c.ensureSelf().Handler(handler)
	return c
}

// Args adds positional arguments to the root's own handler. A first argument
// that matches a child command's name runs that command instead.
func (c *RootCommand) Args(args ...Arg) *RootCommand {
	c.ensureSelf().Args(args...)
	return c
}

// Flags adds flags that only the root's own handler accepts. Use GlobalFlags
// for flags every command accepts.
func (c *RootCommand) Flags(flags ...Flag) *RootCommand {
	c.ensureSelf().Flags(flags...)
	return c
}

// MutuallyExclusive makes it an error to set more than one of the named
// flags of the root's own handler.
func (c *RootCommand) MutuallyExclusive(names ...string) *RootCommand {
	c.ensureSelf().MutuallyExclusive(names...)
	return c
}

func (c *RootCommand) ensureSelf() *ExecutableCommand {
	if c.self == nil {
		c.self = NewExecutableCommand(c.label, c.description)
	}
	return c.self
}

// HelpStyle sets how help is drawn: colours, through hooks that style text,
// and the width to wrap at.
func (c *RootCommand) HelpStyle(style HelpStyle) *RootCommand {
	c.helpStyle = style
	return c
}

// HelpFunc replaces how every help page in the tree is drawn. The function
// gets the page's contents; it can draw them itself, or adjust them and pass
// them to WriteHelp.
func (c *RootCommand) HelpFunc(render func(w io.Writer, h Help)) *RootCommand {
	c.helpFunc = render
	return c
}

// HelpFor returns the help of the command at path below the root, such as
// HelpFor("db", "migrate"), or the root's own with no path.
func (c *RootCommand) HelpFor(path ...string) (Help, error) {
	var cmd Command = c
	inv := c.invocation(stdcontext.Background())
	for _, name := range path {
		var child Command
		if g := cmd.subcommands(); g != nil {
			child = g.find(name)
		}
		if child == nil {
			return Help{}, fmt.Errorf("no command %q under %q", name, inv.pathString())
		}
		cmd, inv = child, inv.sub(child.Label())
	}
	return cmd.help(inv), nil
}

// PrintHelp writes the root's help to its output writer.
func (c *RootCommand) PrintHelp() {
	inv := c.invocation(stdcontext.Background())
	inv.printHelp(c)
}

// Run runs the command line args, which don't include the program name:
// usually os.Args[1:]. It returns nil after printing help or the version.
func (c *RootCommand) Run(args []string) error {
	return c.RunContext(stdcontext.Background(), args)
}

// RunContext is Run with a context, which handlers get from Context.Context.
// Cancel it to stop them, for example on Ctrl-C with signal.NotifyContext.
//
// It panics if the tree is set up wrong, such as a command flag with the same
// name as a global flag.
func (c *RootCommand) RunContext(ctx stdcontext.Context, args []string) error {
	mustNot(c.check(c.label, nil))
	err := c.run(c.invocation(ctx), args, newValidatedArgs())
	if errors.Is(err, ErrHelp) {
		return nil
	}
	return err
}

func (c *RootCommand) invocation(ctx stdcontext.Context) *invocation {
	return &invocation{
		ctx:    ctx,
		root:   c,
		path:   []string{c.label},
		global: c.globalFlags,
		out:    c.out,
		err:    c.err,
	}
}

func (c *RootCommand) check(path string, _ []Flag) error {
	if _, err := newFlagSet(nil, c.globalFlags, nil); err != nil {
		return fmt.Errorf("command %q: %w", path, err)
	}
	for _, f := range c.globalFlags {
		if c.version != "" && f.Name() == "version" {
			return fmt.Errorf("command %q: --version is reserved because the root has a Version", path)
		}
	}
	if c.self != nil {
		if err := c.self.check(path, c.globalFlags); err != nil {
			return err
		}
	}
	return c.group.check(path, c.globalFlags)
}

func (c *RootCommand) run(inv *invocation, args []string, v *ValidatedArgs) error {
	if len(args) > 0 && args[0] == completeCommand {
		c.complete(inv, args[1:])
		return nil
	}
	if c.version != "" && len(args) > 0 && args[0] == "--version" {
		fmt.Fprintf(inv.out, "%s %s\n", c.label, c.version)
		return nil
	}
	return c.group.dispatch(inv, c, args, v, c.self)
}

func (c *RootCommand) usage(inv *invocation) string {
	if len(c.children) == 0 && c.self != nil {
		return c.self.usage(inv)
	}
	return inv.pathString() + " <command> [flags]"
}

func (c *RootCommand) help(inv *invocation) Help {
	usage := []string{c.usage(inv)}
	var args []Arg
	var flags []Flag
	if c.self != nil {
		if self := c.self.usage(inv); self != usage[0] {
			usage = append(usage, self)
		}
		args, flags = c.self.args, c.self.flags
	}
	h := buildHelp(inv, c.baseCommand, usage, c.children, args, flags)
	if c.version != "" {
		h.Flags = append(h.Flags, HelpFlag{Name: "version", Description: "Show the version"})
	}
	return h
}
