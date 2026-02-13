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
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

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

func TestFindDefault(t *testing.T) {
	openers := []Opener{
		{Name: "vscode", Command: "code"},
		{Name: "cursor", Command: "cursor", Default: true},
	}

	d := FindDefault(openers)
	if d == nil {
		t.Fatal("expected to find default")
	}
	if d.Name != "cursor" {
		t.Errorf("default = %q, want cursor", d.Name)
	}
}

func TestFindDefaultNone(t *testing.T) {
	openers := []Opener{
		{Name: "vscode", Command: "code"},
		{Name: "cursor", Command: "cursor"},
	}

	if d := FindDefault(openers); d != nil {
		t.Errorf("expected nil, got %+v", d)
	}
}

func TestSetDefault(t *testing.T) {
	openers := []Opener{
		{Name: "vscode", Command: "code", Default: true},
		{Name: "cursor", Command: "cursor"},
	}

	if err := SetDefault(openers, "cursor"); err != nil {
		t.Fatal(err)
	}

	if openers[0].Default {
		t.Error("vscode should not be default")
	}
	if !openers[1].Default {
		t.Error("cursor should be default")
	}
}

func TestSetDefaultNotFound(t *testing.T) {
	openers := []Opener{
		{Name: "vscode", Command: "code"},
	}

	err := SetDefault(openers, "missing")
	if err == nil {
		t.Fatal("expected error for missing opener")
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

func TestSaveAndLoadPreservesDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openers.json")

	openers := []Opener{
		{Name: "vscode", Command: "code"},
		{Name: "cursor", Command: "cursor", Default: true},
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
	if loaded[0].Default {
		t.Error("vscode should not be default after roundtrip")
	}
	if !loaded[1].Default {
		t.Error("cursor should be default after roundtrip")
	}
}

func TestRunMultiWordCommand(t *testing.T) {
	// Use "echo" which exists everywhere — the multi-word command should
	// split correctly and pass extra args before the workspace path.
	o := Opener{Name: "echo-test", Command: "echo --flag extra"}
	err := Run(o, "/tmp/workspace")
	if err != nil {
		t.Fatalf("Run with multi-word command: %v", err)
	}
	// echo starts and exits immediately — no cleanup needed.
	// The key assertion is that Run() didn't error, meaning it correctly
	// split "echo --flag extra" into ["echo", "--flag", "extra"] and appended
	// the workspace path.
}

func TestRunEmptyCommand(t *testing.T) {
	o := Opener{Name: "empty", Command: ""}
	err := Run(o, "/tmp")
	if err == nil {
		t.Fatal("expected error for empty command")
	}
	if !strings.Contains(err.Error(), "empty command") {
		t.Errorf("error = %q, want it to mention 'empty command'", err.Error())
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
