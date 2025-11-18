package command

type Command interface {
	Label() string
	Description() string
	run(args []string) error
	PrintHelp()
}
