package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddDropLoad(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "src"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	abs := filepath.Join(repo, "src", "main.go")

	added, err := Add(repo, []string{"./src/main.go", abs, "src/*.go"})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if len(added.Entries) != 2 {
		t.Fatalf("expected deduped entries, got %v", added.Entries)
	}

	loaded, err := Load(repo)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded.Entries) != 2 {
		t.Fatalf("unexpected entries: %v", loaded.Entries)
	}

	dropped, err := Drop(repo, []string{"src/*.go"})
	if err != nil {
		t.Fatalf("drop: %v", err)
	}
	if len(dropped.Entries) != 1 || dropped.Entries[0] != "src/main.go" {
		t.Fatalf("unexpected drop result: %v", dropped.Entries)
	}
}
