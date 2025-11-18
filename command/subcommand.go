package command

import (
	"fmt"
)

type Subcommand struct {
	*baseCommand

	children    map[string]Command
	globalFlags []Flag
}

func NewSubcommand(label, description string) *Subcommand {
	return &Subcommand{
		baseCommand: newBaseCommand(label, description),
		children:    make(map[string]Command),
	}
}

func (c *Subcommand) AddChild(command Command) *Subcommand {
	c.children[command.Label()] = command
	return c
}

func (c *Subcommand) run(args []string) error {
	if len(args) < 1 {
		c.PrintHelp()
		return nil
	}

	commandName := args[0]
	childCommand, exists := c.children[commandName]
	if !exists {
		return fmt.Errorf("unknown subcommand: %s", commandName)
	}

	childCommand.inheritGlobalFlags(c.globalFlags)
	return childCommand.run(args[1:])
}

func (c *Subcommand) PrintHelp() {
	printer := newHelpPrinter()
	printer.PrintSubcommandHelp(c)
}

func (c *Subcommand) inheritGlobalFlags(flags []Flag) {
	c.globalFlags = append(c.globalFlags, flags...)
}
