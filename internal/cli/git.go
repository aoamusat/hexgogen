package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// initializeGit preserves existing repositories, including parent repositories
// and worktrees. Generation itself succeeds even when this optional step fails.
func initializeGit(directory string) error {
	git, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("Git is not installed; install it and run git init in %s", directory)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	probe := gitCommand(ctx, git, directory, "rev-parse", "--is-inside-work-tree")
	output, err := probe.Output()
	if err == nil {
		if strings.TrimSpace(string(output)) == "true" {
			return nil
		}
		return fmt.Errorf("%s is inside a bare Git repository; leaving its Git configuration unchanged", directory)
	}
	// Do not replace metadata when discovery failed for a damaged or
	// inaccessible destination. A .git file may refer to a worktree elsewhere.
	_, statErr := os.Lstat(filepath.Join(directory, ".git"))
	if statErr == nil {
		return fmt.Errorf("existing Git metadata in %s could not be validated; run git status there", directory)
	}
	if !os.IsNotExist(statErr) {
		return fmt.Errorf("inspect existing Git metadata: %w", statErr)
	}
	var discoveryErr *exec.ExitError
	if !errors.As(err, &discoveryErr) || !strings.HasPrefix(strings.TrimSpace(string(discoveryErr.Stderr)), "fatal: not a git repository") {
		return fmt.Errorf("inspect Git repository in %s: %w", directory, err)
	}
	command := gitCommand(ctx, git, directory, "init", "--quiet")
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("git init in %s: %w: %s", directory, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func gitCommand(ctx context.Context, git, directory string, args ...string) *exec.Cmd {
	command := exec.CommandContext(ctx, git, append([]string{"-C", directory}, args...)...)
	// The destination, rather than repository variables inherited from a hook or
	// parent Git process, determines which repository is inspected or initialized.
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		switch key {
		case "GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE", "GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_PREFIX":
			continue
		}
		command.Env = append(command.Env, entry)
	}
	command.Env = append(command.Env, "LC_ALL=C")
	return command
}
