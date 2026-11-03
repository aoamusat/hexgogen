package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewDryRunPreviewsWithoutWritingOrRunningGit(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // A dry run must not even attempt Git initialization.
	parent := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"new", "sample", "--module", "example.com/sample", "--dir", parent, "--dry-run"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code=%d: %s", code, stderr.String())
	}
	for _, expected := range []string{"Would generate project", "Module: example.com/sample", "create go.mod", "create .env.example", "create cmd/server/main.go", "No files were written"} {
		if !strings.Contains(stdout.String(), expected) {
			t.Fatalf("missing %q: %s", expected, stdout.String())
		}
	}
	if stderr.Len() != 0 {
		t.Fatal("dry run attempted Git setup:", stderr.String())
	}
	entries, err := os.ReadDir(parent)
	if err != nil || len(entries) != 0 {
		t.Fatal("dry run changed the filesystem")
	}
}

func TestInitDryRunPreservesGitMetadata(t *testing.T) {
	parent := t.TempDir()
	project := filepath.Join(parent, "app")
	if err := os.Mkdir(project, 0755); err != nil {
		t.Fatal(err)
	}
	metadata := filepath.Join(project, ".git")
	if err := os.WriteFile(metadata, []byte("gitdir: /unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"init", "--dry-run", "--dir", project}, &stdout, &stderr); code != 0 {
		t.Fatalf("code=%d: %s", code, stderr.String())
	}
	entries, err := os.ReadDir(project)
	if err != nil || len(entries) != 1 {
		t.Fatal("dry run wrote application files")
	}
	data, err := os.ReadFile(metadata)
	if err != nil || string(data) != "gitdir: /unchanged" {
		t.Fatal("dry run changed Git metadata")
	}
	if stderr.Len() != 0 {
		t.Fatal("dry run attempted to inspect Git metadata:", stderr.String())
	}
}

func TestDryRunRejectsConflictsAndInvalidInput(t *testing.T) {
	parent := t.TempDir()
	project := filepath.Join(parent, "app")
	if err := os.Mkdir(project, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "keep.txt"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"new", "app", "--dir", parent, "--dry-run"},
		{"init", "--dir", project, "--dry-run"},
		{"new", "../escape", "--dir", parent, "--dry-run"},
		{"new", "other", "--module", "bad module", "--dir", parent, "--dry-run"},
	} {
		var stdout, stderr bytes.Buffer
		if code := Run(args, &stdout, &stderr); code != 1 || stderr.Len() == 0 {
			t.Fatalf("expected preflight error for %v: code=%d %s", args, code, stderr.String())
		}
	}
	data, err := os.ReadFile(filepath.Join(project, "keep.txt"))
	if err != nil || string(data) != "keep" {
		t.Fatal("conflicting file changed")
	}
	entries, err := os.ReadDir(parent)
	if err != nil || len(entries) != 1 {
		t.Fatal("rejected preview created files")
	}
}
