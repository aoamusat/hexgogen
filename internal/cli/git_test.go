package cli

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func requireGit(t *testing.T) string {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("Git is not installed")
	}
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_TEMPLATE_DIR", t.TempDir())
	return git
}

func runGit(t *testing.T, git, dir string, args ...string) string {
	t.Helper()
	command := gitCommand(context.Background(), git, dir, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func TestNewInitializesGitWithoutStaging(t *testing.T) {
	git := requireGit(t)
	parent := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"new", "app", "--dir", parent}, &stdout, &stderr); code != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	project := filepath.Join(parent, "app")
	if _, err := os.Stat(filepath.Join(project, ".git")); err != nil {
		t.Fatal("repository was not created:", err)
	}
	if got := runGit(t, git, project, "rev-parse", "--is-inside-work-tree"); got != "true" {
		t.Fatal(got)
	}
	if got := runGit(t, git, project, "ls-files"); got != "" {
		t.Fatal("generation staged files")
	}
}

func TestInitPreservesExistingRepository(t *testing.T) {
	git := requireGit(t)
	project := filepath.Join(t.TempDir(), "app")
	if err := os.Mkdir(project, 0755); err != nil {
		t.Fatal(err)
	}
	runGit(t, git, project, "init", "--quiet")
	runGit(t, git, project, "symbolic-ref", "HEAD", "refs/heads/custom-main")
	runGit(t, git, project, "config", "hexgogen.marker", "preserved")
	before, err := os.ReadFile(filepath.Join(project, ".git", "config"))
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"init", "--dir", project}, &stdout, &stderr); code != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	after, err := os.ReadFile(filepath.Join(project, ".git", "config"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("existing Git configuration changed")
	}
	if got := runGit(t, git, project, "symbolic-ref", "HEAD"); got != "refs/heads/custom-main" {
		t.Fatal("existing branch changed:", got)
	}
	if got := runGit(t, git, project, "ls-files"); got != "" {
		t.Fatal("generation staged files")
	}
}

func TestNewPreservesParentRepository(t *testing.T) {
	git := requireGit(t)
	parent := t.TempDir()
	runGit(t, git, parent, "init", "--quiet")
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"new", "app", "--dir", parent}, &stdout, &stderr); code != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	project := filepath.Join(parent, "app")
	if _, err := os.Lstat(filepath.Join(project, ".git")); !os.IsNotExist(err) {
		t.Fatal("created a nested repository")
	}
	if got := runGit(t, git, project, "rev-parse", "--show-prefix"); got != "app/" {
		t.Fatalf("parent repository was not retained: %q", got)
	}
}

func TestInitPreservesWorktree(t *testing.T) {
	git := requireGit(t)
	parent := t.TempDir()
	main := filepath.Join(parent, "main")
	if err := os.Mkdir(main, 0755); err != nil {
		t.Fatal(err)
	}
	runGit(t, git, main, "init", "--quiet")
	runGit(t, git, main, "-c", "user.name=Test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-m", "test fixture")
	project := filepath.Join(parent, "worktree")
	runGit(t, git, main, "worktree", "add", "--detach", project)
	before, err := os.ReadFile(filepath.Join(project, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"init", "--dir", project}, &stdout, &stderr); code != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	after, err := os.ReadFile(filepath.Join(project, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("worktree Git metadata changed")
	}
	if got := runGit(t, git, project, "rev-parse", "--is-inside-work-tree"); got != "true" {
		t.Fatal(got)
	}
}

func TestGitInitializationIgnoresInheritedRepositoryOverrides(t *testing.T) {
	git := requireGit(t)
	existing := t.TempDir()
	runGit(t, git, existing, "init", "--quiet")
	t.Setenv("GIT_DIR", filepath.Join(existing, ".git"))
	t.Setenv("GIT_WORK_TREE", existing)
	project := filepath.Join(t.TempDir(), "app")
	if err := os.Mkdir(project, 0755); err != nil {
		t.Fatal(err)
	}
	if err := initializeGit(project); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(project, ".git", "HEAD")); err != nil {
		t.Fatal("inherited Git variables redirected initialization:", err)
	}
}

func TestMissingGitWarnsAfterSuccessfulGeneration(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	parent := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"new", "app", "--dir", parent}, &stdout, &stderr); code != 0 {
		t.Fatalf("generation failed: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Git is not installed") {
		t.Fatalf("missing warning: %s", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(parent, "app", "go.mod")); err != nil {
		t.Fatal("project was not generated:", err)
	}
}

func TestInvalidGitMetadataIsPreserved(t *testing.T) {
	requireGit(t)
	project := t.TempDir()
	metadata := filepath.Join(project, ".git")
	if err := os.WriteFile(metadata, []byte("invalid metadata"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := initializeGit(project); err == nil {
		t.Fatal("invalid Git metadata was silently replaced")
	}
	data, err := os.ReadFile(metadata)
	if err != nil || string(data) != "invalid metadata" {
		t.Fatal("existing metadata changed")
	}
}
