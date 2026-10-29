package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpVersionAndUsage(t *testing.T) {
	for _, tc := range []struct {
		args     []string
		code     int
		contains string
	}{
		{nil, 0, "hexgogen new"}, {[]string{"--help"}, 0, "hexgogen init"}, {[]string{"version"}, 0, "hexgogen dev"},
		{[]string{"unknown"}, 2, "unknown command"}, {[]string{"new"}, 2, "usage:"},
		{[]string{"init", "unexpected"}, 2, "usage:"}, {[]string{"new", "app", "--module"}, 2, "requires a value"},
		{[]string{"new", "app", "--module="}, 2, "nonempty"}, {[]string{"new", "app", "--unknown"}, 2, "unknown option"},
		{[]string{"new", "--help"}, 0, "hexgogen new"},
	} {
		var out, stderr bytes.Buffer
		if code := Run(tc.args, &out, &stderr); code != tc.code {
			t.Fatalf("%v: code=%d stderr=%s", tc.args, code, stderr.String())
		}
		if !strings.Contains(out.String()+stderr.String(), tc.contains) {
			t.Fatalf("%v: unexpected output: %s%s", tc.args, out.String(), stderr.String())
		}
	}
}

func TestNewWithFlagsAfterProject(t *testing.T) {
	parent := t.TempDir()
	var out, stderr bytes.Buffer
	code := Run([]string{"new", "myapp", "--module", "example.com/team/myapp", "--dir", parent}, &out, &stderr)
	if code != 0 {
		t.Fatalf("%d: %s", code, stderr.String())
	}
	module, err := os.ReadFile(filepath.Join(parent, "myapp", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(module), "module example.com/team/myapp") {
		t.Fatalf("unexpected module: %s", module)
	}
	if code := Run([]string{"new", "myapp", "--dir", parent}, &out, &stderr); code == 0 {
		t.Fatal("existing target was overwritten")
	}
}

func TestNewWithFlagsBeforeProjectAndDefaultModule(t *testing.T) {
	parent := t.TempDir()
	var out, stderr bytes.Buffer
	if code := Run([]string{"new", "--dir=" + parent, "myapp"}, &out, &stderr); code != 0 {
		t.Fatal(stderr.String())
	}
	module, err := os.ReadFile(filepath.Join(parent, "myapp", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(module), "module myapp\n") {
		t.Fatalf("unexpected module: %s", module)
	}
}

func TestInitAndRefusal(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "existing")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	if code := Run([]string{"init", "--dir", target, "--module=example.com/team/existing"}, &out, &stderr); code != 0 {
		t.Fatal(stderr.String())
	}
	if code := Run([]string{"init", "--dir", target}, &out, &stderr); code == 0 {
		t.Fatal("nonempty init target accepted")
	}
}
