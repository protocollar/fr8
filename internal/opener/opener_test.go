package opener

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadNonExistent(t *testing.T) {
	openers, err := Load("/nonexistent/path/openers.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if openers != nil {
		t.Errorf("expected nil, got %v", openers)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openers.json")

	openers := []Opener{
		{Name: "vscode", Command: "code"},
		{Name: "cursor", Command: "cursor"},
	}

	if err := Save(path, openers); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("loaded %d openers, want 2", len(loaded))
	}
	if loaded[0].Name != "vscode" || loaded[0].Command != "code" {
		t.Errorf("opener[0] = %+v, want vscode/code", loaded[0])
	}
	if loaded[1].Name != "cursor" || loaded[1].Command != "cursor" {
		t.Errorf("opener[1] = %+v, want cursor", loaded[1])
	}
}

func TestSaveCreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "openers.json")

	if err := Save(path, []Opener{{Name: "test", Command: "echo"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected file to exist after Save")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openers.json")
	os.WriteFile(path, []byte("not json"), 0644)

	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestFind(t *testing.T) {
	openers := []Opener{
		{Name: "vscode", Command: "code"},
		{Name: "cursor", Command: "cursor"},
	}

	if o := Find(openers, "vscode"); o == nil {
		t.Error("expected to find vscode")
	} else if o.Command != "code" {
		t.Errorf("command = %q, want %q", o.Command, "code")
	}

	if o := Find(openers, "cursor"); o == nil {
		t.Error("expected to find cursor")
	}

	if o := Find(openers, "missing"); o != nil {
		t.Errorf("expected nil for missing opener, got %+v", o)
	}

	if o := Find(nil, "vscode"); o != nil {
		t.Errorf("expected nil for nil slice, got %+v", o)
	}
}

func TestRunMissingExecutable(t *testing.T) {
	o := Opener{Name: "fake", Command: "fr8_nonexistent_binary_xyz"}
	err := Run(o, "/tmp")
	if err == nil {
		t.Fatal("expected error for missing executable")
	}
	if !strings.Contains(err.Error(), "executable not found") {
		t.Errorf("error = %q, want it to mention 'executable not found'", err.Error())
	}
}

func TestSaveEmptySlice(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openers.json")

	if err := Save(path, []Opener{}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 0 {
		t.Errorf("expected empty slice, got %d items", len(loaded))
	}
}
