package generator

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewRendersProjectAndDotfiles(t *testing.T) {
	parent := t.TempDir()
	destination, err := New(parent, "sample-api", "example.com/acme/sample-api")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"go.mod", "go.sum", ".env.example", ".gitignore", ".github/workflows/ci.yml", "cmd/server/main.go", "compose.yaml", "migrations/000001_create_users.up.sql", "README.md"} {
		if _, err := os.Stat(filepath.Join(destination, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	mod, err := os.ReadFile(filepath.Join(destination, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(mod), "module example.com/acme/sample-api\n") {
		t.Fatalf("unexpected go.mod: %s", mod)
	}
	readme, err := os.ReadFile(filepath.Join(destination, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(readme, []byte("# sample-api\n")) {
		t.Fatal("project name was not rendered")
	}
	if err := filepath.WalkDir(destination, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		if bytes.Contains(data, []byte(templateModule)) || bytes.Contains(data, []byte("{{PROJECT_NAME}}")) {
			t.Errorf("unrendered placeholder in %s", name)
		}
		if strings.HasSuffix(name, ".tmpl") {
			t.Errorf("template suffix retained in %s", name)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestNewRefusesExistingTarget(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "sample")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := New(parent, "sample", "sample"); err == nil {
		t.Fatal("existing empty target accepted")
	}
	if err := os.WriteFile(filepath.Join(target, "keep.txt"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(parent, "sample", "sample"); err == nil {
		t.Fatal("existing nonempty target accepted")
	}
	data, err := os.ReadFile(filepath.Join(target, "keep.txt"))
	if err != nil || string(data) != "keep" {
		t.Fatal("existing content changed")
	}
}

func TestInitRequiresEmptyRealDirectory(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "app")
		if err := os.Mkdir(target, 0755); err != nil {
			t.Fatal(err)
		}
		if _, err := Init(target, "app"); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("dotfile", func(t *testing.T) {
		target := t.TempDir()
		name := filepath.Join(target, ".env")
		if err := os.WriteFile(name, []byte("secret"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Init(target, "app"); err == nil {
			t.Fatal("nonempty target accepted")
		}
		entries, err := os.ReadDir(target)
		if err != nil || len(entries) != 1 {
			t.Fatal("files were added on rejection")
		}
		data, err := os.ReadFile(name)
		if err != nil || string(data) != "secret" {
			t.Fatal("existing dotfile changed")
		}
	})
	t.Run("symlink", func(t *testing.T) {
		parent := t.TempDir()
		target := filepath.Join(parent, "real")
		if err := os.Mkdir(target, 0755); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(parent, "link")
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if _, err := Init(link, "app"); err == nil {
			t.Fatal("symlink target accepted")
		}
		entries, err := os.ReadDir(target)
		if err != nil || len(entries) != 0 {
			t.Fatal("symlink destination changed")
		}
	})
}

func TestInvalidInputsWriteNothing(t *testing.T) {
	for _, name := range []string{"", ".", "..", "../escape", "/absolute", "foo/bar", `foo\bar`, "bad name", "-bad", "name.", "NUL", "com1.txt"} {
		t.Run("name="+name, func(t *testing.T) {
			parent := t.TempDir()
			if _, err := New(parent, name, "example.com/app"); err == nil {
				t.Fatalf("invalid name %q accepted", name)
			}
			entries, err := os.ReadDir(parent)
			if err != nil || len(entries) != 0 {
				t.Fatal("invalid input wrote files")
			}
		})
	}
	for _, module := range []string{"", "../bad", "example.com/../bad", "https://example.com/app", "example.com//app", "example.com/app/", "bad module", `example.com\app`, "example.com/app@v1", "example.com/app/v1", "example.com/app/v01", "example.com/app/v2.1", "example.com/NUL", "Example.com/app", "std"} {
		t.Run("module="+module, func(t *testing.T) {
			parent := t.TempDir()
			if _, err := New(parent, "sample", module); err == nil {
				t.Fatalf("invalid module %q accepted", module)
			}
			entries, err := os.ReadDir(parent)
			if err != nil || len(entries) != 0 {
				t.Fatal("invalid module wrote files")
			}
		})
	}
}

func TestPopulateRollsBackOnlyCreatedFiles(t *testing.T) {
	target := t.TempDir()
	// The second file deliberately collides with the first, simulating a late write failure.
	err := populate(target, []renderedFile{{name: "nested/a.txt", data: []byte("one")}, {name: "nested/a.txt", data: []byte("two")}})
	if err == nil {
		t.Fatal("collision accepted")
	}
	entries, err := os.ReadDir(target)
	if err != nil || len(entries) != 0 {
		t.Fatalf("partial output was not cleaned: %v %v", entries, err)
	}
}

func TestPopulateCannotEscapeDestination(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "target")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	if err := populate(target, []renderedFile{{name: "../escaped.txt", data: []byte("bad")}}); err == nil {
		t.Fatal("escaping path accepted")
	}
	if _, err := os.Stat(filepath.Join(parent, "escaped.txt")); !os.IsNotExist(err) {
		t.Fatalf("outside file created: %v", err)
	}
}

func goCommand(t *testing.T, directory string, integration bool, args ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "go", args...)
	command.Dir = directory
	command.Env = append(os.Environ(), "GOWORK=off")
	if !integration {
		command.Env = append(command.Env, "TEST_DATABASE_URL=")
	}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	t.Logf("go %s\n%s", strings.Join(args, " "), output)
}

func TestGeneratedProjectChecks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping generated application checks in short mode")
	}
	destination, err := New(t.TempDir(), "sample-api", "example.com/acme/sample-api")
	if err != nil {
		t.Fatal(err)
	}
	goCommand(t, destination, false, "vet", "./...")
	goCommand(t, destination, false, "test", "-race", "./...")
	goCommand(t, destination, false, "build", "-o", filepath.Join(t.TempDir(), "server"), "./cmd/server")
	// A default local module name should work without a hosted module path, too.
	local, err := New(t.TempDir(), "localapp", "localapp")
	if err != nil {
		t.Fatal(err)
	}
	goCommand(t, local, false, "build", "-o", filepath.Join(t.TempDir(), "local-server"), "./cmd/server")
}

func TestGeneratedPostgresIntegration(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("set TEST_DATABASE_URL to test the generated app against PostgreSQL")
	}
	destination, err := New(t.TempDir(), "integration-api", "example.com/acme/integration-api")
	if err != nil {
		t.Fatal(err)
	}
	goCommand(t, destination, true, "test", "-race", "-count=1", "./internal/adapter/db/...")
}

func TestInitAllowsOnlyGitMetadata(t *testing.T) {
	for _, file := range []bool{false, true} {
		parent := t.TempDir()
		project := filepath.Join(parent, "app")
		if err := os.Mkdir(project, 0755); err != nil {
			t.Fatal(err)
		}
		metadata := filepath.Join(project, ".git")
		if file {
			if err := os.WriteFile(metadata, []byte("gitdir: /elsewhere"), 0600); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := os.Mkdir(metadata, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(metadata, "config"), []byte("preserve"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := Init(project, "app"); err != nil {
			t.Fatal(err)
		}
		if file {
			data, err := os.ReadFile(metadata)
			if err != nil || string(data) != "gitdir: /elsewhere" {
				t.Fatal("Git metadata file changed")
			}
		} else {
			data, err := os.ReadFile(filepath.Join(metadata, "config"))
			if err != nil || string(data) != "preserve" {
				t.Fatal("Git metadata directory changed")
			}
		}
	}
}
