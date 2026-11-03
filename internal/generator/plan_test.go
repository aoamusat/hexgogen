package generator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestPlanMatchesGeneratedFilesWithoutCreatingDestination(t *testing.T) {
	parent := t.TempDir()
	plan, err := PlanNew(parent, "sample", "example.com/sample")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(plan.Destination()); !os.IsNotExist(err) {
		t.Fatalf("planning created a destination: %v", err)
	}
	names := plan.Files()
	if len(names) == 0 {
		t.Fatal("empty generation plan")
	}
	// Callers cannot redirect generation by modifying the preview's file list.
	other := plan.Files()
	other[0] = "../escape"
	destination, err := plan.Generate()
	if err != nil {
		t.Fatal(err)
	}
	var actual []string
	if err := filepath.WalkDir(destination, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(destination, name)
		if err != nil {
			return err
		}
		actual = append(actual, filepath.ToSlash(relative))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(names, actual) {
		t.Fatalf("preview differs from generated files:\n%v\n%v", names, actual)
	}
}

func TestPlanRechecksDestinationBeforeWriting(t *testing.T) {
	t.Run("new destination appeared", func(t *testing.T) {
		parent := t.TempDir()
		plan, err := PlanNew(parent, "sample", "sample")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(plan.Destination(), 0755); err != nil {
			t.Fatal(err)
		}
		keep := filepath.Join(plan.Destination(), "keep.txt")
		if err := os.WriteFile(keep, []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := plan.Generate(); err == nil {
			t.Fatal("new destination conflict was ignored")
		}
		data, err := os.ReadFile(keep)
		if err != nil || string(data) != "keep" {
			t.Fatal("existing destination was modified")
		}
	})
	t.Run("init file appeared", func(t *testing.T) {
		parent := t.TempDir()
		directory := filepath.Join(parent, "app")
		if err := os.Mkdir(directory, 0755); err != nil {
			t.Fatal(err)
		}
		plan, err := PlanInit(directory, "app")
		if err != nil {
			t.Fatal(err)
		}
		keep := filepath.Join(directory, "keep.txt")
		if err := os.WriteFile(keep, []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := plan.Generate(); err == nil {
			t.Fatal("new file conflict was ignored")
		}
		entries, err := os.ReadDir(directory)
		if err != nil || len(entries) != 1 {
			t.Fatal("application files were written despite conflict")
		}
	})
}

func TestPlanRejectsMissingParentWithoutCreatingIt(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "missing")
	if _, err := PlanNew(parent, "sample", "sample"); err == nil {
		t.Fatal("missing parent accepted")
	}
	if _, err := os.Stat(parent); !os.IsNotExist(err) {
		t.Fatal("planning created the parent")
	}
}

func TestGeneratedDomainHasNoInfrastructureMappings(t *testing.T) {
	destination, err := New(t.TempDir(), "sample", "example.com/sample")
	if err != nil {
		t.Fatal(err)
	}
	err = filepath.WalkDir(filepath.Join(destination, "domain"), func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(name, ".go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			imported, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			if strings.HasPrefix(imported, "gorm.io/") || strings.HasPrefix(imported, "github.com/gin-") || strings.Contains(imported, "/internal/adapter/") {
				t.Errorf("domain file %s imports infrastructure %s", name, imported)
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			field, ok := node.(*ast.Field)
			if !ok || field.Tag == nil {
				return true
			}
			if strings.Contains(field.Tag.Value, `gorm:`) {
				t.Errorf("domain file %s contains an ORM mapping", name)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
