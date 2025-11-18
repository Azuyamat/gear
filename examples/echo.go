package main

import (
	"fmt"
	"os"

	gear "github.com/azuyamat/gear/gear/command"
)

var CommandName = "echo"
var CommandDescription = "Echoes the input text"

var EchoCommand = gear.NewRootCommand(CommandName, CommandDescription).
	AddChild(
		gear.NewExecutableCommand("say", "Say something with echo").
			Args(
				gear.NewStringArg("text", "Text to echo").
					ExtendValidators(
						func(value interface{}) error {
							str, ok := value.(string)
							if !ok {
								return fmt.Errorf("expected string, got %T", value)
							}
							if len(str) < 5 || str[:5] != "Hello" {
								return fmt.Errorf("text must start with 'Hello'")
							}
							return nil
						},
					),
			).
			Handler(func(ctx *gear.Context, args gear.ValidatedArgs) error {
				text := args.String("text")
				fmt.Println(text)
				return nil
			}),
	)

func main() {
	args := os.Args[1:]
	if err := EchoCommand.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
