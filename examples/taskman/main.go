// Command taskman shows most of what gear does: subcommands, aliases, command
// groups, examples, global flags, validators, choices, environment variables,
// mutually exclusive flags, cancellation, styled help and shell completion.
//
//	go run ./examples/taskman --help
//	go run ./examples/taskman add "Write docs" 3 --tag docs --tag writing
//	go run ./examples/taskman ls -v --format json
//	go run ./examples/taskman db migrate --steps -1
//	source <(go run ./examples/taskman completion bash)
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/azuyamat/gear/command"
)

func main() {
	root := command.NewRootCommand("taskman", "Manage tasks from the terminal").
		Version("0.1.0").
		GlobalFlags(
			command.NewBoolFlag("verbose", "v", "Print more detail", false),
			command.NewStringFlag("store", "", "Where tasks are kept", "tasks.json").Env("TASKMAN_STORE"),
		)
	if isTerminal(os.Stdout) {
		bold := func(s string) string { return "\x1b[1m" + s + "\x1b[0m" }
		dim := func(s string) string { return "\x1b[2m" + s + "\x1b[0m" }
		root.HelpStyle(command.HelpStyle{Heading: bold, Command: bold, Flag: bold, Placeholder: dim, Note: dim})
	}

	root.
		AddChild(add()).
		AddChild(list()).
		AddChild(command.NewSubcommand("db", "Manage the task store").
			Group("Setup").
			AddChild(migrate())).
		AddChild(root.CompletionCommand().Group("Setup"))

	// Ctrl-C cancels ctx, which handlers get from ctx.Context().
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := root.RunContext(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(command.ExitCode(err))
	}
}

func add() *command.ExecutableCommand {
	return command.NewExecutableCommand("add", "Add a task").
		Group("Tasks").
		Long("Add a task to the list. Tasks start unfinished, with priority 3 unless you give one. Tag a task to find it again with 'taskman ls <tag>'.").
		Example(
			`taskman add "Write docs"`,
			"# high priority, two tags and a reminder\ntaskman add \"Fix login\" 5 -t bug -t urgent --remind 2h",
		).
		Args(
			command.NewStringArg("name", "What to do"),
			command.NewIntArg("priority", "From 1 (low) to 5 (high)").
				AsOptional().
				ExtendValidators(func(p int) error {
					if p < 1 || p > 5 {
						return errors.New("must be between 1 and 5")
					}
					return nil
				}),
		).
		Flags(
			command.NewStringsFlag("tag", "t", "Tag the task", nil),
			command.NewDurationFlag("remind", "r", "Remind me after this long", 0),
		).
		Handler(func(ctx *command.Context, args command.ValidatedArgs) error {
			priority := 3
			if args.Has("priority") {
				priority = command.ArgValue[int](args, "priority")
			}
			fmt.Fprintf(ctx.Out(), "Added %q (priority %d)", command.ArgValue[string](args, "name"), priority)
			if tags := command.FlagValue[[]string](args, "tag"); len(tags) > 0 {
				fmt.Fprintf(ctx.Out(), " tagged %s", strings.Join(tags, ", "))
			}
			if d := command.FlagValue[time.Duration](args, "remind"); d > 0 {
				fmt.Fprintf(ctx.Out(), ", reminder in %s", d)
			}
			fmt.Fprintln(ctx.Out())
			return nil
		})
}

func list() *command.ExecutableCommand {
	return command.NewExecutableCommand("ls", "List tasks").
		Group("Tasks").
		Aliases("list").
		Args(command.NewStringArg("tags", "Only list tasks with these tags").
			AsVariadic().
			AsOptional().
			Complete(func(ctx *command.Context, args command.ValidatedArgs, prefix string) []string {
				// A real program would read these from its store.
				return []string{"bug\tThings that are broken", "docs", "urgent"}
			})).
		Flags(
			command.NewStringFlag("format", "f", "Output format", "text").Choices("text", "json"),
			command.NewBoolFlag("done", "d", "Only list finished tasks", false),
			command.NewBoolFlag("todo", "", "Only list unfinished tasks", false),
		).
		MutuallyExclusive("done", "todo").
		Handler(func(ctx *command.Context, args command.ValidatedArgs) error {
			if command.FlagValue[bool](args, "verbose") {
				fmt.Fprintln(ctx.Err(), "reading", command.FlagValue[string](args, "store"))
			}
			tags := command.ArgValue[[]string](args, "tags")
			if command.FlagValue[string](args, "format") == "json" {
				fmt.Fprintf(ctx.Out(), `{"tags": %q, "tasks": []}`+"\n", tags)
				return nil
			}
			fmt.Fprintln(ctx.Out(), "No tasks yet.")
			return nil
		})
}

func migrate() *command.ExecutableCommand {
	return command.NewExecutableCommand("migrate", "Upgrade the task store's format").
		Flags(command.NewIntFlag("steps", "n", "How many versions to move; negative to go back", 0)).
		Handler(func(ctx *command.Context, args command.ValidatedArgs) error {
			select {
			case <-ctx.Context().Done():
				return ctx.Context().Err()
			case <-time.After(100 * time.Millisecond):
			}
			if !args.Changed("steps") {
				fmt.Fprintln(ctx.Out(), "Migrated to the latest version.")
				return nil
			}
			fmt.Fprintf(ctx.Out(), "Migrated %d steps.\n", command.FlagValue[int](args, "steps"))
			return nil
		})
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
