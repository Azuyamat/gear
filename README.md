# Gear

A small, typed command-line framework for Go with no dependencies outside the standard library.

```bash
go get github.com/azuyamat/gear
```

```go
root := command.NewRootCommand("greet", "Greet people").
    AddChild(command.NewExecutableCommand("greet", "Greet someone by name").
        Args(command.NewStringArg("name", "Who to greet")).
        Flags(command.NewBoolFlag("shout", "s", "Greet loudly", false)).
        Handler(func(ctx *command.Context, args command.ValidatedArgs) error {
            fmt.Fprintln(ctx.Out(), "Hello,", command.ArgValue[string](args, "name"))
            return nil
        }))

if err := root.Run(os.Args[1:]); err != nil {
    fmt.Fprintln(os.Stderr, "Error:", err)
    os.Exit(command.ExitCode(err)) // 2 for bad input, 1 for other failures
}
```

Runnable programs live in [`examples/`](examples):

- [`greet`](examples/greet/main.go): the smallest useful program.
- [`taskman`](examples/taskman/main.go): subcommands, aliases, command groups, examples, global flags, validators, choices, environment variables, mutually exclusive flags, Ctrl-C cancellation, styled help and shell completion.

Reference documentation is on [pkg.go.dev](https://pkg.go.dev/github.com/azuyamat/gear/command).

## Commands

A tree has one `RootCommand`. A `Subcommand` groups commands (`app db migrate`), and an `ExecutableCommand` runs a handler. Help lists commands in the order they were added.

Executable commands and subcommands can have `Aliases`, can be `Hidden` or `Deprecated(message)`, and can go under a titled section of their parent's help with `Group("Setup")`. Any command can have a `Long` description and `Example`s for its own help page. The root can have a `Version`, and its own `Handler`, `Args` and `Flags` for when no command is named.

`RunContext(ctx, args)` passes `ctx` to handlers through `ctx.Context()`, so `signal.NotifyContext` cancels them on Ctrl-C. `Output(out, err)` redirects what gear prints.

## Arguments

| Constructor | Value |
|---|---|
| `NewStringArg` | `string` |
| `NewIntArg` | `int` (negative numbers work: `app hint -1`) |
| `NewFloatArg` | `float64` |
| `NewBoolArg` | `bool` |
| `NewDurationArg` | `time.Duration` |

Each returns a `TypedArg[T]` with `AsOptional()`, `AsVariadic()`, `Choices(...)` and `ExtendValidators(func(T) error)`. A variadic argument must come last, and a required argument can't follow an optional one.

Read values with `command.ArgValue[T](args, "label")`. A variadic argument reads as a slice, such as `ArgValue[[]string]`. `args.Has("label")` reports whether the argument was given.

## Flags

| Constructor | Value |
|---|---|
| `NewStringFlag` | `string` |
| `NewIntFlag` | `int` |
| `NewFloatFlag` | `float64` |
| `NewBoolFlag` | `bool` |
| `NewDurationFlag` | `time.Duration` |
| `NewStringsFlag` | `[]string`, collected from repeated use: `-t a -t b` |

Each returns a `TypedFlag[T]` with:

- `Choices(...)`: allow only these values
- `Env("APP_TOKEN")`: read from an environment variable when the flag isn't given
- `Required()`
- `Requires("other")`: only valid alongside `--other`
- `Negatable()`: add `--no-<name>` to a bool flag
- `ExtendValidators(func(T) error)`

`ExecutableCommand.MutuallyExclusive("a", "b")` rejects command lines that set both. `RootCommand.GlobalFlags` adds flags that every command accepts, and they can also go before the command name.

All of these work:

```bash
app serve --port=3000 --port 3000 -p 3000 -p=3000 -p3000
app ls -us            # same as -u -s when both are bool flags
app new -- --odd-name # everything after -- is positional
```

Read values with `command.FlagValue[T](args, "name")`. `args.Changed("name")` reports whether the flag was set on the command line or from its environment variable, as opposed to left at its default.

## Help and errors

Every command understands `--help`, and also `-h` unless one of its flags uses `-h`. Groups also understand `help <command>`. Help shows the full command path, a usage line built from the arguments, and global flags in their own section.

Bad input, such as an unknown command or flag, a missing argument or an invalid value, returns a `*UsageError`. Gear prints a short usage hint to stderr, and suggests the nearest match for typos. A handler can return its own with `command.UsageErrorf`. `command.ExitCode(err)` maps errors to the usual exit statuses.

### Customizing help

Help wraps to `$COLUMNS` (or 80 columns). `RootCommand.HelpStyle` takes hooks that style each part of a page, plus a fixed `Width`. Each hook is a plain `func(string) string`, so a lipgloss `Style.Render` fits without gear depending on it. Columns are lined up and wrapped before styling, so escape codes don't break the layout.

```go
root.HelpStyle(command.HelpStyle{
    Heading: headingStyle.Render,
    Flag:    flagStyle.Render,
    Note:    dimStyle.Render, // defaults, $ENV, choices
})
```

For full control, `RootCommand.HelpFunc` gets each page as a `command.Help` value with every section broken out. Draw it yourself, or change it and pass it to `command.WriteHelp`. `root.HelpFor("db", "migrate")` and `ctx.Help()` return the same value outside of `--help`, for example to generate docs.

Mistakes in how the tree is built panic: duplicate names, clashing flags, or misordered arguments. The builder method panics if it can see the problem; otherwise the first run does. Either way, the developer sees it the first time the program runs, whatever command they try.

## Shell completion

```go
root.AddChild(root.CompletionCommand())
```

```bash
source <(app completion bash)   # or zsh
app completion fish | source
```

The scripts ask the program what to suggest, through a hidden `__complete` command, so completion always matches the command tree. It covers command names, flags, choices, and values from `Complete` functions on arguments and flags:

```go
command.NewStringArg("template", "Template to use").
    Complete(func(ctx *command.Context, args command.ValidatedArgs, prefix string) []string {
        return listTemplates() // "name\tdescription" adds a description in zsh and fish
    })
```

A `Complete` function sees the flags and arguments typed before the value. Where there's nothing to suggest, the shell completes file names. `root.WriteCompletion(w, shell)` writes a script directly.

## Notes

`Command` has unexported methods, so it can only be one of the three types above. Commands keep no state between runs, so a tree can be run repeatedly or from several goroutines.

## License

MIT
