package command

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Help is everything a command's help page shows. Gear builds one for each
// page and hands it to the renderer: WriteHelp unless RootCommand.HelpFunc
// replaces it. Fields that don't apply are empty.
type Help struct {
	// Path is the command's full name, such as "app db migrate".
	Path string
	// Summary is the one-line description, and Long the longer one set with
	// Long. The default renderer shows Long if there is one.
	Summary    string
	Long       string
	Deprecated string
	// Usage holds lines such as "app db migrate [steps] [flags]".
	Usage    []string
	Aliases  []string
	Examples []string
	// CommandGroups lists the child commands, in sections. Without any
	// Group calls there is one section, titled "Commands".
	CommandGroups []HelpCommandGroup
	Args          []HelpArg
	// Flags are the command's own flags, followed by --help (and --version
	// on a root that has a Version).
	Flags       []HelpFlag
	GlobalFlags []HelpFlag
	Footer      string
}

// HelpCommandGroup is one titled section of child commands.
type HelpCommandGroup struct {
	Title    string
	Commands []HelpCommand
}

// HelpCommand is a child command as help lists it.
type HelpCommand struct {
	Name        string
	Aliases     []string
	Description string
}

// HelpArg is a positional argument as help lists it.
type HelpArg struct {
	Name        string
	Description string
	Optional    bool
	Variadic    bool
	Choices     []string
}

// HelpFlag is a flag as help lists it.
type HelpFlag struct {
	Name      string
	Shorthand string
	// Placeholder names what the flag takes, such as "string" or
	// "text|json". It's empty for a bool flag.
	Placeholder string
	Description string
	// Default is the default formatted for display, or empty when it's the
	// type's zero value.
	Default    string
	Env        string
	Required   bool
	Negatable  bool
	Repeatable bool
}

// HelpStyle changes how WriteHelp draws a page. Each hook gets plain text and
// returns it styled, for example with ANSI colours; a nil hook leaves text
// alone. Gear lines columns up and wraps before styling, so escape codes don't
// upset the layout. A lipgloss Style's Render method fits these hooks as is.
type HelpStyle struct {
	Heading     func(string) string // section titles such as "Usage:"
	Code        func(string) string // usage lines and examples
	Command     func(string) string // child command names
	Arg         func(string) string // argument names
	Flag        func(string) string // flag spellings such as "-n, --dry-run"
	Placeholder func(string) string // what a flag takes, such as "string"
	Note        func(string) string // defaults, environment variables, choices
	// Width is the column to wrap text at. 0 uses $COLUMNS, or 80 if that
	// isn't set; a negative width turns wrapping off.
	Width int
}

func (s HelpStyle) width() int {
	if s.Width != 0 {
		return s.Width
	}
	if n, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && n > 0 {
		return n
	}
	return 80
}

func apply(style func(string) string, s string) string {
	if style == nil || s == "" {
		return s
	}
	return style(s)
}

// WriteHelp draws h on w the way gear does by default. A HelpFunc can change
// h and then call it, to adjust a page without drawing it from scratch.
func WriteHelp(w io.Writer, h Help, style HelpStyle) {
	width := style.width()
	var b strings.Builder
	heading := func(title string) {
		fmt.Fprintf(&b, "\n%s\n", apply(style.Heading, title+":"))
	}

	description := h.Long
	if description == "" {
		description = h.Summary
	}
	if description != "" {
		b.WriteString("\n")
		for _, l := range wrapText(strings.TrimSpace(description), width) {
			fmt.Fprintf(&b, "%s\n", l)
		}
	}
	if h.Deprecated != "" {
		b.WriteString("\n")
		for _, l := range wrapText("Deprecated: "+h.Deprecated, width) {
			fmt.Fprintf(&b, "%s\n", l)
		}
	}

	codeBlock := func(title string, lines []string) {
		if len(lines) == 0 {
			return
		}
		heading(title)
		for _, block := range lines {
			for _, l := range strings.Split(strings.TrimRight(block, "\n"), "\n") {
				fmt.Fprintf(&b, "  %s\n", apply(style.Code, l))
			}
		}
	}
	codeBlock("Usage", h.Usage)
	if len(h.Aliases) > 0 {
		heading("Aliases")
		fmt.Fprintf(&b, "  %s\n", strings.Join(h.Aliases, ", "))
	}
	codeBlock("Examples", h.Examples)

	// Command sections share one column width so they line up.
	var commandRows [][]row
	for _, g := range h.CommandGroups {
		var rows []row
		for _, c := range g.Commands {
			rows = append(rows, row{
				left:  spans{{c.Name, style.Command}},
				right: spans{{c.Description, nil}},
			})
		}
		commandRows = append(commandRows, rows)
	}
	commandWidth := 0
	for _, rows := range commandRows {
		commandWidth = max(commandWidth, columnWidth(rows))
	}
	for i, g := range h.CommandGroups {
		writeRows(&b, heading, g.Title, commandRows[i], commandWidth, width)
	}

	var args []row
	for _, a := range h.Args {
		name := a.Name
		if a.Variadic {
			name += "..."
		}
		right := spans{{a.Description, nil}}
		if len(a.Choices) > 0 {
			right = append(right, span{"(one of " + strings.Join(a.Choices, ", ") + ")", style.Note})
		}
		args = append(args, row{left: spans{{name, style.Arg}}, right: right})
	}
	writeRows(&b, heading, "Arguments", args, columnWidth(args), width)

	flags, global := flagRows(h.Flags, style), flagRows(h.GlobalFlags, style)
	writeRows(&b, heading, "Flags", flags, columnWidth(flags), width)
	writeRows(&b, heading, "Global flags", global, columnWidth(global), width)

	if h.Footer != "" {
		b.WriteString("\n")
		for _, l := range wrapText(h.Footer, width) {
			fmt.Fprintf(&b, "%s\n", l)
		}
	}
	io.WriteString(w, strings.TrimPrefix(b.String(), "\n"))
}

// span is a run of text and the style to draw it in.
type span struct {
	text  string
	style func(string) string
}

type spans []span

func (s spans) width() int {
	n := 0
	for _, sp := range s {
		n += utf8.RuneCountInString(sp.text)
	}
	return n
}

func (s spans) render() string {
	var b strings.Builder
	for _, sp := range s {
		b.WriteString(apply(sp.style, sp.text))
	}
	return b.String()
}

// row is one line of a two-column section: a name and its description.
type row struct {
	left, right spans
}

func columnWidth(rows []row) int {
	w := 0
	for _, r := range rows {
		w = max(w, r.left.width())
	}
	return w
}

const (
	indent = 2 // before the first column
	gutter = 3 // between the columns
)

// writeRows writes a titled two-column section. The second column is wrapped
// to fit width, with continuation lines lined up under it.
func writeRows(b *strings.Builder, heading func(string), title string, rows []row, leftWidth, width int) {
	if len(rows) == 0 {
		return
	}
	heading(title)
	pad := strings.Repeat(" ", indent+leftWidth+gutter)
	for _, r := range rows {
		lines := wrapSpans(r.right, width-indent-leftWidth-gutter)
		if len(lines) == 0 {
			fmt.Fprintf(b, "%*s%s\n", indent, "", r.left.render())
			continue
		}
		gap := strings.Repeat(" ", leftWidth-r.left.width()+gutter)
		fmt.Fprintf(b, "%*s%s%s%s\n", indent, "", r.left.render(), gap, lines[0].render())
		for _, l := range lines[1:] {
			fmt.Fprintf(b, "%s%s\n", pad, l.render())
		}
	}
}

// wrapSpans breaks styled text into lines of at most width characters,
// breaking between words. A width under 20 leaves the text on one line, as
// a column that narrow is harder to read than one that runs long.
func wrapSpans(text spans, width int) []spans {
	type word struct {
		text string
		from int // index of the span the word came from
	}
	var words []word
	for i, sp := range text {
		for _, w := range strings.Fields(sp.text) {
			words = append(words, word{w, i})
		}
	}
	if len(words) == 0 {
		return nil
	}
	if width < 20 {
		width = int(^uint(0) >> 1)
	}

	var lines [][]word
	var line []word
	lineWidth := 0
	for _, w := range words {
		n := utf8.RuneCountInString(w.text)
		if len(line) > 0 && lineWidth+1+n > width {
			lines = append(lines, line)
			line, lineWidth = nil, 0
		}
		if len(line) > 0 {
			lineWidth++
		}
		line = append(line, w)
		lineWidth += n
	}
	lines = append(lines, line)

	// Rebuild each line's spans, joining neighbouring words from the same
	// span so each is styled once.
	out := make([]spans, len(lines))
	for i, l := range lines {
		for j, w := range l {
			switch {
			case j == 0:
				out[i] = append(out[i], span{w.text, text[w.from].style})
			case l[j-1].from == w.from:
				out[i][len(out[i])-1].text += " " + w.text
			default:
				out[i] = append(out[i], span{" ", nil}, span{w.text, text[w.from].style})
			}
		}
	}
	return out
}

// wrapText wraps plain text to width, keeping its own line breaks.
func wrapText(text string, width int) []string {
	var out []string
	for _, para := range strings.Split(text, "\n") {
		lines := wrapSpans(spans{{para, nil}}, width)
		if len(lines) == 0 {
			out = append(out, "")
		}
		for _, l := range lines {
			out = append(out, l.render())
		}
	}
	return out
}

func flagRows(flags []HelpFlag, style HelpStyle) []row {
	var rows []row
	for _, f := range flags {
		var left spans
		if f.Shorthand != "" {
			left = append(left, span{"-" + f.Shorthand + ", ", style.Flag})
		} else {
			left = append(left, span{"    ", nil})
		}
		if f.Negatable {
			left = append(left, span{"--[no-]" + f.Name, style.Flag})
		} else {
			left = append(left, span{"--" + f.Name, style.Flag})
		}
		if f.Placeholder != "" {
			left = append(left, span{" ", nil}, span{f.Placeholder, style.Placeholder})
		}

		var notes []string
		if f.Repeatable {
			notes = append(notes, "repeatable")
		}
		if f.Default != "" {
			notes = append(notes, "default "+f.Default)
		}
		if f.Env != "" {
			notes = append(notes, "$"+f.Env)
		}
		if f.Required {
			notes = append(notes, "required")
		}
		right := spans{{f.Description, nil}}
		if len(notes) > 0 {
			right = append(right, span{" ", nil}, span{"(" + strings.Join(notes, ", ") + ")", style.Note})
		}
		rows = append(rows, row{left: left, right: right})
	}
	return rows
}

// buildHelp assembles the parts of a page that every kind of command shares.
func buildHelp(inv *invocation, c *baseCommand, usage []string, children []Command, args []Arg, flags []Flag) Help {
	h := Help{
		Path:       inv.pathString(),
		Summary:    c.description,
		Long:       c.long,
		Deprecated: c.deprecated,
		Usage:      usage,
		Examples:   c.examples,
	}
	if len(c.aliases) > 0 {
		h.Aliases = c.names()
	}

	h.CommandGroups = groupCommands(children)
	if len(children) > 0 {
		h.Footer = fmt.Sprintf("Run '%s <command> --help' for more about a command.", h.Path)
	}

	for _, a := range args {
		s := a.toArg()
		h.Args = append(h.Args, HelpArg{
			Name:        s.label,
			Description: s.description,
			Optional:    s.optional,
			Variadic:    s.variadic,
			Choices:     s.choices,
		})
	}

	// -h means help unless a flag on the command line claims it.
	helpFlag := HelpFlag{Name: "help", Shorthand: "h", Description: "Show help"}
	for _, f := range append(append([]Flag(nil), flags...), inv.global...) {
		if f.Shorthand() == "h" {
			helpFlag.Shorthand = ""
		}
	}
	h.Flags = append(helpFlags(flags), helpFlag)
	h.GlobalFlags = helpFlags(inv.global)
	return h
}

func helpFlags(flags []Flag) []HelpFlag {
	var out []HelpFlag
	for _, fl := range flags {
		f := fl.toFlag()
		hf := HelpFlag{
			Name:        f.name,
			Shorthand:   f.shorthand,
			Placeholder: valueHint(f),
			Description: f.description,
			Env:         f.env,
			Required:    f.required,
			Negatable:   f.negatable,
			Repeatable:  f.expected == ValueTypeStrings,
		}
		if !isZero(f.defaultValue) {
			hf.Default = formatDefault(f.defaultValue)
		}
		out = append(out, hf)
	}
	return out
}

// groupCommands sorts listed children into their Group sections, in the order
// each section first appears. Children without a group come last, under
// "Other commands", or under "Commands" when nothing is grouped.
func groupCommands(children []Command) []HelpCommandGroup {
	var groups []HelpCommandGroup
	var ungrouped []HelpCommand
	for _, c := range children {
		b := c.base()
		if !b.listed() {
			continue
		}
		hc := HelpCommand{Name: b.label, Aliases: b.aliases, Description: b.description}
		if b.groupTitle == "" {
			ungrouped = append(ungrouped, hc)
			continue
		}
		i := 0
		for i < len(groups) && groups[i].Title != b.groupTitle {
			i++
		}
		if i == len(groups) {
			groups = append(groups, HelpCommandGroup{Title: b.groupTitle})
		}
		groups[i].Commands = append(groups[i].Commands, hc)
	}
	if len(ungrouped) > 0 {
		title := "Commands"
		if len(groups) > 0 {
			title = "Other commands"
		}
		groups = append(groups, HelpCommandGroup{Title: title, Commands: ungrouped})
	}
	return groups
}

// valueHint names what a flag takes. A bool flag takes nothing.
func valueHint(f flag) string {
	switch {
	case len(f.choices) > 0:
		return strings.Join(f.choices, "|")
	case f.expected == ValueTypeBool:
		return ""
	case f.expected == ValueTypeStrings:
		return "string"
	default:
		return string(f.expected)
	}
}
