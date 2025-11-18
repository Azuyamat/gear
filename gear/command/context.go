package command

import stdcontext "context"

type Context struct {
	context stdcontext.Context

	command Command
}

func newContext(ctx stdcontext.Context, command Command) *Context {
	return &Context{
		context: ctx,
		command: command,
	}
}

func (c *Context) Context() stdcontext.Context {
	return c.context
}

func (c *Context) Command() Command {
	return c.command
}
