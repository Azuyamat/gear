package gear

type Command interface {
	Label() string
	Description() string
	run(args []string) error
}
