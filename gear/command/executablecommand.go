package gear

import (
	stdcontext "context"
	"fmt"
	"strconv"
)

type executableCommand struct {
	*baseCommand

	handler handler
	args    []arg
}

var defaultExecutableCommandHandler = func(ctx *Context, args ValidatedArgs) error {
	return fmt.Errorf("no handler defined for command: %s", ctx.command.Label())
}

func NewExecutableCommand(label, description string) *executableCommand {
	return &executableCommand{
		baseCommand: newBaseCommand(label, description),
		handler:     defaultExecutableCommandHandler,
		args:        []arg{},
	}
}

func (c *executableCommand) Handler(handler handler) *executableCommand {
	c.handler = handler
	return c
}

func (c *executableCommand) Args(args ...arg) *executableCommand {
	c.args = args
	return c
}

func (c *executableCommand) execute(ctx *Context, args ValidatedArgs) error {
	return c.handler(ctx, args)
}

func (c *executableCommand) run(args []string) error {
	ctx := newContext(stdcontext.Background(), c)
	validatedArgs, err := c.parseAndValidateArgs(args)
	if err != nil {
		return err
	}
	return c.execute(ctx, *validatedArgs)
}

func parseArgValue(value string, expectedType ArgType) (interface{}, error) {
	switch expectedType {
	case ArgTypeString:
		return value, nil
	case ArgTypeInt:
		return strconv.Atoi(value)
	case ArgTypeFloat:
		return strconv.ParseFloat(value, 64)
	case ArgTypeBool:
		return strconv.ParseBool(value)
	default:
		return nil, fmt.Errorf("unsupported argument type: %s", expectedType)
	}
}

func (c *executableCommand) parseAndValidateArgs(args []string) (*ValidatedArgs, error) {
	requiredCount := 0
	for _, arg := range c.args {
		if !arg.IsOptional() {
			requiredCount++
		}
	}

	if len(args) < requiredCount {
		c.PrintHelp()
		return nil, fmt.Errorf("not enough arguments for command: %s", c.Label())
	}

	if len(args) > len(c.args) {
		c.PrintHelp()
		return nil, fmt.Errorf("too many arguments for command: %s", c.Label())
	}

	validatedArgs := newValidatedArgs()
	for i, arg := range c.args {
		if i >= len(args) {
			if !arg.IsOptional() {
				return nil, fmt.Errorf("missing required argument: %s", arg.Label())
			}
			continue
		}

		rawValue := args[i]
		parsedValue, err := parseArgValue(rawValue, arg.Expected())

		if err != nil {
			return nil, fmt.Errorf("invalid value for argument '%s': %v", arg.Label(), err)
		}

		if err := arg.validate(parsedValue); err != nil {
			return nil, fmt.Errorf("validation failed for argument '%s': %v", arg.Label(), err)
		}

		validatedArgs.args[arg.Label()] = parsedValue
	}
	return validatedArgs, nil
}

func (c *executableCommand) PrintHelp() {
	printer := newHelpPrinter()
	printer.PrintExecutableCommandHelp(c)
}
