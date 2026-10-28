// Command greet is the smallest useful gear program: one command, one
// argument, one flag.
//
//	go run ./examples/greet greet Ada --shout
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/azuyamat/gear/command"
)

func main() {
	root := command.NewRootCommand("greet", "Greet people").
		AddChild(command.NewExecutableCommand("greet", "Greet someone by name").
			Args(command.NewStringArg("name", "Who to greet")).
			Flags(command.NewBoolFlag("shout", "s", "Greet loudly", false)).
			Handler(func(ctx *command.Context, args command.ValidatedArgs) error {
				msg := "Hello, " + command.ArgValue[string](args, "name") + "!"
				if command.FlagValue[bool](args, "shout") {
					msg = strings.ToUpper(msg)
				}
				fmt.Fprintln(ctx.Out(), msg)
				return nil
			}))

	if err := root.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(command.ExitCode(err))
	}
}
