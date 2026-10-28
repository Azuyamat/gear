package command

// HandlerFunc runs an executable command. Return a *UsageError, for example
// from UsageErrorf, to report bad input that gear doesn't check itself.
type HandlerFunc func(ctx *Context, args ValidatedArgs) error
