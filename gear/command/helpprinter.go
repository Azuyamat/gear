package gear

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
