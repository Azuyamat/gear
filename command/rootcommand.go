package command

import (
	"fmt"
)

type RootCommand struct {
	*baseCommand

	children    map[string]Command
	globalFlags []Flag
}

func NewRootCommand(label, description string) *RootCommand {
	return &RootCommand{
		baseCommand: newBaseCommand(label, description),
		children:    make(map[string]Command),
		globalFlags: []Flag{},
	}
}

func (c *RootCommand) AddChild(command Command) *RootCommand {
	c.children[command.Label()] = command
	return c
}

func (c *RootCommand) GlobalFlags(flags ...Flag) *RootCommand {
	c.globalFlags = flags
	return c
}

func (c *RootCommand) Run(args []string) error {
	return c.run(args)
}

func (c *RootCommand) run(args []string) error {
	if len(args) < 1 {
		c.PrintHelp()
		return nil
	}

	commandName := args[0]
	childCommand, exists := c.children[commandName]
	if !exists {
		return fmt.Errorf("unknown command: %s", commandName)
	}

	childCommand.inheritGlobalFlags(c.globalFlags)
	return childCommand.run(args[1:])
}

func (c *RootCommand) PrintHelp() {
	printer := newHelpPrinter()
	printer.PrintRootCommandHelp(c)
}

func (c *RootCommand) inheritGlobalFlags(flags []Flag) {
	c.globalFlags = append(c.globalFlags, flags...)
}
