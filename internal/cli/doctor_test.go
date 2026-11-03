package cli

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestDoctorToolCompatibilityAndOptionalDependencies(t *testing.T) {
	for _, tc := range []struct {
		name, goVersion string
		missing         map[string]bool
		composeError    bool
		code            int
		expected        []string
	}{
		{name: "ready", goVersion: "go version go1.25.6 linux/amd64", code: 0, expected: []string{"PASS Go", "PASS Compose"}},
		{name: "newer", goVersion: "go version go1.26.0 linux/amd64", code: 0, expected: []string{"PASS Go"}},
		{name: "older patch", goVersion: "go version go1.25.5 linux/amd64", code: 1, expected: []string{"FAIL Go", "older than required"}},
		{name: "older minor", goVersion: "go version go1.24.9 linux/amd64", code: 1, expected: []string{"FAIL Go"}},
		{name: "missing Go", missing: map[string]bool{"go": true}, code: 1, expected: []string{"FAIL Go", "install Go"}},
		{name: "unrecognized Go", goVersion: "unexpected output", code: 1, expected: []string{"FAIL Go"}},
		{name: "development Go", goVersion: "go version devel go1.27-deadbeef linux/amd64", code: 0, expected: []string{"WARN Go", "development toolchain"}},
		{name: "optional tools missing", goVersion: "go version go1.25.6 linux/amd64", missing: map[string]bool{"git": true, "make": true, "docker": true}, code: 0, expected: []string{"PASS Go", "WARN Git", "WARN Make", "WARN Docker", "WARN Compose"}},
		{name: "Compose plugin missing", goVersion: "go version go1.25.6 linux/amd64", composeError: true, code: 0, expected: []string{"PASS Docker", "WARN Compose"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lookup := func(name string) (string, error) {
				if tc.missing[name] {
					return "", os.ErrNotExist
				}
				return name, nil
			}
			run := func(name string, args ...string) (string, error) {
				switch name {
				case "go":
					return tc.goVersion, nil
				case "git":
					return "git version 2.43.0", nil
				case "docker":
					if len(args) > 0 && args[0] == "compose" {
						if tc.composeError {
							return "", errors.New("plugin missing")
						}
						return "Docker Compose version v2.0.0", nil
					}
					return "Docker version 28.0.0", nil
				}
				t.Fatalf("unexpected command %s %v", name, args)
				return "", nil
			}
			var output bytes.Buffer
			if code := reportDoctor(&output, inspectTools(lookup, run)); code != tc.code {
				t.Fatalf("code=%d, want %d: %s", code, tc.code, output.String())
			}
			for _, expected := range tc.expected {
				if !strings.Contains(output.String(), expected) {
					t.Fatalf("missing %q: %s", expected, output.String())
				}
			}
		})
	}
}

func TestDoctorReportsBrokenGoExecutable(t *testing.T) {
	checks := inspectTools(func(name string) (string, error) {
		if name == "go" {
			return name, nil
		}
		return "", os.ErrNotExist
	}, func(string, ...string) (string, error) { return "", errors.New("command failed") })
	var output bytes.Buffer
	if code := reportDoctor(&output, checks); code != 1 || !strings.Contains(output.String(), "could not run go version") {
		t.Fatalf("code=%d: %s", code, output.String())
	}
}

func TestDoctorCommandMissingToolsAndHelp(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"doctor"}, &stdout, &stderr); code != 1 {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "FAIL Go") || stderr.Len() != 0 {
		t.Fatalf("unexpected doctor output: %s%s", stdout.String(), stderr.String())
	}
	stdout.Reset()
	if code := Run([]string{"doctor", "--help"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "Usage: hexgogen doctor") {
		t.Fatal("doctor help failed")
	}
	if code := Run([]string{"doctor", "--dry-run"}, &stdout, &stderr); code != 2 {
		t.Fatal("doctor accepted an unsupported option")
	}
}
