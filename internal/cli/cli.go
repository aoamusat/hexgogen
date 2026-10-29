// Package cli implements the command-line interface independently of os.Exit.
package cli

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/aoamusat/hexgogen/internal/generator"
)

// Version is set by release builds with -ldflags -X.
var Version = "dev"

const help = `Usage:
  hexgogen new <PROJECT> [--module MODULE] [--dir PARENT]
  hexgogen init [--module MODULE] [--dir DIRECTORY]
  hexgogen version

Create a Go API with Gin, PostgreSQL, Zerolog, migrations, and tests.
The module defaults to the project directory name. Use --module for a published path.
Generation works offline and never runs go mod, git, Docker, or migration commands.
New refuses existing targets. Init requires an existing, empty directory.
`

type options struct {
	module, dir string
	positional  []string
	help        bool
}

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprint(stdout, help)
		return 0
	}
	if args[0] == "version" || args[0] == "--version" {
		if len(args) != 1 {
			fmt.Fprintln(stderr, "version does not accept arguments")
			return 2
		}
		fmt.Fprintln(stdout, "hexgogen "+Version)
		return 0
	}
	command := args[0]
	if command != "new" && command != "init" {
		fmt.Fprintf(stderr, "unknown command %q\n%s", command, help)
		return 2
	}
	opts, err := parse(args[1:])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if opts.help {
		fmt.Fprint(stdout, help)
		return 0
	}
	var destination string
	switch command {
	case "new":
		if len(opts.positional) != 1 {
			fmt.Fprintln(stderr, "usage: hexgogen new <PROJECT> [--module MODULE] [--dir PARENT]")
			return 2
		}
		name := opts.positional[0]
		if opts.module == "" {
			opts.module = name
		}
		destination, err = generator.New(opts.dir, name, opts.module)
	case "init":
		if len(opts.positional) != 0 {
			fmt.Fprintln(stderr, "usage: hexgogen init [--module MODULE] [--dir DIRECTORY]")
			return 2
		}
		absolute, absErr := filepath.Abs(opts.dir)
		if absErr != nil {
			fmt.Fprintln(stderr, absErr)
			return 1
		}
		if opts.module == "" {
			opts.module = filepath.Base(absolute)
		}
		destination, err = generator.Init(absolute, opts.module)
	}
	if err != nil {
		fmt.Fprintln(stderr, "hexgogen:", err)
		return 1
	}
	fmt.Fprintf(stdout, "Created project at %s\nModule: %s\nSee README.md in the project for database setup, migrations, and running the server.\n", destination, opts.module)
	return 0
}

func parse(args []string) (options, error) {
	opts := options{dir: "."}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--help" || arg == "-h":
			opts.help = true
		case arg == "--module" || arg == "--dir":
			if i+1 == len(args) || strings.HasPrefix(args[i+1], "--") {
				return opts, fmt.Errorf("%s requires a value", arg)
			}
			i++
			if args[i] == "" {
				return opts, fmt.Errorf("%s requires a nonempty value", arg)
			}
			if arg == "--module" {
				opts.module = args[i]
			} else {
				opts.dir = args[i]
			}
		case strings.HasPrefix(arg, "--module="):
			opts.module = strings.TrimPrefix(arg, "--module=")
			if opts.module == "" {
				return opts, errors.New("--module requires a nonempty value")
			}
		case strings.HasPrefix(arg, "--dir="):
			opts.dir = strings.TrimPrefix(arg, "--dir=")
			if opts.dir == "" {
				return opts, errors.New("--dir requires a nonempty value")
			}
		case strings.HasPrefix(arg, "-"):
			return opts, fmt.Errorf("unknown option %q", arg)
		default:
			opts.positional = append(opts.positional, arg)
		}
	}
	return opts, nil
}
