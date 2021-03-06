package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/constabulary/gb"
	"github.com/constabulary/gb/cmd"
)

type Command struct {
	ShortDesc string
	Run       func(ctx *gb.Context, args []string) error
	AddFlags  func(fs *flag.FlagSet)
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		gb.Fatalf("unable to determine current working directory: %v", err)
	}
	return wd
}

var (
	fs          = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	goroot      = fs.String("goroot", runtime.GOROOT(), "override GOROOT")
	projectroot string
)

func init() {
	fs.BoolVar(&gb.Quiet, "q", gb.Quiet, "suppress log messages below ERROR level")
	fs.BoolVar(&gb.Verbose, "v", gb.Verbose, "enable log levels below INFO level")
	fs.StringVar(&projectroot, "R", mustGetwd(), "set the project root")

	// TODO some flags are specific to a specific commands
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:")
		for name, cmd := range commands {
			fmt.Fprintf(os.Stderr, "  gb %s [flags] [package] - %s\n",
				name, cmd.ShortDesc)
		}
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Flags:")
		fs.PrintDefaults()
	}
}

var commands = make(map[string]*Command)

// registerCommand registers a command for main.
// registerCommand should only be called from init().
func registerCommand(name string, command *Command) {
	commands[name] = command
}

func main() {
	args := os.Args
	if len(args) < 2 || args[1] == "-h" {
		fs.Usage()
		os.Exit(1)
	}

	name := args[1]
	parseargs := name != "plugin"
	command, ok := commands[name]
	if !ok {
		if _, err := lookupPlugin(name); err != nil {
			gb.Errorf("unknown command %q", name)
			fs.Usage()
			os.Exit(1)
		}
		command = commands["plugin"]
		args = append([]string{"plugin"}, args...)
		parseargs = false // don't parse args as import paths
	}

	// add extra flags if necessary
	if command.AddFlags != nil {
		command.AddFlags(fs)
	}

	if err := fs.Parse(args[2:]); err != nil {
		gb.Fatalf("could not parse flags: %v", err)
	}
	args = fs.Args() // reset args to the leftovers from fs.Parse

	gopath := filepath.SplitList(os.Getenv("GOPATH"))
	root, err := cmd.FindProjectroot(projectroot, gopath)
	if err != nil {
		gb.Fatalf("could not locate project root: %v", err)
	}
	project := gb.NewProject(root)

	gb.Debugf("project root %q", project.Projectdir())

	ctx, err := project.NewContext(
		gb.GcToolchain(gb.Goroot(*goroot)),
	)
	if err != nil {
		gb.Fatalf("unable to construct context: %v", err)
	}

	if parseargs {
		args = cmd.ImportPaths(ctx, projectroot, args)
	}
	gb.Debugf("args: %v", args)
	if err := command.Run(ctx, args); err != nil {
		gb.Fatalf("command %q failed: %v", name, err)
	}
}
