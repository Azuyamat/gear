package command

type handler func(ctx *Context, args ValidatedArgs) error
