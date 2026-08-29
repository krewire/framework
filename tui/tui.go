// Package tui provides the terminal UI and command-line application model for the
// Krewire unified framework.
//
// It composes the Go standard library's flag and log/slog packages with the
// ecosystem's term package into a cohesive TUI/CLI application model. The
// package was formerly named cli; it was renamed to tui to avoid confusion
// with CLI tool binaries (e.g., krewire/krewire).
package tui

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/krewire/libs/term"
	"github.com/krewire/libs/vein"
)

// Command is a runnable sub-command.
type Command struct {
	Name     string
	About    string
	Group    string
	Example  string
	Register func(*flag.FlagSet)
	Run      func(*flag.FlagSet) vein.ExitCode
}

// NewCommand creates a Command with the given name, about, register and run
// functions.
func NewCommand(name, about string, register func(*flag.FlagSet), run func(*flag.FlagSet) vein.ExitCode) *Command {
	return &Command{Name: name, About: about, Register: register, Run: run}
}

// WithGroup sets the help group for the command and returns the command for chaining.
func (c *Command) WithGroup(group string) *Command {
	c.Group = group
	return c
}

// WithExample sets an example invocation and returns the command.
func (c *Command) WithExample(example string) *Command {
	c.Example = example
	return c
}

// App is the framework's CLI application model.
type App struct {
	name        string
	version     string
	description string
	commands    []*Command
	logger      *slog.Logger
	terminal    *term.Terminal
	stderr      io.Writer
}

// NewApp creates an App with the given name and version.
func NewApp(name, version string) *App {
	return &App{
		name:     name,
		version:  version,
		logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
		terminal: term.NewTerminal(),
		stderr:   os.Stderr,
	}
}

// WithDescription sets a one-line description shown under the banner.
func (a *App) WithDescription(desc string) *App {
	a.description = desc
	return a
}

// Command registers a Command and returns the App for chaining.
func (a *App) Command(cmd *Command) *App {
	a.commands = append(a.commands, cmd)
	return a
}

// Name returns the application name.
func (a *App) Name() string {
	return a.name
}

// Run parses tokens, dispatches to the matching command, and returns the
// resulting exit code. Help is consistent across all forms (subcommand and flag):
//
//	kiw help                → general help (success)
//	kiw --help | -h         → general help (success) — flag alias
//	kiw help <command>      → help for <command> (success) — canonical subcommand form
//	kiw <command> help      → help for <command> (success) — alias, kept for ergonomics
//	kiw <command> --help    → help for <command> (success) — flag form
//	kiw <command> -h        → help for <command> (success) — short flag
//
// Both subcommand and flag are supported for every command; they are equivalent
// and always exit 0 when help is shown. The canonical form is `kiw help <command>`.
func (a *App) Run(tokens []string) vein.ExitCode {
	if len(tokens) == 0 {
		a.printHelp()
		return vein.ExitCodeUsage
	}

	// General help via flag: kiw --help, kiw -h, kiw -help
	if tokens[0] == "--help" || tokens[0] == "-h" || tokens[0] == "-help" {
		a.printHelp()
		return vein.ExitCodeSuccess
	}

	// kiw help [command] — canonical subcommand form
	if tokens[0] == "help" {
		if len(tokens) == 1 {
			a.printHelp()
			return vein.ExitCodeSuccess
		}
		// Support `kiw help --help` → general help
		if tokens[1] == "--help" || tokens[1] == "-h" || tokens[1] == "-help" {
			a.printHelp()
			return vein.ExitCodeSuccess
		}
		if cmd := a.findCommand(tokens[1]); cmd != nil {
			a.printCommandHelp(cmd)
			return vein.ExitCodeSuccess
		}
		a.printUnknownCommand(tokens[1])
		a.printHelp()
		return vein.ExitCodeUsage
	}

	name := tokens[0]
	rest := tokens[1:]

	cmd := a.findCommand(name)
	if cmd == nil {
		a.printUnknownCommand(name)
		a.printHelp()
		return vein.ExitCodeUsage
	}

	// kiw <command> help | --help | -h  → show command help without running
	for _, arg := range rest {
		if arg == "help" || arg == "--help" || arg == "-h" || arg == "-help" {
			a.printCommandHelp(cmd)
			return vein.ExitCodeSuccess
		}
	}

	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	if cmd.Register != nil {
		cmd.Register(fs)
	}

	a.logger.Debug("dispatching command", "command", cmd.Name)
	if err := fs.Parse(rest); err != nil {
		return vein.ExitCodeUsage
	}
	return cmd.Run(fs)
}

// findCommand returns the Command with the given name, or nil.
func (a *App) findCommand(name string) *Command {
	for _, c := range a.commands {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func (a *App) printUnknownCommand(name string) {
	msg := a.terminal.Paint("✗", term.ColorRed, []term.Style{term.StyleBold}) + " unknown command " + a.terminal.Paint(fmt.Sprintf("%q", name), term.ColorYellow, nil)
	a.stderr.Write([]byte(msg + "\n"))
	if sug := a.suggestCommand(name); sug != "" {
		hint := a.terminal.Paint("  did you mean ", term.ColorDefault, []term.Style{term.StyleDim}) + a.terminal.Paint(sug, term.ColorCyan, nil) + a.terminal.Paint(" ?", term.ColorDefault, []term.Style{term.StyleDim})
		a.stderr.Write([]byte(hint + "\n"))
	}
	a.stderr.Write([]byte("\n"))
}

func (a *App) suggestCommand(name string) string {
	best := ""
	bestDist := 3
	for _, c := range a.commands {
		if strings.HasPrefix(c.Name, name) {
			return c.Name
		}
		d := levenshtein(name, c.Name)
		if d < bestDist {
			bestDist = d
			best = c.Name
		}
	}
	if bestDist <= 2 {
		return best
	}
	return ""
}

func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	cur := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		cur[0] = i
		for j := 1; j <= lb; j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[lb]
}

func (a *App) printHelp() {
	t := a.terminal
	// Banner
	name := t.Paint(a.name, term.ColorCyan, []term.Style{term.StyleBold})
	ver := t.Paint("v"+a.version, term.ColorDefault, []term.Style{term.StyleDim})
	a.stderr.Write([]byte(name + " " + ver + "\n"))
	if a.description != "" {
		a.stderr.Write([]byte(t.Paint(a.description, term.ColorDefault, []term.Style{term.StyleDim}) + "\n"))
	}
	a.stderr.Write([]byte("\n"))

	// Usage
	usageTitle := t.Paint("USAGE", term.ColorDefault, []term.Style{term.StyleBold, term.StyleDim})
	a.stderr.Write([]byte(usageTitle + "\n"))
	a.stderr.Write([]byte("  " + t.Paint(a.name, term.ColorCyan, nil) + " " + t.Paint("<command>", term.ColorYellow, nil) + " " + t.Paint("[options]", term.ColorDefault, []term.Style{term.StyleDim}) + "\n"))
	a.stderr.Write([]byte("  " + t.Paint(a.name+" help [command]", term.ColorDefault, []term.Style{term.StyleDim}) + t.Paint("          canonical", term.ColorDefault, []term.Style{term.StyleDim}) + "\n"))
	a.stderr.Write([]byte("  " + t.Paint(a.name+" --help | -h", term.ColorDefault, []term.Style{term.StyleDim}) + t.Paint("              flag alias", term.ColorDefault, []term.Style{term.StyleDim}) + "\n"))
	a.stderr.Write([]byte("\n"))

	// Commands grouped
	if len(a.commands) > 0 {
		cmdTitle := t.Paint("COMMANDS", term.ColorDefault, []term.Style{term.StyleBold, term.StyleDim})
		a.stderr.Write([]byte(cmdTitle + "\n"))
		groups := a.groupedCommands()
		maxLen := 0
		for _, c := range a.commands {
			if len(c.Name) > maxLen {
				maxLen = len(c.Name)
			}
		}
		for _, g := range groups {
			if g.name != "" {
				a.stderr.Write([]byte("  " + t.Paint(strings.ToUpper(g.name), term.ColorDefault, []term.Style{term.StyleDim}) + "\n"))
			}
			for _, cmd := range g.cmds {
				pad := strings.Repeat(" ", maxLen-len(cmd.Name))
				about := cmd.About
				if about == "" {
					about = t.Paint("—", term.ColorDefault, []term.Style{term.StyleDim})
				}
				line := "    " + t.Paint(cmd.Name, term.ColorCyan, nil) + pad + "  " + about
				if cmd.Example != "" {
					line += "\n      " + t.Paint("→ "+cmd.Example, term.ColorDefault, []term.Style{term.StyleDim})
				}
				a.stderr.Write([]byte(line + "\n"))
			}
			a.stderr.Write([]byte("\n"))
		}
		a.stderr.Write([]byte("  " + t.Paint("help", term.ColorCyan, nil) + strings.Repeat(" ", maxLen-4) + "  " + t.Paint("Show help", term.ColorDefault, []term.Style{term.StyleDim}) + "\n\n"))
	}

	// Footer
	a.stderr.Write([]byte(t.Paint("Run ", term.ColorDefault, []term.Style{term.StyleDim}) + t.Paint(a.name+" help <command>", term.ColorCyan, nil) + t.Paint(" or ", term.ColorDefault, []term.Style{term.StyleDim}) + t.Paint(a.name+" <command> --help", term.ColorCyan, nil) + t.Paint(" for details.", term.ColorDefault, []term.Style{term.StyleDim}) + "\n"))
}

type cmdGroup struct {
	name string
	cmds []*Command
}

func (a *App) groupedCommands() []cmdGroup {
	// Preserve insertion order of groups as first seen, but sort ungrouped last
	seen := map[string]int{}
	var groups []cmdGroup
	for _, c := range a.commands {
		g := c.Group
		if g == "" {
			g = "other"
		}
		if idx, ok := seen[g]; ok {
			groups[idx].cmds = append(groups[idx].cmds, c)
		} else {
			seen[g] = len(groups)
			groups = append(groups, cmdGroup{name: g, cmds: []*Command{c}})
		}
	}
	// Ensure deterministic order for known groups
	order := map[string]int{
		"project": 0,
		"inspect": 1,
		"build":   2,
		"develop": 3,
		"ship":    4,
		"other":   5,
	}
	sort.Slice(groups, func(i, j int) bool {
		oi, okI := order[groups[i].name]
		oj, okJ := order[groups[j].name]
		if okI && okJ {
			return oi < oj
		}
		if okI {
			return true
		}
		if okJ {
			return false
		}
		return groups[i].name < groups[j].name
	})
	// hide "other" label
	for i := range groups {
		if groups[i].name == "other" {
			groups[i].name = ""
		}
	}
	return groups
}

func (a *App) printCommandHelp(cmd *Command) {
	t := a.terminal
	title := t.Paint(a.name+" "+cmd.Name, term.ColorCyan, []term.Style{term.StyleBold})
	a.stderr.Write([]byte(title + "\n"))
	if cmd.About != "" {
		a.stderr.Write([]byte(cmd.About + "\n"))
	}
	if cmd.Example != "" {
		a.stderr.Write([]byte("\n" + t.Paint("Example:", term.ColorDefault, []term.Style{term.StyleBold, term.StyleDim}) + "\n"))
		a.stderr.Write([]byte("  " + t.Paint(cmd.Example, term.ColorGreen, nil) + "\n"))
	}
	a.stderr.Write([]byte("\n"))
	usageTitle := t.Paint("USAGE", term.ColorDefault, []term.Style{term.StyleBold, term.StyleDim})
	a.stderr.Write([]byte(usageTitle + "\n"))
	a.stderr.Write([]byte("  " + t.Paint(a.name+" "+cmd.Name, term.ColorCyan, nil) + " " + t.Paint("[options]", term.ColorDefault, []term.Style{term.StyleDim}) + "\n"))
	a.stderr.Write([]byte("  " + t.Paint(a.name+" help "+cmd.Name, term.ColorDefault, []term.Style{term.StyleDim}) + t.Paint("          canonical", term.ColorDefault, []term.Style{term.StyleDim}) + "\n"))
	a.stderr.Write([]byte("  " + t.Paint(a.name+" "+cmd.Name+" --help | -h", term.ColorDefault, []term.Style{term.StyleDim}) + t.Paint("    flag alias", term.ColorDefault, []term.Style{term.StyleDim}) + "\n"))
	a.stderr.Write([]byte("\n"))

	fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	if cmd.Register != nil {
		cmd.Register(fs)
	}
	hasFlags := false
	fs.VisitAll(func(f *flag.Flag) { hasFlags = true })
	if hasFlags {
		optsTitle := t.Paint("OPTIONS", term.ColorDefault, []term.Style{term.StyleBold, term.StyleDim})
		a.stderr.Write([]byte(optsTitle + "\n"))
		// Custom flag printing for aligned, colored output
		fs.VisitAll(func(f *flag.Flag) {
			defVal := ""
			if f.DefValue != "" && f.DefValue != "false" {
				defVal = t.Paint(" (default "+f.DefValue+")", term.ColorDefault, []term.Style{term.StyleDim})
			}
			name := t.Paint("--"+f.Name, term.ColorYellow, nil)
			if f.DefValue == "false" {
				// boolean flag
				name = t.Paint("--"+f.Name, term.ColorYellow, nil)
			}
			a.stderr.Write([]byte(fmt.Sprintf("  %-22s %s%s\n", name, f.Usage, defVal)))
		})
	} else {
		a.stderr.Write([]byte(t.Paint("No options.", term.ColorDefault, []term.Style{term.StyleDim}) + "\n"))
	}
	if cmd.Example != "" {
		a.stderr.Write([]byte("\n" + t.Paint("Try:", term.ColorDefault, []term.Style{term.StyleBold, term.StyleDim}) + " " + t.Paint(cmd.Example, term.ColorGreen, nil) + "\n"))
	}
}
