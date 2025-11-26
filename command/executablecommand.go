package command

import (
	stdcontext "context"
	"fmt"
)

type executableCommand struct {
	*baseCommand

	handler            handler
	args               []Arg
	flags              []Flag
	cachedValidatedArgs *ValidatedArgs
}

var defaultExecutableCommandHandler = func(ctx *Context, args ValidatedArgs) error {
	return fmt.Errorf("no handler defined for command: %s", ctx.command.Label())
}

func NewExecutableCommand(label, description string) *executableCommand {
	return &executableCommand{
		baseCommand: newBaseCommand(label, description),
		handler:     defaultExecutableCommandHandler,
		args:        []Arg{},
		flags:       []Flag{},
	}
}

func (c *executableCommand) Handler(handler handler) *executableCommand {
	c.handler = handler
	return c
}

func (c *executableCommand) Args(args ...Arg) *executableCommand {
	c.args = args
	return c
}

func (c *executableCommand) Flags(flags ...Flag) *executableCommand {
	c.flags = flags
	return c
}

func (c *executableCommand) execute(ctx *Context, args ValidatedArgs) error {
	return c.handler(ctx, args)
}

func (c *executableCommand) run(args []string) error {
	ctx := newContext(stdcontext.Background(), c)
	positionalArgs, err := c.separateFlagsFromArgs(args)
	if err != nil {
		return err
	}
	validatedArgs, err := c.parseAndValidateArgs(positionalArgs)
	if err != nil {
		return err
	}
	return c.execute(ctx, *validatedArgs)
}


type flagMaps struct {
	byName      map[string]Flag
	byShorthand map[string]Flag
}

func (c *executableCommand) buildFlagMaps() flagMaps {
	flagMap := make(map[string]Flag)
	shorthandMap := make(map[string]Flag)

	for i := range c.flags {
		f := c.flags[i]
		flagMap[f.Name()] = f
		if f.Shorthand() != "" {
			shorthandMap[f.Shorthand()] = f
		}
	}

	return flagMaps{byName: flagMap, byShorthand: shorthandMap}
}

func (c *executableCommand) setDefaultFlagValues(validatedArgs *ValidatedArgs) {
	for _, f := range c.flags {
		if f.DefaultValue() != nil {
			validatedArgs.setFlag(f.Name(), f.DefaultValue())
		}
	}
}

func splitFlagNameValue(flagStr string) (name string, value string, hasValue bool) {
	for i, ch := range flagStr {
		if ch == '=' {
			return flagStr[:i], flagStr[i+1:], true
		}
	}
	return flagStr, "", false
}

func (c *executableCommand) parseSingleFlag(f Flag, value string, hasExplicitValue bool, args []string, currentIndex int) (parsedValue interface{}, newIndex int, err error) {
	flagValue := value
	nextIndex := currentIndex

	if f.Expected() == ValueTypeBool && !hasExplicitValue {
		flagValue = ""
	} else if !hasExplicitValue {
		if currentIndex+1 >= len(args) {
			return nil, currentIndex, fmt.Errorf("flag --%s requires a value", f.Name())
		}
		nextIndex++
		flagValue = args[nextIndex]
	}

	parsedValue, err = parseValue(flagValue, f.Expected())
	if err != nil {
		return nil, currentIndex, fmt.Errorf("invalid value for flag '%s': %v", f.Name(), err)
	}

	if err := f.validate(parsedValue); err != nil {
		return nil, currentIndex, fmt.Errorf("validation failed for flag '%s': %v", f.Name(), err)
	}

	return parsedValue, nextIndex, nil
}

func (c *executableCommand) separateFlagsFromArgs(args []string) ([]string, error) {
	validatedArgs := newValidatedArgs()
	positionalArgs := []string{}

	maps := c.buildFlagMaps()
	c.setDefaultFlagValues(validatedArgs)

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--" {
			positionalArgs = append(positionalArgs, args[i+1:]...)
			break
		}

		if len(arg) > 2 && arg[0:2] == "--" {
			flagName, flagValue, hasValue := splitFlagNameValue(arg[2:])

			if flagName == "help" {
				c.PrintHelp()
				return nil, fmt.Errorf("")
			}

			f, ok := maps.byName[flagName]
			if !ok {
				return nil, fmt.Errorf("unknown flag: --%s", flagName)
			}

			parsedValue, newIndex, err := c.parseSingleFlag(f, flagValue, hasValue, args, i)
			if err != nil {
				return nil, err
			}

			validatedArgs.setFlag(f.Name(), parsedValue)
			i = newIndex
			continue
		}

		if len(arg) > 1 && arg[0] == '-' && arg[1] != '-' {
			shorthand, flagValue, hasValue := splitFlagNameValue(arg[1:])

			if shorthand == "h" {
				c.PrintHelp()
				return nil, fmt.Errorf("")
			}

			f, ok := maps.byShorthand[shorthand]
			if !ok {
				return nil, fmt.Errorf("unknown flag: -%s", shorthand)
			}

			parsedValue, newIndex, err := c.parseSingleFlag(f, flagValue, hasValue, args, i)
			if err != nil {
				return nil, err
			}

			validatedArgs.setFlag(f.Name(), parsedValue)
			i = newIndex
			continue
		}

		positionalArgs = append(positionalArgs, arg)
	}

	c.cachedValidatedArgs = validatedArgs
	return positionalArgs, nil
}

func (c *executableCommand) parseAndValidateArgs(args []string) (*ValidatedArgs, error) {
	requiredCount := 0
	variadicIndex := -1
	for i, arg := range c.args {
		if !arg.IsOptional() {
			requiredCount++
		}
		if arg.IsVariadic() {
			if variadicIndex != -1 {
				return nil, fmt.Errorf("multiple variadic arguments not allowed")
			}
			if i != len(c.args)-1 {
				return nil, fmt.Errorf("variadic argument must be the last argument")
			}
			variadicIndex = i
		}
	}

	if len(args) < requiredCount {
		c.PrintHelp()
		return nil, fmt.Errorf("not enough arguments for command: %s", c.Label())
	}

	if variadicIndex == -1 && len(args) > len(c.args) {
		c.PrintHelp()
		return nil, fmt.Errorf("too many arguments for command: %s", c.Label())
	}

	validatedArgs := c.cachedValidatedArgs
	if validatedArgs == nil {
		validatedArgs = newValidatedArgs()
	}

	for i, arg := range c.args {
		if arg.IsVariadic() {
			variadicValues := []interface{}{}
			for j := i; j < len(args); j++ {
				rawValue := args[j]
				parsedValue, err := parseValue(rawValue, arg.Expected())

				if err != nil {
					return nil, fmt.Errorf("invalid value for variadic argument '%s' at position %d: %v", arg.Label(), j-i, err)
				}

				if err := arg.validate(parsedValue); err != nil {
					return nil, fmt.Errorf("validation failed for variadic argument '%s' at position %d: %v", arg.Label(), j-i, err)
				}

				variadicValues = append(variadicValues, parsedValue)
			}
			validatedArgs.setVariadic(arg.Label(), variadicValues)
			break
		}

		if i >= len(args) {
			if !arg.IsOptional() {
				return nil, fmt.Errorf("missing required argument: %s", arg.Label())
			}
			continue
		}

		rawValue := args[i]
		parsedValue, err := parseValue(rawValue, arg.Expected())

		if err != nil {
			return nil, fmt.Errorf("invalid value for argument '%s': %v", arg.Label(), err)
		}

		if err := arg.validate(parsedValue); err != nil {
			return nil, fmt.Errorf("validation failed for argument '%s': %v", arg.Label(), err)
		}

		validatedArgs.set(arg.Label(), parsedValue)
	}
	return validatedArgs, nil
}

func (c *executableCommand) PrintHelp() {
	printer := newHelpPrinter()
	printer.PrintExecutableCommandHelp(c)
}

func (c *executableCommand) inheritGlobalFlags(flags []Flag) {
	c.flags = append(c.flags, flags...)
}
