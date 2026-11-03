package cli

import (
	"context"
	"fmt"
	"go/version"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aoamusat/hexgogen/templates"
)

type doctorCheck struct{ status, name, detail string }
type toolLookup func(string) (string, error)
type toolOutput func(string, ...string) (string, error)

func runDoctor(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintln(stdout, "Usage: hexgogen doctor\nChecks Go compatibility and optional Git, Make, Docker, and Compose tools.")
		return 0
	}
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: hexgogen doctor")
		return 2
	}
	return reportDoctor(stdout, inspectTools(exec.LookPath, installedToolOutput))
}

func reportDoctor(output io.Writer, checks []doctorCheck) int {
	failures, warnings := 0, 0
	for _, check := range checks {
		fmt.Fprintf(output, "%s %s: %s\n", check.status, check.name, check.detail)
		if check.status == "FAIL" {
			failures++
		}
		if check.status == "WARN" {
			warnings++
		}
	}
	if failures > 0 {
		fmt.Fprintf(output, "%d required check(s) failed; %d warning(s).\n", failures, warnings)
		return 1
	}
	fmt.Fprintf(output, "No required checks failed; %d warning(s).\n", warnings)
	return 0
}

func inspectTools(find toolLookup, run toolOutput) []doctorCheck {
	required, err := templateGoVersion()
	if err != nil {
		return []doctorCheck{{"FAIL", "Go requirement", err.Error()}}
	}
	var checks []doctorCheck
	goTool, err := find("go")
	if err != nil {
		checks = append(checks, doctorCheck{"FAIL", "Go", "not found; install Go " + strings.TrimPrefix(required, "go") + " or newer to build generated apps"})
	} else if output, err := run(goTool, "version"); err != nil {
		checks = append(checks, doctorCheck{"FAIL", "Go", "could not run go version: " + err.Error()})
	} else {
		fields := strings.Fields(output)
		switch {
		case len(fields) < 3 || fields[0] != "go" || fields[1] != "version":
			checks = append(checks, doctorCheck{"FAIL", "Go", "could not recognize go version output"})
		case fields[2] == "devel":
			checks = append(checks, doctorCheck{"WARN", "Go", "development toolchain detected; verify compatibility with " + required})
		case !version.IsValid(fields[2]):
			checks = append(checks, doctorCheck{"FAIL", "Go", "could not recognize toolchain version: " + fields[2]})
		case version.Compare(fields[2], required) < 0:
			checks = append(checks, doctorCheck{"FAIL", "Go", fields[2] + " is older than required " + required})
		default:
			checks = append(checks, doctorCheck{"PASS", "Go", fields[2] + " (requires " + required + " or newer)"})
		}
	}
	git, err := find("git")
	if err != nil {
		checks = append(checks, doctorCheck{"WARN", "Git", "not found; automatic repository initialization will be skipped"})
	} else if output, err := run(git, "--version"); err != nil {
		checks = append(checks, doctorCheck{"WARN", "Git", "could not run git --version"})
	} else {
		checks = append(checks, doctorCheck{"PASS", "Git", strings.TrimSpace(output)})
	}
	if _, err := find("make"); err != nil {
		checks = append(checks, doctorCheck{"WARN", "Make", "not found; install Make to use the generated setup and check commands"})
	} else {
		checks = append(checks, doctorCheck{"PASS", "Make", "executable found"})
	}
	docker, err := find("docker")
	if err != nil {
		checks = append(checks, doctorCheck{"WARN", "Docker", "not found; use an existing PostgreSQL server or install Docker"}, doctorCheck{"WARN", "Compose", "unavailable without Docker; needed for make db-up"})
	} else {
		if output, err := run(docker, "--version"); err != nil {
			checks = append(checks, doctorCheck{"WARN", "Docker", "could not run docker --version"})
		} else {
			checks = append(checks, doctorCheck{"PASS", "Docker", strings.TrimSpace(output)})
		}
		if output, err := run(docker, "compose", "version"); err != nil {
			checks = append(checks, doctorCheck{"WARN", "Compose", "not available; install the Docker Compose plugin to use make db-up"})
		} else {
			checks = append(checks, doctorCheck{"PASS", "Compose", strings.TrimSpace(output)})
		}
	}
	return checks
}

func templateGoVersion() (string, error) {
	data, err := templates.Files.ReadFile("go.mod.tmpl")
	if err != nil {
		return "", fmt.Errorf("read template go.mod: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "go" && version.IsValid("go"+fields[1]) {
			return "go" + fields[1], nil
		}
	}
	return "", fmt.Errorf("embedded template has no valid Go version requirement")
}

func installedToolOutput(tool string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, tool, args...)
	// Inspect the installed Go toolchain without triggering automatic downloads.
	command.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off", "GOTELEMETRY=off")
	output, err := command.CombinedOutput()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	return strings.TrimSpace(string(output)), err
}
