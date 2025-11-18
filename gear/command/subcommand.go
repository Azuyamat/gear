package gear

type Subcommand struct {
	baseCommand *BaseCommand
	parent      *Command

	label       string
	description string
}

func (c *Subcommand) Label() string {
	return c.label
}

func (c *Subcommand) Description() string {
	return c.description
}

func (c *Subcommand) BaseCommand() *BaseCommand {
	return c.baseCommand
}

func (c *Subcommand) Parent() *Command {
	return c.parent
}
