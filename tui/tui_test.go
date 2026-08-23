package tui

import (
	"flag"
	"io"
	"testing"

	"github.com/krewire/libs/core"
)

func successHandler(*flag.FlagSet) core.ExitCode {
	return core.ExitCodeSuccess
}

func testApp() *App {
	app := NewApp("demo", "0.1.0").
		Command(NewCommand("ping", "ping the app", nil, successHandler))
	app.stderr = io.Discard
	return app
}

func TestRunDispatchesToKnownCommand(t *testing.T) {
	app := testApp()
	if got := app.Run([]string{"ping"}); got != core.ExitCodeSuccess {
		t.Errorf("Run() = %v, want %v", got, core.ExitCodeSuccess)
	}
}

func TestRunEmptyIsUsage(t *testing.T) {
	app := testApp()
	if got := app.Run(nil); got != core.ExitCodeUsage {
		t.Errorf("Run() = %v, want %v", got, core.ExitCodeUsage)
	}
}

func TestRunUnknownCommandIsUsage(t *testing.T) {
	app := testApp()
	if got := app.Run([]string{"nope"}); got != core.ExitCodeUsage {
		t.Errorf("Run() = %v, want %v", got, core.ExitCodeUsage)
	}
}

func TestRunParseErrorIsUsage(t *testing.T) {
	app := NewApp("demo", "0.1.0").
		Command(NewCommand("ping", "ping the app", func(fs *flag.FlagSet) {
			fs.String("name", "", "name of the target")
		}, successHandler))
	app.stderr = io.Discard
	if got := app.Run([]string{"ping", "--nope"}); got != core.ExitCodeUsage {
		t.Errorf("Run() = %v, want %v", got, core.ExitCodeUsage)
	}
}

func TestRunHelpGeneral(t *testing.T) {
	app := testApp()
	app.stderr = io.Discard
	if got := app.Run([]string{"help"}); got != core.ExitCodeSuccess {
		t.Errorf("Run(help) = %v, want %v", got, core.ExitCodeSuccess)
	}
	if got := app.Run([]string{"--help"}); got != core.ExitCodeSuccess {
		t.Errorf("Run(--help) = %v, want %v", got, core.ExitCodeSuccess)
	}
	if got := app.Run([]string{"-h"}); got != core.ExitCodeSuccess {
		t.Errorf("Run(-h) = %v, want %v", got, core.ExitCodeSuccess)
	}
}

func TestRunHelpCommand(t *testing.T) {
	app := testApp()
	app.stderr = io.Discard
	if got := app.Run([]string{"help", "ping"}); got != core.ExitCodeSuccess {
		t.Errorf("Run(help ping) = %v, want %v", got, core.ExitCodeSuccess)
	}
	if got := app.Run([]string{"help", "unknown"}); got != core.ExitCodeUsage {
		t.Errorf("Run(help unknown) = %v, want %v", got, core.ExitCodeUsage)
	}
}

func TestRunCommandHelpAliases(t *testing.T) {
	app := testApp()
	app.stderr = io.Discard
	aliases := [][]string{
		{"ping", "help"},
		{"ping", "--help"},
		{"ping", "-h"},
		{"ping", "-help"},
	}
	for _, args := range aliases {
		if got := app.Run(args); got != core.ExitCodeSuccess {
			t.Errorf("Run(%v) = %v, want %v", args, got, core.ExitCodeSuccess)
		}
	}
}
