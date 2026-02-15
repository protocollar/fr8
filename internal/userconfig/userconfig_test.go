package userconfig

import (
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	c, err := Load("/nonexistent/path/config.json")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(c.Openers) != 0 {
		t.Fatalf("expected empty config, got %d openers", len(c.Openers))
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	c := &Config{
		Openers: []Opener{
			{Name: "vscode", Command: "code", Default: true},
			{Name: "cursor", Command: "cursor"},
		},
	}

	if err := c.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded.Openers) != 2 {
		t.Fatalf("expected 2 openers, got %d", len(loaded.Openers))
	}
	if loaded.Openers[0].Name != "vscode" || !loaded.Openers[0].Default {
		t.Errorf("unexpected first opener: %+v", loaded.Openers[0])
	}
}

func TestOpenerCRUD(t *testing.T) {
	c := &Config{}

	// Add
	if err := c.AddOpener(Opener{Name: "vscode", Command: "code"}); err != nil {
		t.Fatalf("AddOpener: %v", err)
	}

	// Find
	if o := c.FindOpener("vscode"); o == nil {
		t.Fatal("expected to find vscode")
	}

	// Duplicate add
	if err := c.AddOpener(Opener{Name: "vscode", Command: "code"}); err == nil {
		t.Fatal("expected error for duplicate")
	}

	// Set default
	if err := c.SetDefaultOpener("vscode"); err != nil {
		t.Fatalf("SetDefaultOpener: %v", err)
	}
	if d := c.FindDefaultOpener(); d == nil || d.Name != "vscode" {
		t.Error("expected vscode as default")
	}

	// Names
	if names := c.OpenerNames(); len(names) != 1 || names[0] != "vscode" {
		t.Errorf("OpenerNames = %v, want [vscode]", names)
	}

	// Remove
	if err := c.RemoveOpener("vscode"); err != nil {
		t.Fatalf("RemoveOpener: %v", err)
	}
	if c.FindOpener("vscode") != nil {
		t.Error("expected vscode to be removed")
	}

	// Remove nonexistent
	if err := c.RemoveOpener("nonexistent"); err == nil {
		t.Error("expected error for nonexistent opener")
	}
}

func TestSetDefaultOpenerNotFound(t *testing.T) {
	c := &Config{}
	if err := c.SetDefaultOpener("nonexistent"); err == nil {
		t.Error("expected error for nonexistent opener")
	}
}

func TestFindDefaultOpenerNone(t *testing.T) {
	c := &Config{Openers: []Opener{{Name: "vscode", Command: "code"}}}
	if c.FindDefaultOpener() != nil {
		t.Error("expected nil when no default is set")
	}
}

