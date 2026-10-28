package command

import (
	stdcontext "context"
	"io"
)

// Context is what a handler knows about the run it's part of.
type Context struct {
	inv     *invocation
	context stdcontext.Context
	command Command
	path    string
	out     io.Writer
	err     io.Writer
}

func newContext(inv *invocation, command Command) *Context {
	return &Context{
		inv:     inv,
		context: inv.ctx,
		command: command,
		path:    inv.pathString(),
		out:     inv.out,
		err:     inv.err,
	}
}

// Context returns the context passed to RunContext, which is cancelled when
// the caller cancels it.
func (c *Context) Context() stdcontext.Context { return c.context }

// Command returns the command being run.
func (c *Context) Command() Command { return c.command }

// Path returns the full name the command was run by, such as "app db migrate".
func (c *Context) Path() string { return c.path }

// Help returns the contents of the command's help page, for drawing with
// WriteHelp or a renderer of your own.
func (c *Context) Help() Help { return c.command.help(c.inv) }

// PrintHelp draws the command's help on Out, the way --help would.
func (c *Context) PrintHelp() { c.inv.printHelp(c.command) }

// Out returns the writer for normal output, set with RootCommand.Output.
func (c *Context) Out() io.Writer { return c.out }

// Err returns the writer for errors and warnings, set with RootCommand.Output.
func (c *Context) Err() io.Writer { return c.err }
