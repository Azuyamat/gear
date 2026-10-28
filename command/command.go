package command

import (
	stdcontext "context"
	"fmt"
	"io"
	"strings"
)

// Command is a node in a command tree: a *RootCommand, *Subcommand or
// *ExecutableCommand.
//
// Command has unexported methods, so it can't be implemented outside this
// package. Build commands from the three types above instead.
type Command interface {
	Label() string
	Description() string

	base() *baseCommand
	subcommands() *group
	usage(inv *invocation) string
	help(inv *invocation) Help
	check(path string, global []Flag) error
	run(inv *invocation, args []string, v *ValidatedArgs) error
}

type baseCommand struct {
	label       string
	description string
	long        string
	examples    []string
	groupTitle  string
	aliases     []string
	hidden      bool
	deprecated  string
}

func newBaseCommand(label, description string) *baseCommand {
	return &baseCommand{label: label, description: description}
}

// Label returns the name the command is run by.
func (c *baseCommand) Label() string { return c.label }

// Description returns the one-line summary shown in help.
func (c *baseCommand) Description() string { return c.description }

func (c *baseCommand) base() *baseCommand { return c }

// names returns the label followed by the aliases.
func (c *baseCommand) names() []string {
	return append([]string{c.label}, c.aliases...)
}

// listed reports whether the command shows up in help, suggestions and
// completions.
func (c *baseCommand) listed() bool {
	return !c.hidden && c.deprecated == ""
}

func checkName(name string) error {
	switch {
	case name == "":
		return fmt.Errorf("a command has no label")
	case strings.HasPrefix(name, "-"):
		return fmt.Errorf("command name %q can't start with '-'", name)
	case name == completeCommand:
		return fmt.Errorf("command name %q is reserved for completion", name)
	case strings.ContainsAny(name, " \t\n'\"`$\\"):
		return fmt.Errorf("command name %q can't contain spaces or quotes", name)
	}
	return nil
}

func mustNot(err error) {
	if err != nil {
		panic("gear: " + err.Error())
	}
}

// invocation is what a run passes down the tree. Commands keep nothing from
// one run to the next, so a tree can be run again, or from several goroutines
// at once.
type invocation struct {
	ctx    stdcontext.Context
	root   *RootCommand
	path   []string
	global []Flag
	out    io.Writer
	err    io.Writer
}

func (inv *invocation) sub(label string) *invocation {
	next := *inv
	next.path = append(inv.path[:len(inv.path):len(inv.path)], label)
	return &next
}

func (inv *invocation) pathString() string {
	return strings.Join(inv.path, " ")
}

// printHelp draws cmd's help on the output writer, with the root's HelpFunc
// or HelpStyle.
func (inv *invocation) printHelp(cmd Command) {
	h := cmd.help(inv)
	if inv.root.helpFunc != nil {
		inv.root.helpFunc(inv.out, h)
		return
	}
	WriteHelp(inv.out, h, inv.root.helpStyle)
}

// usageError points the user at help and wraps err as a *UsageError.
func (inv *invocation) usageError(cmd Command, err error) error {
	fmt.Fprintf(inv.err, "Usage: %s\nRun '%s --help' for more.\n", cmd.usage(inv), inv.pathString())
	return &UsageError{Command: inv.pathString(), Err: err}
}
