// Example CLI application demonstrating the Krewire CLI framework.
//
// Run with:
//
//	go run ./examples/greet hello --name Alice
//	GREET_GREETING=Halo go run ./examples/greet hello --name Alice
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/krewire/framework/tui"
	"github.com/krewire/libs/core"
)

func main() {
	app := tui.NewApp("greet", "0.1.0").
		Command(tui.NewCommand("hello", "greet a user", registerHello, hello))

	os.Exit(int(app.Run(os.Args[1:])))
}

func registerHello(fs *flag.FlagSet) {
	fs.String("name", "", "name of the user to greet")
}

func hello(fs *flag.FlagSet) core.ExitCode {
	name := fs.Lookup("name").Value.String()
	// KWF-FGNZ9: standardized <APP>_-prefixed, typed, defaulted config.
	greeting := tui.NewEnv("greet").GetString("greeting", "Hello")
	if name == "" {
		fmt.Fprintln(os.Stderr, "usage: greet hello --name <name>")
		return core.ExitCodeUsage
	}
	fmt.Printf("%s, %s!\n", greeting, name)
	return core.ExitCodeSuccess
}
