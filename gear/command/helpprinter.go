package command

import (
	"fmt"
	"io"
	"os"
)

type helpPrinter struct {
	writer io.Writer
}

func newHelpPrinter() *helpPrinter {
	return &helpPrinter{
		writer: os.Stdout,
	}
}

func (h *helpPrinter) PrintRootCommandHelp(cmd *RootCommand) {
	fmt.Fprintf(h.writer, "Usage: %s [command] [options]\n\n", cmd.Label())
	fmt.Fprintf(h.writer, "%s\n\n", cmd.Description())
	fmt.Fprintf(h.writer, "Available Commands:\n")
	for _, child := range cmd.children {
		fmt.Fprintf(h.writer, "  %-15s %s\n", child.Label(), child.Description())
	}
}

func (h *helpPrinter) PrintSubcommandHelp(cmd *Subcommand) {
	fmt.Fprintf(h.writer, "Subcommand: %s\n", cmd.Label())
	fmt.Fprintf(h.writer, "%s\n\n", cmd.Description())
	fmt.Fprintf(h.writer, "Available Subcommands:\n")
	for _, child := range cmd.children {
		fmt.Fprintf(h.writer, "  %-15s %s\n", child.Label(), child.Description())
	}
}

func (h *helpPrinter) PrintExecutableCommandHelp(cmd *executableCommand) {
	fmt.Fprintf(h.writer, "Command: %s\n", cmd.Label())
	fmt.Fprintf(h.writer, "%s\n\n", cmd.Description())

	if len(cmd.flags) > 0 {
		fmt.Fprintf(h.writer, "Flags:\n")
		for _, flag := range cmd.flags {
			shorthandDisplay := ""
			if flag.Shorthand() != "" {
				shorthandDisplay = fmt.Sprintf(", -%s", flag.Shorthand())
			}
			defaultDisplay := ""
			if flag.DefaultValue() != nil {
				defaultDisplay = fmt.Sprintf(" (default: %v)", flag.DefaultValue())
			}
			fmt.Fprintf(h.writer, "  --%s%s (%s)%s\n      %s\n",
				flag.Name(),
				shorthandDisplay,
				flag.Expected(),
				defaultDisplay,
				flag.Description())
		}
		fmt.Fprintf(h.writer, "\n")
	}

	if len(cmd.args) > 0 {
		fmt.Fprintf(h.writer, "Arguments:\n")
		for _, arg := range cmd.args {
			optionalMarker := ""
			if arg.IsOptional() {
				optionalMarker = " (optional)"
			}
			fmt.Fprintf(h.writer, "  %-15s (%s)%s - %s\n",
				arg.Label(),
				arg.Expected(),
				optionalMarker,
				arg.Description())
		}
	}
}
