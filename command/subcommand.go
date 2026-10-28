package command

// Subcommand groups related commands under one name, as in "app db migrate".
type Subcommand struct {
	*baseCommand
	group
}

// NewSubcommand creates a command that groups the commands added with
// AddChild.
func NewSubcommand(label, description string) *Subcommand {
	return &Subcommand{baseCommand: newBaseCommand(label, description)}
}

// AddChild adds a command below this one. It panics if the command's label or
// one of its aliases is already taken.
func (c *Subcommand) AddChild(command Command) *Subcommand {
	c.group.add(c.label, command)
	return c
}

// Long sets a longer description, shown at the top of the command's own help
// in place of the one-line description. Line breaks are kept, and lines are
// wrapped to fit the terminal.
func (c *Subcommand) Long(text string) *Subcommand {
	c.long = text
	return c
}

// Example adds examples to the command's help, shown as written. An example
// can span several lines, for instance a comment and then the command.
func (c *Subcommand) Example(examples ...string) *Subcommand {
	c.examples = append(c.examples, examples...)
	return c
}

// Group puts the command under a titled section in its parent's help, such as
// "Setup commands". Sections appear in the order their first command was
// added; ungrouped commands come last.
func (c *Subcommand) Group(title string) *Subcommand {
	c.groupTitle = title
	return c
}

// Aliases adds other names the command can be run by.
func (c *Subcommand) Aliases(aliases ...string) *Subcommand {
	c.aliases = append(c.aliases, aliases...)
	return c
}

// Hidden leaves the command out of help, suggestions and completions. It
// still runs.
func (c *Subcommand) Hidden() *Subcommand {
	c.hidden = true
	return c
}

// Deprecated hides the command like Hidden, and prints message as a warning
// whenever it runs.
func (c *Subcommand) Deprecated(message string) *Subcommand {
	c.deprecated = message
	return c
}

func (c *Subcommand) check(path string, global []Flag) error {
	return c.group.check(path, global)
}

func (c *Subcommand) run(inv *invocation, args []string, v *ValidatedArgs) error {
	return c.group.dispatch(inv, c, args, v, nil)
}

func (c *Subcommand) usage(inv *invocation) string {
	return inv.pathString() + " <command> [flags]"
}

func (c *Subcommand) help(inv *invocation) Help {
	return buildHelp(inv, c.baseCommand, []string{c.usage(inv)}, c.children, nil, nil)
}
