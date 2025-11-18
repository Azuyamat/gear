package gear

type BaseCommand struct {
	label       string
	description string
}

func (c *BaseCommand) Label() string {
	return c.label
}

func (c *BaseCommand) Description() string {
	return c.description
}

func NewBaseCommand(label string, description string) *BaseCommand {
	return &BaseCommand{
		label:       label,
		description: description,
	}
}
