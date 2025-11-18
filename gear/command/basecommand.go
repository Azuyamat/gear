package gear

type baseCommand struct {
	label       string
	description string
}

func (c *baseCommand) Label() string {
	return c.label
}

func (c *baseCommand) Description() string {
	return c.description
}

func newBaseCommand(label, description string) *baseCommand {
	return &baseCommand{
		label:       label,
		description: description,
	}
}
