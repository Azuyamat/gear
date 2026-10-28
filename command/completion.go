package command

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"
)

// CompleteFunc suggests values for an argument or flag during shell
// completion. prefix is the part of the value typed so far; gear drops
// suggestions that don't start with it, so the function can ignore it. args
// holds the flags and arguments typed before the value, parsed as far as they
// would parse, so a suggestion can depend on them.
//
// A suggestion can carry a description after a tab, as in "web\tWeb
// challenges", which zsh and fish show. Returning nothing lets the shell
// complete file names instead.
type CompleteFunc func(ctx *Context, args ValidatedArgs, prefix string) []string

// completeCommand is the hidden command the completion scripts run. It gets
// the words typed after the program name, the last being the one to complete,
// and prints one suggestion per line.
const completeCommand = "__complete"

// CompletionCommand returns a "completion" command that prints the script
// WriteCompletion makes for the shell it's given. Add it with AddChild.
func (c *RootCommand) CompletionCommand() *ExecutableCommand {
	return NewExecutableCommand("completion", "Print a shell completion script").
		Args(NewStringArg("shell", "Shell to complete for").Choices("bash", "zsh", "fish")).
		Example(
			"# bash or zsh, for this shell session\nsource <("+c.label+" completion bash)",
			"# fish\n"+c.label+" completion fish | source",
		).
		Handler(func(ctx *Context, args ValidatedArgs) error {
			return c.WriteCompletion(ctx.Out(), ArgValue[string](args, "shell"))
		})
}

// WriteCompletion writes a completion script for shell, which is "bash",
// "zsh" or "fish", to w. The script asks the program itself what to suggest,
// so suggestions follow the command tree and any CompleteFunc. Where gear has
// nothing to suggest, the shell completes file names.
func (c *RootCommand) WriteCompletion(w io.Writer, shell string) error {
	fn := "_" + nonIdentifier.ReplaceAllString(c.label, "_") + "_complete"
	var script string
	switch shell {
	case "bash":
		script = bashScript
	case "zsh":
		script = zshScript
	case "fish":
		script = fishScript
	default:
		return fmt.Errorf("unsupported shell %q (want bash, zsh or fish)", shell)
	}
	script = strings.NewReplacer("{{fn}}", fn, "{{name}}", shellQuote(c.label), "{{complete}}", completeCommand).Replace(script)
	_, err := io.WriteString(w, script)
	return err
}

var nonIdentifier = regexp.MustCompile(`[^A-Za-z0-9_]`)

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// The scripts split the line on spaces without honouring quotes, which keeps
// them short and is right for nearly every command line.

const bashScript = `# bash completion for {{name}}
{{fn}}() {
    local line="${COMP_LINE:0:COMP_POINT}" cur out c
    local -a words lines
    read -ra words <<< "$line"
    [[ $line == *[[:space:]] ]] && words+=("")
    cur="${words[${#words[@]}-1]}"
    out=$("${words[0]}" {{complete}} "${words[@]:1}" 2>/dev/null) || return
    [[ -z $out ]] && return
    mapfile -t lines <<< "$out"
    COMPREPLY=()
    for c in "${lines[@]}"; do
        c="${c%%$'\t'*}"
        # bash completes only the part after "=" in --flag=value.
        if [[ $cur == --*=* && $COMP_WORDBREAKS == *=* ]]; then
            c="${c#"${cur%%=*}="}"
        fi
        COMPREPLY+=("$c")
    done
}
complete -o default -F {{fn}} {{name}}
`

const zshScript = `#compdef {{name}}
# zsh completion for {{name}}
{{fn}}() {
    local out line value desc
    local -a lines candidates
    out=$("${words[1]}" {{complete}} "${(@)words[2,CURRENT]}" 2>/dev/null) || return 1
    if [[ -z $out ]]; then
        _files
        return
    fi
    lines=("${(@f)out}")
    for line in "${lines[@]}"; do
        value="${line%%$'\t'*}"
        desc=""
        [[ $line == *$'\t'* ]] && desc="${line#*$'\t'}"
        candidates+=("${value//:/\\:}${desc:+:$desc}")
    done
    _describe -t values 'values' candidates
}
compdef {{fn}} {{name}}
`

const fishScript = `# fish completion for {{name}}
function {{fn}}
    set -l tokens (commandline -opc)
    set -l program $tokens[1]
    set -e tokens[1]
    set -l current (commandline -ct)
    set -l out ($program {{complete}} $tokens "$current" 2>/dev/null)
    if test (count $out) -eq 0
        __fish_complete_path "$current"
        return
    end
    printf '%s\n' $out
end
complete -c {{name}} -f -a '({{fn}})'
`

// complete prints the suggestions for the last of words, one per line.
func (c *RootCommand) complete(inv *invocation, words []string) {
	if len(words) == 0 {
		words = []string{""}
	}
	current := words[len(words)-1]
	for _, s := range c.suggestions(inv, words[:len(words)-1], current) {
		if strings.HasPrefix(s, current) {
			fmt.Fprintln(inv.out, s)
		}
	}
}

// completer walks the words before the one being completed, tracking where
// in the tree they lead and parsing what it can along the way.
type completer struct {
	inv  *invocation
	page Command            // the command whose help applies, for Context
	sub  *group             // the children words can still name; nil once past them
	exec *ExecutableCommand // whose arguments and flags apply, if any
	fs   *flagSet

	v          *ValidatedArgs
	positional []string
	pending    *flag // a flag waiting for its value in the next word
	dashdash   bool
}

func (c *RootCommand) suggestions(inv *invocation, before []string, current string) []string {
	k := &completer{inv: inv, page: c, sub: &c.group, exec: c.self, v: newValidatedArgs()}
	k.resetFlags()
	for _, w := range before {
		if !k.step(w) {
			return nil
		}
	}
	return k.suggest(current)
}

func (k *completer) resetFlags() {
	var own []Flag
	if k.exec != nil {
		own = k.exec.flags
	}
	k.fs = mustFlagSet(own, k.inv.global, nil)
}

// step takes in one word. It returns false if the word makes the line
// impossible to complete, such as an unknown command name.
func (k *completer) step(w string) bool {
	switch {
	case k.pending != nil:
		_ = k.fs.set(k.v, *k.pending, w)
		k.pending = nil
	case w == "--" && !k.dashdash:
		k.dashdash = true
	case !k.dashdash && len(w) > 1 && w[0] == '-' && !k.fs.isNegativeNumber(w):
		k.pending = k.fs.consume(w, k.v)
	case k.sub != nil:
		if child := k.sub.find(w); child != nil {
			k.inv = k.inv.sub(child.Label())
			k.page, k.sub, k.exec = child, child.subcommands(), nil
			if e, ok := child.(*ExecutableCommand); ok {
				k.exec = e
			}
			k.resetFlags()
			return true
		}
		if k.exec == nil {
			return false
		}
		// The root's own handler takes the word as its first argument.
		k.sub = nil
		k.positional = append(k.positional, w)
	default:
		k.positional = append(k.positional, w)
	}
	return true
}

// consume parses one flag word as well as it can, ignoring errors. It returns
// the flag if it still needs its value from the next word.
func (fs *flagSet) consume(w string, v *ValidatedArgs) *flag {
	if body, ok := strings.CutPrefix(w, "--"); ok {
		name, value, hasValue := strings.Cut(body, "=")
		if f, ok := fs.negated[name]; ok {
			_ = fs.set(v, f, "false")
			return nil
		}
		f, ok := fs.byName[name]
		switch {
		case !ok:
		case hasValue:
			_ = fs.set(v, f, value)
		case !f.takesValue():
			_ = fs.set(v, f, "true")
		default:
			return &f
		}
		return nil
	}
	body := w[1:]
	for i, r := range body {
		f, ok := fs.byShorthand[string(r)]
		if !ok {
			return nil
		}
		if !f.takesValue() {
			_ = fs.set(v, f, "true")
			continue
		}
		rest := body[i+utf8.RuneLen(r):]
		if rest == "" {
			return &f
		}
		_ = fs.set(v, f, strings.TrimPrefix(rest, "="))
		return nil
	}
	return nil
}

func (k *completer) suggest(current string) []string {
	switch {
	case k.pending != nil:
		return k.values(*k.pending, current)
	case k.dashdash:
		return k.argValues(current)
	case strings.HasPrefix(current, "--") && strings.Contains(current, "="):
		name, prefix, _ := strings.Cut(current[2:], "=")
		f, ok := k.fs.byName[name]
		if !ok || !f.takesValue() {
			return nil
		}
		var out []string
		for _, s := range k.values(f, prefix) {
			out = append(out, "--"+name+"="+s)
		}
		return out
	case strings.HasPrefix(current, "-"):
		return k.flagNames()
	}

	var out []string
	if k.sub != nil {
		for _, c := range k.sub.visible() {
			out = append(out, c.Label()+"\t"+c.Description())
		}
	}
	if k.sub == nil || k.exec != nil {
		out = append(out, k.argValues(current)...)
	}
	return out
}

func (k *completer) flagNames() []string {
	var out []string
	for _, f := range k.fs.flags {
		out = append(out, "--"+f.name+"\t"+f.description)
		if f.negatable {
			out = append(out, "--no-"+f.name+"\tTurn off --"+f.name)
		}
	}
	out = append(out, "--help\tShow help")
	if root, ok := k.page.(*RootCommand); ok && root.version != "" && k.sub != nil {
		out = append(out, "--version\tShow the version")
	}
	return out
}

func (k *completer) context() *Context {
	return newContext(k.inv, k.page)
}

// values suggests values for a flag: its choices, or what its CompleteFunc
// returns.
func (k *completer) values(f flag, prefix string) []string {
	if len(f.choices) > 0 {
		return f.choices
	}
	if f.complete != nil {
		return f.complete(k.context(), *k.v, prefix)
	}
	return nil
}

// argValues suggests values for the next positional argument.
func (k *completer) argValues(prefix string) []string {
	if k.exec == nil || len(k.exec.args) == 0 {
		return nil
	}
	i := len(k.positional)
	args := k.exec.args
	if i >= len(args) {
		if !args[len(args)-1].IsVariadic() {
			return nil
		}
		i = len(args) - 1
	}
	spec := args[i].toArg()
	if len(spec.choices) > 0 {
		return spec.choices
	}
	if spec.complete == nil {
		return nil
	}
	// Bind what's been typed so far, so the function can see earlier
	// arguments. Errors don't matter here.
	_ = k.exec.bindArgs(k.positional, k.v)
	return spec.complete(k.context(), *k.v, prefix)
}
