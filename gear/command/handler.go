package gear

type handler func(ctx *Context, args ValidatedArgs) error
