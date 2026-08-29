package tui

import (
	"flag"
	"io"
	"testing"

	"github.com/krewire/libs/vein"
)

func successHandler(*flag.FlagSet) vein.ExitCode {
	return vein.ExitCodeSuccess
}

func testApp() *App {
	app := NewApp("demo", "0.1.0").
		Command(NewCommand("ping", "ping the app", nil, successHandler))
	app.stderr = io.Discard
	return app
}

func TestRunDispatchesToKnownCommand(t *testing.T) {
	app := testApp()
	if got := app.Run([]string{"ping"}); got != vein.ExitCodeSuccess {
		t.Errorf("Run() = %v, want %v", got, vein.ExitCodeSuccess)
	}
}

func TestRunEmptyIsUsage(t *testing.T) {
	app := testApp()
	if got := app.Run(nil); got != vein.ExitCodeUsage {
		t.Errorf("Run() = %v, want %v", got, vein.ExitCodeUsage)
	}
}

func TestRunUnknownCommandIsUsage(t *testing.T) {
	app := testApp()
	if got := app.Run([]string{"nope"}); got != vein.ExitCodeUsage {
		t.Errorf("Run() = %v, want %v", got, vein.ExitCodeUsage)
	}
}

func TestRunParseErrorIsUsage(t *testing.T) {
	app := NewApp("demo", "0.1.0").
		Command(NewCommand("ping", "ping the app", func(fs *flag.FlagSet) {
			fs.String("name", "", "name of the target")
		}, successHandler))
	app.stderr = io.Discard
	if got := app.Run([]string{"ping", "--nope"}); got != vein.ExitCodeUsage {
		t.Errorf("Run() = %v, want %v", got, vein.ExitCodeUsage)
	}
}

func TestRunHelpGeneral(t *testing.T) {
	app := testApp()
	app.stderr = io.Discard
	if got := app.Run([]string{"help"}); got != vein.ExitCodeSuccess {
		t.Errorf("Run(help) = %v, want %v", got, vein.ExitCodeSuccess)
	}
	if got := app.Run([]string{"--help"}); got != vein.ExitCodeSuccess {
		t.Errorf("Run(--help) = %v, want %v", got, vein.ExitCodeSuccess)
	}
	if got := app.Run([]string{"-h"}); got != vein.ExitCodeSuccess {
		t.Errorf("Run(-h) = %v, want %v", got, vein.ExitCodeSuccess)
	}
}

func TestRunHelpCommand(t *testing.T) {
	app := testApp()
	app.stderr = io.Discard
	if got := app.Run([]string{"help", "ping"}); got != vein.ExitCodeSuccess {
		t.Errorf("Run(help ping) = %v, want %v", got, vein.ExitCodeSuccess)
	}
	if got := app.Run([]string{"help", "unknown"}); got != vein.ExitCodeUsage {
		t.Errorf("Run(help unknown) = %v, want %v", got, vein.ExitCodeUsage)
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
		if got := app.Run(args); got != vein.ExitCodeSuccess {
			t.Errorf("Run(%v) = %v, want %v", args, got, vein.ExitCodeSuccess)
		}
	}
}
