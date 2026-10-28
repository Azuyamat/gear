package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// result is what one run produced.
type result struct {
	args   ValidatedArgs
	ran    bool
	out    string
	errOut string
	err    error
}

// runTree runs args against root, with output captured, and records what the
// handler named "capture" saw.
func runTree(t *testing.T, root *RootCommand, args ...string) result {
	t.Helper()
	var out, errOut bytes.Buffer
	root.Output(&out, &errOut)
	var r result
	captured = &r
	r.err = root.Run(args)
	r.out, r.errOut = out.String(), errOut.String()
	return r
}

// captured is the result the current runTree fills in. Tests that use it
// don't run in parallel.
var captured *result

func record(ctx *Context, args ValidatedArgs) error {
	captured.args, captured.ran = args, true
	return nil
}

func newTestRoot(cmds ...Command) *RootCommand {
	root := NewRootCommand("app", "Test app")
	for _, c := range cmds {
		root.AddChild(c)
	}
	return root
}

func exe(label string) *ExecutableCommand {
	return NewExecutableCommand(label, "The "+label+" command").Handler(record)
}

func mustPanic(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("no panic, want one mentioning %q", want)
		}
		if !strings.Contains(fmt.Sprint(r), want) {
			t.Fatalf("panic %q doesn't mention %q", r, want)
		}
	}()
	f()
}

func TestHelpReturnsNil(t *testing.T) {
	root := newTestRoot(exe("pull"), NewSubcommand("config", "Config").AddChild(exe("set")))
	for _, args := range [][]string{
		{"pull", "--help"}, {"pull", "-h"},
		{"--help"}, {"-h"}, {"help"}, {},
		{"help", "pull"}, {"help", "config", "set"},
		{"config", "--help"}, {"config", "help", "set"}, {"config"},
	} {
		r := runTree(t, root, args...)
		if r.err != nil || r.ran {
			t.Errorf("%v: err %v, ran %v", args, r.err, r.ran)
		}
		if !strings.Contains(r.out, "Usage:") {
			t.Errorf("%v: no help printed:\n%s", args, r.out)
		}
	}
	if r := runTree(t, root, "help", "config", "set"); !strings.Contains(r.out, "app config set [flags]") {
		t.Errorf("help config set shows the wrong page:\n%s", r.out)
	}
}

func TestChildrenKeepTheirOrder(t *testing.T) {
	names := []string{"zeta", "alpha", "mid", "beta", "omega"}
	root := newTestRoot()
	for _, n := range names {
		root.AddChild(exe(n))
	}
	out := runTree(t, root, "--help").out
	last := -1
	for _, n := range names {
		i := strings.Index(out, "  "+n+" ")
		if i < last {
			t.Fatalf("commands out of order:\n%s", out)
		}
		last = i
	}
}

func TestHShorthandOnlyMeansHelpWhenFree(t *testing.T) {
	root := newTestRoot(exe("serve").Flags(NewStringFlag("host", "h", "Host", "localhost")))
	r := runTree(t, root, "serve", "-h", "0.0.0.0")
	if r.err != nil || FlagValue[string](r.args, "host") != "0.0.0.0" {
		t.Fatalf("err %v, host %q", r.err, FlagValue[string](r.args, "host"))
	}
	if r := runTree(t, root, "serve", "--help"); !strings.Contains(r.out, "    --help") {
		t.Errorf("help lists -h for help although --host has it:\n%s", r.out)
	}
}

func TestNegativeNumbers(t *testing.T) {
	root := newTestRoot(exe("hint").Args(NewIntArg("number", "N"), NewFloatArg("f", "F").AsOptional()))
	r := runTree(t, root, "hint", "-1", "-2.5")
	if r.err != nil || ArgValue[int](r.args, "number") != -1 || ArgValue[float64](r.args, "f") != -2.5 {
		t.Fatalf("err %v, args %v", r.err, r.args.args)
	}
}

func TestShortFlagForms(t *testing.T) {
	cmd := exe("ls").Flags(
		NewBoolFlag("unsolved", "u", "", false),
		NewBoolFlag("short", "s", "", false),
		NewIntFlag("port", "p", "", 0),
	)
	root := newTestRoot(cmd)

	for _, args := range [][]string{{"-us"}, {"-u", "-s"}, {"-su"}} {
		r := runTree(t, root, append([]string{"ls"}, args...)...)
		if r.err != nil || !FlagValue[bool](r.args, "unsolved") || !FlagValue[bool](r.args, "short") {
			t.Errorf("%v: err %v, flags %v", args, r.err, r.args.flags)
		}
	}
	for _, args := range [][]string{{"-p3000"}, {"-p", "3000"}, {"-p=3000"}, {"-up3000"}} {
		r := runTree(t, root, append([]string{"ls"}, args...)...)
		if r.err != nil || FlagValue[int](r.args, "port") != 3000 {
			t.Errorf("%v: err %v, flags %v", args, r.err, r.args.flags)
		}
	}
	if r := runTree(t, root, "ls", "-uh"); r.err != nil || r.ran || !strings.Contains(r.out, "Usage:") {
		t.Errorf("-uh: err %v, ran %v", r.err, r.ran)
	}
	if r := runTree(t, root, "ls", "-ux"); r.err == nil || !strings.Contains(r.err.Error(), "-x in -ux") {
		t.Errorf("-ux: err %v", r.err)
	}
}

func TestChangedAndDefaults(t *testing.T) {
	root := newTestRoot(exe("pull").Flags(
		NewStringFlag("template", "t", "", ""),
		NewIntFlag("n", "", "", 5),
	).Args(NewStringArg("cats", "").AsVariadic().AsOptional()))

	r := runTree(t, root, "pull")
	if r.args.Changed("template") || r.args.Changed("n") || FlagValue[int](r.args, "n") != 5 || r.args.Has("cats") {
		t.Errorf("nothing given: %+v", r.args)
	}
	r = runTree(t, root, "pull", "--template", "", "a", "b")
	if !r.args.Changed("template") || r.args.Changed("n") {
		t.Errorf("--template \"\": %+v", r.args)
	}
	if got := ArgValue[[]string](r.args, "cats"); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("cats = %v", got)
	}
}

func TestArgSetupPanicsEarly(t *testing.T) {
	mustPanic(t, "must be last", func() {
		exe("x").Args(NewStringArg("a", "").AsVariadic(), NewStringArg("b", "").AsVariadic())
	})
	mustPanic(t, "must be last", func() {
		exe("x").Args(NewStringArg("a", "").AsVariadic(), NewStringArg("b", ""))
	})
	mustPanic(t, "after optional", func() {
		exe("x").Args(NewStringArg("a", "").AsOptional(), NewStringArg("b", ""))
	})
	mustPanic(t, "labelled", func() {
		exe("x").Args(NewStringArg("a", "")).Args(NewStringArg("a", ""))
	})
}

func TestDuplicateNamesPanic(t *testing.T) {
	mustPanic(t, "two commands named", func() { newTestRoot(exe("a"), exe("a")) })
	mustPanic(t, "two commands named", func() { newTestRoot(exe("a"), exe("b").Aliases("a")) })
	mustPanic(t, "defined twice", func() {
		exe("a").Flags(NewBoolFlag("x", "", "", false), NewBoolFlag("x", "", "", false))
	})
	mustPanic(t, "used by both", func() {
		exe("a").Flags(NewBoolFlag("x", "v", "", false), NewBoolFlag("y", "v", "", false))
	})
	mustPanic(t, "global flag --verbose", func() {
		root := newTestRoot(exe("a").Flags(NewBoolFlag("verbose", "", "", false))).
			GlobalFlags(NewBoolFlag("verbose", "", "", false))
		root.Output(&bytes.Buffer{}, &bytes.Buffer{}).Run(nil)
	})
	mustPanic(t, "doesn't exist", func() {
		newTestRoot(exe("a").MutuallyExclusive("x", "y")).Run(nil)
	})
	mustPanic(t, "reserved", func() { exe("a").Flags(NewBoolFlag("help", "", "", false)) })
}

func TestRunIsRepeatableAndConcurrent(t *testing.T) {
	var mu sync.Mutex
	seen := map[int]int{}
	root := NewRootCommand("app", "").
		GlobalFlags(NewBoolFlag("verbose", "v", "", false)).
		AddChild(NewExecutableCommand("n", "").
			Args(NewIntArg("n", "")).
			Handler(func(ctx *Context, args ValidatedArgs) error {
				mu.Lock()
				seen[ArgValue[int](args, "n")]++
				mu.Unlock()
				return nil
			}))
	root.Output(&bytes.Buffer{}, &bytes.Buffer{})

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := root.Run([]string{"n", fmt.Sprint(i), "-v"}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	for i := range 50 {
		if seen[i] != 1 {
			t.Fatalf("n=%d ran %d times", i, seen[i])
		}
	}
	if n := len(root.globalFlags); n != 1 {
		t.Errorf("%d global flags after many runs, want 1", n)
	}
}

func TestGlobalFlagsAccumulate(t *testing.T) {
	root := newTestRoot(exe("a")).
		GlobalFlags(NewBoolFlag("x", "", "", false)).
		GlobalFlags(NewBoolFlag("y", "", "", false))
	r := runTree(t, root, "--x", "a", "--y")
	if r.err != nil || !FlagValue[bool](r.args, "x") || !FlagValue[bool](r.args, "y") {
		t.Fatalf("err %v, flags %v", r.err, r.args.flags)
	}
}

func TestBoolArgRejectsEmpty(t *testing.T) {
	root := newTestRoot(exe("a").Args(NewBoolArg("b", "")))
	if r := runTree(t, root, "a", ""); ExitCode(r.err) != 2 {
		t.Fatalf("err %v", r.err)
	}
}

func TestUsageErrors(t *testing.T) {
	root := newTestRoot(exe("pull").Args(NewIntArg("number", "")).Flags(NewBoolFlag("dry-run", "n", "", false)))
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"pul"}, `did you mean "pull"`},
		{[]string{"pull", "x"}, `'number' must be a whole number (got "x")`},
		{[]string{"pull"}, "missing argument <number>"},
		{[]string{"pull", "1", "2"}, `unexpected argument "2"`},
		{[]string{"pull", "1", "--dryrun"}, "did you mean --dry-run"},
		{[]string{"pull", "1", "--dry-run=maybe"}, "must be true or false"},
		{[]string{"help", "nope"}, "unknown command"},
	}
	for _, c := range cases {
		r := runTree(t, root, c.args...)
		var usage *UsageError
		if !errors.As(r.err, &usage) || !strings.Contains(r.err.Error(), c.want) {
			t.Errorf("%v: err %v, want usage error with %q", c.args, r.err, c.want)
		}
		if !strings.Contains(r.errOut, "--help' for more") {
			t.Errorf("%v: no hint on stderr: %q", c.args, r.errOut)
		}
		if ExitCode(r.err) != 2 {
			t.Errorf("%v: exit code %d", c.args, ExitCode(r.err))
		}
	}

	failing := newTestRoot(NewExecutableCommand("a", "").Handler(func(*Context, ValidatedArgs) error {
		return errors.New("boom")
	}))
	if r := runTree(t, failing, "a"); ExitCode(r.err) != 1 {
		t.Errorf("handler error: exit code %d", ExitCode(r.err))
	}
	userUsage := newTestRoot(NewExecutableCommand("a", "").Handler(func(*Context, ValidatedArgs) error {
		return UsageErrorf("bad")
	}))
	var usage *UsageError
	if r := runTree(t, userUsage, "a"); !errors.As(r.err, &usage) || usage.Command != "app a" {
		t.Errorf("UsageErrorf: %#v", r.err)
	}
}

func TestFlagRules(t *testing.T) {
	cmd := exe("login").Flags(
		NewBoolFlag("session", "", "", false),
		NewStringFlag("user", "u", "", "").Requires("session"),
		NewBoolFlag("cookie", "", "", false),
		NewStringFlag("token", "", "", "").Env("GEAR_TEST_TOKEN"),
		NewStringFlag("format", "f", "", "text").Choices("text", "json"),
		NewStringsFlag("tag", "t", "", []string{"default"}),
		NewDurationFlag("timeout", "", "", time.Second),
		NewBoolFlag("color", "", "", true).Negatable(),
	).MutuallyExclusive("session", "cookie")
	root := newTestRoot(cmd)

	if r := runTree(t, root, "login", "--session", "--cookie"); r.err == nil || !strings.Contains(r.err.Error(), "--session and --cookie can't be used together") {
		t.Errorf("exclusive: %v", r.err)
	}
	if r := runTree(t, root, "login", "-u", "me"); r.err == nil || !strings.Contains(r.err.Error(), "--user can only be used with --session") {
		t.Errorf("requires: %v", r.err)
	}
	if r := runTree(t, root, "login", "-f", "xml"); r.err == nil || !strings.Contains(r.err.Error(), "one of text, json") {
		t.Errorf("choices: %v", r.err)
	}

	t.Setenv("GEAR_TEST_TOKEN", "secret")
	r := runTree(t, root, "login", "-t", "a", "--tag=b", "--timeout", "2m", "--no-color")
	if r.err != nil {
		t.Fatal(r.err)
	}
	if FlagValue[string](r.args, "token") != "secret" || !r.args.Changed("token") {
		t.Errorf("env: %v", r.args.flags)
	}
	if got := FlagValue[[]string](r.args, "tag"); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("tag = %v", got)
	}
	if FlagValue[time.Duration](r.args, "timeout") != 2*time.Minute || FlagValue[bool](r.args, "color") {
		t.Errorf("flags %v", r.args.flags)
	}

	required := newTestRoot(exe("a").Flags(NewStringFlag("key", "", "", "").Required()))
	if r := runTree(t, required, "a"); r.err == nil || !strings.Contains(r.err.Error(), "required flag --key") {
		t.Errorf("required: %v", r.err)
	}
}

func TestAliasesHiddenDeprecated(t *testing.T) {
	root := newTestRoot(exe("ls").Aliases("list"), exe("secret").Hidden(), exe("old").Deprecated("use ls"))
	if r := runTree(t, root, "list"); !r.ran {
		t.Error("alias didn't run")
	}
	if r := runTree(t, root, "secret"); !r.ran {
		t.Error("hidden command didn't run")
	}
	r := runTree(t, root, "old")
	if !r.ran || !strings.Contains(r.errOut, "use ls") {
		t.Errorf("deprecated: ran %v, stderr %q", r.ran, r.errOut)
	}
	out := runTree(t, root).out
	if strings.Contains(out, "secret") || strings.Contains(out, "old") {
		t.Errorf("help lists hidden commands:\n%s", out)
	}
}

func TestRootHandlerAndVersion(t *testing.T) {
	root := NewRootCommand("app", "").Version("1.2.3").
		Args(NewStringArg("file", "").AsOptional()).
		Handler(record).
		AddChild(exe("sub"))

	if r := runTree(t, root, "--version"); r.out != "app 1.2.3\n" {
		t.Errorf("version: %q", r.out)
	}
	if r := runTree(t, root, "notes.txt"); !r.ran || ArgValue[string](r.args, "file") != "notes.txt" {
		t.Errorf("root handler: %+v", r)
	}
	if r := runTree(t, root); !r.ran {
		t.Error("root handler didn't run without args")
	}
	if r := runTree(t, root, "sub"); !r.ran || r.args.Has("file") {
		t.Errorf("child: %+v", r)
	}
}

func TestRunContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	root := newTestRoot(NewExecutableCommand("a", "").Handler(func(c *Context, _ ValidatedArgs) error {
		return c.Context().Err()
	}))
	root.Output(&bytes.Buffer{}, &bytes.Buffer{})
	if err := root.RunContext(ctx, []string{"a"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err %v", err)
	}
}

func TestHelpOutput(t *testing.T) {
	root := NewRootCommand("ctf", "Manage CTF challenges").
		GlobalFlags(NewBoolFlag("verbose", "v", "Verbose output", false)).
		AddChild(NewSubcommand("config", "Manage configuration").
			AddChild(exe("set").Args(NewStringArg("key", "Key"), NewStringArg("value", "Value")))).
		AddChild(NewExecutableCommand("pull", "Pull challenges").
			Aliases("p").
			Args(NewStringArg("categories", "Only pull these categories").AsVariadic().AsOptional()).
			Flags(
				NewBoolFlag("dry-run", "n", "Show what would change", false),
				NewStringFlag("template", "t", "Template to use", ""),
				NewIntFlag("jobs", "", "Downloads at once", 4),
			))

	want := `Pull challenges

Usage:
  ctf pull [categories...] [flags]

Aliases:
  pull, p

Arguments:
  categories...   Only pull these categories

Flags:
  -n, --dry-run           Show what would change
  -t, --template string   Template to use
      --jobs int          Downloads at once (default 4)
  -h, --help              Show help

Global flags:
  -v, --verbose   Verbose output
`
	if got := runTree(t, root, "pull", "--help").out; got != want {
		t.Errorf("pull help:\n%s\nwant:\n%s", got, want)
	}

	want = `Manage CTF challenges

Usage:
  ctf <command> [flags]

Commands:
  config   Manage configuration
  pull     Pull challenges

Flags:
  -h, --help   Show help

Global flags:
  -v, --verbose   Verbose output

Run 'ctf <command> --help' for more about a command.
`
	if got := runTree(t, root).out; got != want {
		t.Errorf("root help:\n%s\nwant:\n%s", got, want)
	}

	if got := runTree(t, root, "config", "set", "-h").out; !strings.Contains(got, "ctf config set <key> <value> [flags]") {
		t.Errorf("nested help doesn't show the full path:\n%s", got)
	}
}

// completionRoot is the tree the completion tests use. TestMain runs it as a
// real program when the bash test calls back into the test binary.
func completionRoot() *RootCommand {
	categories := func(ctx *Context, args ValidatedArgs, prefix string) []string {
		if FlagValue[bool](args, "all") {
			return []string{"web\tWeb challenges", "pwn", "crypto", "misc"}
		}
		return []string{"web\tWeb challenges", "pwn"}
	}
	return NewRootCommand("app", "").Version("1").
		GlobalFlags(NewBoolFlag("verbose", "v", "Verbose output", false)).
		AddChild(NewSubcommand("db", "Database").Aliases("d").
			AddChild(exe("migrate").Flags(
				NewStringFlag("format", "f", "Output format", "text").Choices("text", "json"),
				NewBoolFlag("color", "", "Color", true).Negatable(),
			))).
		AddChild(exe("pull").
			Args(NewStringArg("categories", "").AsVariadic().AsOptional().Complete(categories)).
			Flags(
				NewBoolFlag("all", "a", "Every category", false),
				NewStringFlag("template", "t", "Template", "").Complete(func(*Context, ValidatedArgs, string) []string {
					return []string{"default", "pwn-template"}
				}),
			)).
		AddChild(exe("open").Args(NewStringArg("file", "")))
}

func TestMain(m *testing.M) {
	if os.Getenv("GEAR_TEST_PROGRAM") == "1" {
		root := completionRoot()
		root.AddChild(root.CompletionCommand())
		if err := root.Run(os.Args[1:]); err != nil {
			os.Exit(ExitCode(err))
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestCompletionSuggestions(t *testing.T) {
	root := completionRoot()
	for _, c := range []struct {
		words []string
		want  []string
	}{
		{[]string{""}, []string{"db\tDatabase", "pull\tThe pull command", "open\tThe open command"}},
		{[]string{"p"}, []string{"pull\tThe pull command"}},
		{[]string{"d", ""}, []string{"migrate\tThe migrate command"}},
		{[]string{"db", "migrate", "--f"}, []string{"--format\tOutput format"}},
		{[]string{"db", "migrate", "-f", ""}, []string{"text", "json"}},
		{[]string{"db", "migrate", "--format=j"}, []string{"--format=json"}},
		{[]string{"db", "migrate", "--no"}, []string{"--no-color\tTurn off --color"}},
		{[]string{"--ver"}, []string{"--verbose\tVerbose output", "--version\tShow the version"}},
		{[]string{"pull", "--ver"}, []string{"--verbose\tVerbose output"}},
		{[]string{"pull", ""}, []string{"web\tWeb challenges", "pwn"}},
		{[]string{"pull", "web", "-a", "c"}, []string{"crypto"}},
		{[]string{"pull", "--template", "p"}, []string{"pwn-template"}},
		{[]string{"pull", "-t", ""}, []string{"default", "pwn-template"}},
		{[]string{"-v", "pull", "-v", "w"}, []string{"web\tWeb challenges"}},
		{[]string{"open", ""}, nil},
		{[]string{"nope", ""}, nil},
	} {
		r := runTree(t, root, append([]string{completeCommand}, c.words...)...)
		var got []string
		if r.out != "" {
			got = strings.Split(strings.TrimSuffix(r.out, "\n"), "\n")
		}
		if r.err != nil || !slices.Equal(got, c.want) {
			t.Errorf("%q: got %q (err %v), want %q", c.words, got, r.err, c.want)
		}
	}
}

func TestCompletionScriptsParse(t *testing.T) {
	root := completionRoot()
	root.AddChild(root.CompletionCommand())
	for shell, checker := range map[string][]string{
		"bash": {"bash", "-n"},
		"zsh":  {"zsh", "-n"},
		"fish": {"fish", "--no-execute"},
	} {
		r := runTree(t, root, "completion", shell)
		if r.err != nil || !strings.Contains(r.out, "__complete") {
			t.Errorf("%s: err %v\n%s", shell, r.err, r.out)
			continue
		}
		if _, err := exec.LookPath(checker[0]); err != nil {
			continue
		}
		cmd := exec.Command(checker[0], checker[1:]...)
		cmd.Stdin = strings.NewReader(r.out)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("%s rejects its script: %v\n%s\n%s", shell, err, out, r.out)
		}
	}
	if r := runTree(t, root, "completion", "tcsh"); ExitCode(r.err) != 2 {
		t.Errorf("tcsh: %v", r.err)
	}
}

// TestBashCompletion runs the bash script against this test binary, acting as
// the program through TestMain.
func TestBashCompletion(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("no bash")
	}
	var script bytes.Buffer
	if err := completionRoot().WriteCompletion(&script, "bash"); err != nil {
		t.Fatal(err)
	}
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	complete := func(line string) string {
		t.Helper()
		src := fmt.Sprintf("app() { GEAR_TEST_PROGRAM=1 %s \"$@\"; }\n%s\n"+
			"COMP_LINE=%s; COMP_POINT=${#COMP_LINE}; COMP_WORDBREAKS=$' \\t\\n\"'\\''><=;|&(:'\n"+
			"_app_complete; printf '%%s\\n' \"${COMPREPLY[@]}\"\n",
			shellQuote(bin), script.String(), shellQuote(line))
		out, err := exec.Command("bash", "-c", src).CombinedOutput()
		if err != nil {
			t.Fatalf("%q: %v\n%s", line, err, out)
		}
		return strings.Join(strings.Fields(string(out)), " ")
	}
	for line, want := range map[string]string{
		"app ":                      "db pull open completion",
		"app d ":                    "migrate",
		"app db migrate -f ":        "text json",
		"app db migrate --format=j": "json",
		"app pull -a c":             "crypto",
		"app pull --template pwn":   "pwn-template",
		"app open ":                 "",
	} {
		if got := complete(line); got != want {
			t.Errorf("%q: got %q, want %q", line, got, want)
		}
	}
}

func TestCommandGroupsLongAndExamples(t *testing.T) {
	root := NewRootCommand("ctf", "").
		AddChild(exe("pull").Group("Challenges")).
		AddChild(exe("version")).
		AddChild(exe("login").Group("Setup")).
		AddChild(exe("new").Group("Challenges")).
		AddChild(exe("secret").Group("Setup").Hidden()).
		AddChild(exe("submit").
			Long("Submit a flag.\n\nThe flag is checked against the event's format first.").
			Example("ctf submit 'flag{x}'", "# skip the format check\nctf submit --force x"))

	want := `Usage:
  ctf <command> [flags]

Challenges:
  pull      The pull command
  new       The new command

Setup:
  login     The login command

Other commands:
  version   The version command
  submit    The submit command
`
	if got := runTree(t, root).out; !strings.HasPrefix(got, want) {
		t.Errorf("groups:\n%s\nwant:\n%s", got, want)
	}

	want = `Submit a flag.

The flag is checked against the event's format first.

Usage:
  ctf submit [flags]

Examples:
  ctf submit 'flag{x}'
  # skip the format check
  ctf submit --force x
`
	if got := runTree(t, root, "submit", "-h").out; !strings.HasPrefix(got, want) {
		t.Errorf("long and examples:\n%s\nwant:\n%s", got, want)
	}
}

func TestHelpWraps(t *testing.T) {
	root := NewRootCommand("app", "A description long enough that it has to wrap at forty columns.").
		HelpStyle(HelpStyle{Width: 40}).
		AddChild(exe("x").Flags(NewIntFlag("count", "c", "How many items to process in one go", 10)))

	want := `A description long enough that it has to
wrap at forty columns.
`
	if got := runTree(t, root).out; !strings.HasPrefix(got, want) {
		t.Errorf("description:\n%s", got)
	}
	want = `  -c, --count int   How many items to
                    process in one go
                    (default 10)
`
	if got := runTree(t, root, "x", "-h").out; !strings.Contains(got, want) {
		t.Errorf("flags:\n%s\nwant:\n%s", got, want)
	}

	t.Setenv("COLUMNS", "200")
	root.HelpStyle(HelpStyle{})
	if got := runTree(t, root).out; !strings.HasPrefix(got, "A description long enough that it has to wrap at forty columns.\n") {
		t.Errorf("$COLUMNS ignored:\n%s", got)
	}
}

func TestHelpStyleKeepsAlignment(t *testing.T) {
	bold := func(s string) string { return "\x1b[1m" + s + "\x1b[0m" }
	root := NewRootCommand("app", "").
		HelpStyle(HelpStyle{Heading: bold, Flag: bold, Placeholder: bold, Note: bold, Command: bold, Code: bold, Arg: bold}).
		AddChild(exe("x").Flags(
			NewIntFlag("count", "c", "How many", 10),
			NewBoolFlag("quiet", "", "Say less", false),
		))
	got := runTree(t, root, "x", "-h").out
	plain := regexp.MustCompile("\x1b\\[[0-9]*m").ReplaceAllString(got, "")
	want := `Flags:
  -c, --count int   How many (default 10)
      --quiet       Say less
`
	if !strings.Contains(plain, want) {
		t.Errorf("styled help misaligned:\n%s", plain)
	}
	if !strings.Contains(got, bold("--count")) || !strings.Contains(got, bold("int")) || !strings.Contains(got, bold("Flags:")) {
		t.Errorf("styles not applied:\n%q", got)
	}
}

func TestHelpFuncAndHelpFor(t *testing.T) {
	var seen []Help
	root := newTestRoot(NewSubcommand("db", "Database").AddChild(exe("migrate").Example("app db migrate"))).
		HelpFunc(func(w io.Writer, h Help) {
			seen = append(seen, h)
			h.Footer = "custom footer"
			WriteHelp(w, h, HelpStyle{})
		})

	r := runTree(t, root, "db", "migrate", "--help")
	if len(seen) != 1 || seen[0].Path != "app db migrate" || !slices.Equal(seen[0].Examples, []string{"app db migrate"}) {
		t.Errorf("HelpFunc got %+v", seen)
	}
	if !strings.HasSuffix(r.out, "custom footer\n") {
		t.Errorf("HelpFunc output:\n%s", r.out)
	}

	h, err := root.HelpFor("db")
	if err != nil || h.Path != "app db" || h.CommandGroups[0].Commands[0].Name != "migrate" {
		t.Errorf("HelpFor: %+v, %v", h, err)
	}
	if _, err := root.HelpFor("db", "nope"); err == nil {
		t.Error("HelpFor found a missing command")
	}

	var fromCtx Help
	root = newTestRoot(NewExecutableCommand("a", "A").Handler(func(ctx *Context, _ ValidatedArgs) error {
		fromCtx = ctx.Help()
		return nil
	}))
	runTree(t, root, "a")
	if fromCtx.Path != "app a" || fromCtx.Summary != "A" {
		t.Errorf("ctx.Help: %+v", fromCtx)
	}
}
