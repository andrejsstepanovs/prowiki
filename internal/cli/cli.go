package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// App represents the CLI application configuration and state.
type App struct {
	Out io.Writer
	Err io.Writer
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{
		Out: os.Stdout,
		Err: os.Stderr,
	}
}

// Run executes the application logic based on command-line arguments.
func (a *App) Run(args []string) error {
	fs := flag.NewFlagSet("prowiki", flag.ContinueOnError)
	fs.SetOutput(a.Err)

	// Define basic flags here
	versionFlag := fs.Bool("version", false, "Print version information")

	// Set usage message
	fs.Usage = func() {
		fmt.Fprintf(a.Err, "Usage: prowiki [options] <command> [arguments]\n\n")
		fmt.Fprintf(a.Err, "Options:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *versionFlag {
		fmt.Fprintln(a.Out, "prowiki version 0.1.0")
		return nil
	}

	// Retrieve remaining arguments (after flags)
	cmdArgs := fs.Args()
	if len(cmdArgs) == 0 {
		fs.Usage()
		return fmt.Errorf("missing command")
	}

	command := cmdArgs[0]
	switch command {
	case "help":
		fs.Usage()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}
