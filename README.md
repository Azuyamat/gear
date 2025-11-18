# Gear

A lightweight, type-safe command-line interface framework for Go with automatic help generation, argument validation, and hierarchical command structures.

## Installation

```bash
go get github.com/azuyamat/gear
```

## Quick Start

```go
package main

import (
    "fmt"
    "os"

    "github.com/azuyamat/gear/command"
)

func main() {
    root := command.NewRootCommand("myapp", "A simple CLI application")

    greet := command.NewExecutableCommand("greet", "Greet a user").
        Args(
            command.NewStringArg("name", "Name of the person to greet"),
        ).
        Handler(func(ctx *command.Context, args command.ValidatedArgs) error {
            name := args.String("name")
            fmt.Printf("Hello, %s!\n", name)
            return nil
        })

    root.AddChild(greet)

    if err := root.Run(os.Args[1:]); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

```bash
$ myapp greet Alice
Hello, Alice!
```

## Features

- **Type-safe arguments and flags** with automatic parsing and validation
- **Hierarchical command structure** with root commands, subcommands, and executable commands
- **Automatic help generation** with `--help` / `-h` flags
- **Custom validators** for arguments and flags
- **Global flags** that propagate to child commands
- **Zero dependencies** - only uses Go standard library
- **Fluent API** for clean, readable command definitions

## Core Concepts

### Command Types

Gear provides three types of commands:

#### RootCommand

The top-level command that manages child commands.

```go
root := command.NewRootCommand("cli", "My CLI application")
root.AddChild(subcommand1)
root.AddChild(subcommand2)
root.Run(os.Args[1:])
```

#### Subcommand

A command that groups related child commands together.

```go
database := command.NewSubcommand("database", "Database operations")
database.AddChild(migrateCmd)
database.AddChild(seedCmd)

root.AddChild(database)
```

#### ExecutableCommand

A command that performs an action when invoked.

```go
migrate := command.NewExecutableCommand("migrate", "Run database migrations").
    Handler(func(ctx *command.Context, args command.ValidatedArgs) error {
        // Migration logic here
        return nil
    })
```

### Arguments

Arguments are positional parameters that come after the command name.

#### Type-Safe Arguments

```go
cmd := command.NewExecutableCommand("add", "Add two numbers").
    Args(
        command.NewIntArg("a", "First number"),
        command.NewIntArg("b", "Second number"),
    ).
    Handler(func(ctx *command.Context, args command.ValidatedArgs) error {
        a := args.Int("a")
        b := args.Int("b")
        fmt.Printf("%d + %d = %d\n", a, b, a+b)
        return nil
    })
```

#### Optional Arguments

```go
cmd := command.NewExecutableCommand("greet", "Greet someone").
    Args(
        command.NewStringArg("name", "Name of the person"),
        command.NewStringArg("greeting", "Custom greeting").AsOptional(),
    ).
    Handler(func(ctx *command.Context, args command.ValidatedArgs) error {
        name := args.String("name")
        greeting := args.String("greeting")
        if greeting == "" {
            greeting = "Hello"
        }
        fmt.Printf("%s, %s!\n", greeting, name)
        return nil
    })
```

#### Custom Validators

```go
import "fmt"

positiveValidator := func(value int) error {
    if value <= 0 {
        return fmt.Errorf("value must be positive")
    }
    return nil
}

cmd := command.NewExecutableCommand("process", "Process items").
    Args(
        command.NewIntArg("count", "Number of items").
            ExtendValidators(positiveValidator),
    )
```

### Flags

Flags are named options that can appear anywhere in the command line.

#### Basic Flags

```go
cmd := command.NewExecutableCommand("serve", "Start the server").
    Flags(
        command.NewIntFlag("port", "p", "Port to listen on", 8080),
        command.NewStringFlag("host", "h", "Host to bind to", "localhost"),
        command.NewBoolFlag("verbose", "v", "Enable verbose logging", false),
    ).
    Handler(func(ctx *command.Context, args command.ValidatedArgs) error {
        port := args.FlagInt("port")
        host := args.FlagString("host")
        verbose := args.FlagBool("verbose")

        fmt.Printf("Starting server on %s:%d (verbose: %v)\n", host, port, verbose)
        return nil
    })
```

Usage examples:

```bash
$ myapp serve --port=3000 --host=0.0.0.0 --verbose
$ myapp serve -p 3000 -h 0.0.0.0 -v
$ myapp serve --port=3000  # uses default host and verbose values
```

#### Flag Validators

```go
portValidator := func(value int) error {
    if value < 1024 || value > 65535 {
        return fmt.Errorf("port must be between 1024 and 65535")
    }
    return nil
}

cmd := command.NewExecutableCommand("serve", "Start server").
    Flags(
        command.NewIntFlag("port", "p", "Server port", 8080).
            ExtendValidators(portValidator),
    )
```

#### Global Flags

Global flags are inherited by all child commands.

```go
root := command.NewRootCommand("cli", "My application").
    GlobalFlags(
        command.NewBoolFlag("debug", "d", "Enable debug mode", false),
        command.NewStringFlag("config", "c", "Config file path", "config.yaml"),
    )

cmd := command.NewExecutableCommand("run", "Run the application").
    Handler(func(ctx *command.Context, args command.ValidatedArgs) error {
        debug := args.FlagBool("debug")
        config := args.FlagString("config")
        fmt.Printf("Debug: %v, Config: %s\n", debug, config)
        return nil
    })

root.AddChild(cmd)
```

```bash
$ myapp run --debug --config=custom.yaml
Debug: true, Config: custom.yaml
```

### Supported Types

Gear supports four built-in value types:

| Type | Constructor | Accessor Methods |
|------|-------------|------------------|
| String | `NewStringArg` / `NewStringFlag` | `String()` / `GetString()` / `FlagString()` / `GetFlagString()` |
| Int | `NewIntArg` / `NewIntFlag` | `Int()` / `GetInt()` / `FlagInt()` / `GetFlagInt()` |
| Float | `NewFloatArg` / `NewFloatFlag` | `Float()` / `GetFloat()` / `FlagFloat()` / `GetFlagFloat()` |
| Bool | `NewBoolArg` / `NewBoolFlag` | `Bool()` / `GetBool()` / `FlagBool()` / `GetFlagBool()` |

Each type has:
- Safe accessor (e.g., `GetString()`) - returns value and error
- Unsafe accessor (e.g., `String()`) - returns value only, error ignored

## Complete Example

```go
package main

import (
    "fmt"
    "os"

    "github.com/azuyamat/gear/command"
)

func main() {
    root := command.NewRootCommand("taskman", "Task management CLI").
        GlobalFlags(
            command.NewBoolFlag("verbose", "v", "Enable verbose output", false),
        )

    list := command.NewExecutableCommand("list", "List all tasks").
        Handler(func(ctx *command.Context, args command.ValidatedArgs) error {
            verbose := args.FlagBool("verbose")
            if verbose {
                fmt.Println("Listing all tasks (verbose mode)...")
            }
            fmt.Println("- Task 1")
            fmt.Println("- Task 2")
            return nil
        })

    add := command.NewExecutableCommand("add", "Add a new task").
        Args(
            command.NewStringArg("name", "Task name"),
            command.NewIntArg("priority", "Task priority (1-5)").
                ExtendValidators(func(value int) error {
                    if value < 1 || value > 5 {
                        return fmt.Errorf("priority must be between 1 and 5")
                    }
                    return nil
                }),
        ).
        Flags(
            command.NewStringFlag("tag", "t", "Task tag", ""),
        ).
        Handler(func(ctx *command.Context, args command.ValidatedArgs) error {
            name := args.String("name")
            priority := args.Int("priority")
            tag := args.FlagString("tag")

            fmt.Printf("Adding task: %s (priority: %d", name, priority)
            if tag != "" {
                fmt.Printf(", tag: %s", tag)
            }
            fmt.Println(")")
            return nil
        })

    database := command.NewSubcommand("db", "Database operations")

    migrate := command.NewExecutableCommand("migrate", "Run database migrations").
        Handler(func(ctx *command.Context, args command.ValidatedArgs) error {
            fmt.Println("Running migrations...")
            return nil
        })

    database.AddChild(migrate)

    root.AddChild(list)
    root.AddChild(add)
    root.AddChild(database)

    if err := root.Run(os.Args[1:]); err != nil {
        if err.Error() != "" {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            os.Exit(1)
        }
    }
}
```

Usage:

```bash
$ taskman list
- Task 1
- Task 2

$ taskman list --verbose
Listing all tasks (verbose mode)...
- Task 1
- Task 2

$ taskman add "Write documentation" 3 --tag=docs
Adding task: Write documentation (priority: 3, tag: docs)

$ taskman add "Invalid priority" 10
Error: validation failed for argument 'priority': priority must be between 1 and 5

$ taskman db migrate
Running migrations...

$ taskman --help
# Displays auto-generated help for all commands
```

## API Reference

### Command Creation

```go
command.NewRootCommand(label, description string) *RootCommand
command.NewSubcommand(label, description string) *Subcommand
command.NewExecutableCommand(label, description string) *ExecutableCommand
```

### Argument Creation

```go
command.NewStringArg(label, description string) typedArg[string]
command.NewIntArg(label, description string) typedArg[int]
command.NewFloatArg(label, description string) typedArg[float64]
command.NewBoolArg(label, description string) typedArg[bool]
```

### Flag Creation

```go
command.NewStringFlag(name, shorthand, description string, defaultValue string) typedFlag[string]
command.NewIntFlag(name, shorthand, description string, defaultValue int) typedFlag[int]
command.NewFloatFlag(name, shorthand, description string, defaultValue float64) typedFlag[float64]
command.NewBoolFlag(name, shorthand, description string, defaultValue bool) typedFlag[bool]
```

### ValidatedArgs Methods

```go
// Arguments
Get(name string) interface{}
Has(name string) bool
String(name string) string
GetString(name string) (string, error)
Int(name string) int
GetInt(name string) (int, error)
Float(name string) float64
GetFloat(name string) (float64, error)
Bool(name string) bool
GetBool(name string) (bool, error)

// Flags
GetFlag(name string) interface{}
HasFlag(name string) bool
FlagString(name string) string
GetFlagString(name string) (string, error)
FlagInt(name string) int
GetFlagInt(name string) (int, error)
FlagFloat(name string) float64
GetFlagFloat(name string) (float64, error)
FlagBool(name string) bool
GetFlagBool(name string) (bool, error)
```

## License

MIT
