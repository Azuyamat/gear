// Package command builds command-line interfaces from a tree of commands.
//
// A tree has one *RootCommand at the top. Below it, a *Subcommand groups
// related commands, and an *ExecutableCommand runs a handler:
//
//	root := command.NewRootCommand("app", "An example").
//		Version("1.0.0").
//		AddChild(command.NewExecutableCommand("greet", "Greet someone").
//			Args(command.NewStringArg("name", "Who to greet")).
//			Flags(command.NewBoolFlag("shout", "s", "Greet loudly", false)).
//			Handler(func(ctx *command.Context, args command.ValidatedArgs) error {
//				fmt.Fprintln(ctx.Out(), "Hello,", command.ArgValue[string](args, "name"))
//				return nil
//			}))
//
//	if err := root.RunContext(ctx, os.Args[1:]); err != nil {
//		fmt.Fprintln(os.Stderr, "Error:", err)
//		os.Exit(command.ExitCode(err))
//	}
//
// Every command understands --help and -h (unless a flag claims -h), and
// groups also understand "help <command>". Bad input returns a *UsageError,
// so a program can tell it apart from a handler failing.
//
// Mistakes in how the tree is set up, such as two flags with the same name or
// a required argument after an optional one, panic: as soon as the builder
// method sees them if it can, or otherwise at the start of the first run.
package command
