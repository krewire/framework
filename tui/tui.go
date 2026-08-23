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
	"io"
	"log/slog"
	"os"

	"github.com/krewire/libs/core"
	"github.com/krewire/libs/term"
)

// Command is a runnable sub-command.
type Command struct {
	// Name is the sub-command name as typed on the command line.
	Name string
	// About is a one-line description shown in help output.
	About string
	// Register defines the command's flags on the FlagSet.
	Register func(*flag.FlagSet)
	// Run executes the command against the parsed FlagSet and returns an
	// exit code.
	Run func(*flag.FlagSet) core.ExitCode
}

// NewCommand creates a Command with the given name, about, register and run
// functions.
func NewCommand(name, about string, register func(*flag.FlagSet), run func(*flag.FlagSet) core.ExitCode) *Command {
	return &Command{Name: name, About: about, Register: register, Run: run}
}

// App is the framework's CLI application model.
type App struct {
	name     string
	version  string
	commands []*Command
	logger   *slog.Logger
	terminal *term.Terminal
	stderr   io.Writer
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
func (a *App) Run(tokens []string) core.ExitCode {
	if len(tokens) == 0 {
		a.printHelp()
		return core.ExitCodeUsage
	}

	// General help via flag: kiw --help, kiw -h, kiw -help
	if tokens[0] == "--help" || tokens[0] == "-h" || tokens[0] == "-help" {
		a.printHelp()
		return core.ExitCodeSuccess
	}

	// kiw help [command] — canonical subcommand form
	if tokens[0] == "help" {
		if len(tokens) == 1 {
			a.printHelp()
			return core.ExitCodeSuccess
		}
		// Support `kiw help --help` → general help
		if tokens[1] == "--help" || tokens[1] == "-h" || tokens[1] == "-help" {
			a.printHelp()
			return core.ExitCodeSuccess
		}
		if cmd := a.findCommand(tokens[1]); cmd != nil {
			a.printCommandHelp(cmd)
			return core.ExitCodeSuccess
		}
		a.logger.Warn("unknown command", "command", tokens[1])
		a.printHelp()
		return core.ExitCodeUsage
	}

	name := tokens[0]
	rest := tokens[1:]

	cmd := a.findCommand(name)
	if cmd == nil {
		a.logger.Warn("unknown command", "command", name)
		a.printHelp()
		return core.ExitCodeUsage
	}

	// kiw <command> help | --help | -h  → show command help without running
	for _, arg := range rest {
		if arg == "help" || arg == "--help" || arg == "-h" || arg == "-help" {
			a.printCommandHelp(cmd)
			return core.ExitCodeSuccess
		}
	}

	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	if cmd.Register != nil {
		cmd.Register(fs)
	}

	a.logger.Debug("dispatching command", "command", cmd.Name)
	if err := fs.Parse(rest); err != nil {
		return core.ExitCodeUsage
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

func (a *App) printHelp() {
	title := a.terminal.Paint(a.name, term.ColorCyan, []term.Style{term.StyleBold})
	a.stderr.Write([]byte(title + " v" + a.version + "\n"))
	a.stderr.Write([]byte("Usage: " + a.name + " <command> [options]\n"))
	a.stderr.Write([]byte("       " + a.name + " help [command]          (canonical)\n"))
	a.stderr.Write([]byte("       " + a.name + " --help | -h              (flag alias for general help)\n"))
	a.stderr.Write([]byte("       " + a.name + " <command> --help | -h    (flag alias for command help)\n"))
	a.stderr.Write([]byte("       " + a.name + " <command> help           (alias, kept for ergonomics)\n\n"))
	a.stderr.Write([]byte("Commands:\n"))
	for _, cmd := range a.commands {
		a.stderr.Write([]byte("  " + cmd.Name + "  " + cmd.About + "\n"))
	}
	a.stderr.Write([]byte("  help  Show help (same as --help)\n"))
	a.stderr.Write([]byte("\nRun \"" + a.name + " help <command>\" (canonical) or \"" + a.name + " <command> --help\" (flag) for details.\n"))
}

func (a *App) printCommandHelp(cmd *Command) {
	title := a.terminal.Paint(a.name+" "+cmd.Name, term.ColorCyan, []term.Style{term.StyleBold})
	a.stderr.Write([]byte(title + "\n"))
	if cmd.About != "" {
		a.stderr.Write([]byte(cmd.About + "\n\n"))
	}
	a.stderr.Write([]byte("Usage: " + a.name + " " + cmd.Name + " [options]\n"))
	a.stderr.Write([]byte("       " + a.name + " help " + cmd.Name + "          (canonical)\n"))
	a.stderr.Write([]byte("       " + a.name + " " + cmd.Name + " --help | -h    (flag alias)\n"))
	a.stderr.Write([]byte("       " + a.name + " " + cmd.Name + " help           (alias)\n\n"))
	fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	if cmd.Register != nil {
		cmd.Register(fs)
	}
	// Only print flags if the command registers any.
	hasFlags := false
	fs.VisitAll(func(f *flag.Flag) { hasFlags = true })
	if hasFlags {
		a.stderr.Write([]byte("Options:\n"))
		fs.PrintDefaults()
	} else {
		a.stderr.Write([]byte("No options.\n"))
	}
}
