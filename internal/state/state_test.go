package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAddAndFind(t *testing.T) {
	s := &State{}

	ws := Workspace{Name: "test-ws", Path: "/tmp/ws", Branch: "main", Port: 5000, CreatedAt: time.Now()}
	if err := s.Add(ws); err != nil {
		t.Fatal(err)
	}

	found := s.Find("test-ws")
	if found == nil {
		t.Fatal("expected to find workspace")
	}
	if found.Port != 5000 {
		t.Errorf("Port = %d, want 5000", found.Port)
	}
}

func TestAddDuplicate(t *testing.T) {
	s := &State{}
	ws := Workspace{Name: "test-ws"}
	s.Add(ws)

	err := s.Add(ws)
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}
}

func TestRemove(t *testing.T) {
	s := &State{}
	s.Add(Workspace{Name: "a"})
	s.Add(Workspace{Name: "b"})
	s.Add(Workspace{Name: "c"})

	if err := s.Remove("b"); err != nil {
		t.Fatal(err)
	}
	if len(s.Workspaces) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(s.Workspaces))
	}
	if s.Find("b") != nil {
		t.Error("expected b to be removed")
	}
	if s.Find("a") == nil || s.Find("c") == nil {
		t.Error("expected a and c to remain")
	}
}

func TestRemoveNotFound(t *testing.T) {
	s := &State{}
	err := s.Remove("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent workspace")
	}
}

func TestFindNil(t *testing.T) {
	s := &State{}
	if s.Find("nope") != nil {
		t.Error("expected nil for nonexistent workspace")
	}
}

func TestFindByPathExact(t *testing.T) {
	s := &State{
		Workspaces: []Workspace{
			{Name: "ws1", Path: "/tmp/workspaces/ws1"},
			{Name: "ws2", Path: "/tmp/workspaces/ws2"},
		},
	}

	found := s.FindByPath("/tmp/workspaces/ws2")
	if found == nil || found.Name != "ws2" {
		t.Errorf("FindByPath exact = %v, want ws2", found)
	}
}

func TestFindByPathSubdirectory(t *testing.T) {
	s := &State{
		Workspaces: []Workspace{
			{Name: "ws1", Path: "/tmp/workspaces/ws1"},
		},
	}

	found := s.FindByPath("/tmp/workspaces/ws1/app/models")
	if found == nil || found.Name != "ws1" {
		t.Errorf("FindByPath subdirectory = %v, want ws1", found)
	}
}

func TestFindByPathNoMatch(t *testing.T) {
	s := &State{
		Workspaces: []Workspace{
			{Name: "ws1", Path: "/tmp/workspaces/ws1"},
		},
	}

	if s.FindByPath("/tmp/other") != nil {
		t.Error("expected nil for non-matching path")
	}
}

func TestFindByPathPrefixFalsePositive(t *testing.T) {
	s := &State{
		Workspaces: []Workspace{
			{Name: "ws1", Path: "/tmp/workspaces/ws1"},
		},
	}

	// /tmp/workspaces/ws10 should NOT match /tmp/workspaces/ws1
	if s.FindByPath("/tmp/workspaces/ws10") != nil {
		t.Error("expected nil — ws10 should not match ws1 prefix")
	}
}

func TestAllocatedPorts(t *testing.T) {
	s := &State{
		Workspaces: []Workspace{
			{Name: "a", Port: 5000},
			{Name: "b", Port: 5010},
		},
	}

	ports := s.AllocatedPorts()
	if len(ports) != 2 || ports[0] != 5000 || ports[1] != 5010 {
		t.Errorf("AllocatedPorts = %v, want [5000 5010]", ports)
	}
}

func TestNames(t *testing.T) {
	s := &State{
		Workspaces: []Workspace{
			{Name: "alpha"},
			{Name: "beta"},
		},
	}

	names := s.Names()
	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("Names = %v, want [alpha beta]", names)
	}
}

func TestRename(t *testing.T) {
	s := &State{}
	s.Add(Workspace{Name: "alpha"})
	s.Add(Workspace{Name: "beta"})

	if err := s.Rename("alpha", "gamma"); err != nil {
		t.Fatal(err)
	}
	if s.Find("alpha") != nil {
		t.Error("expected alpha to be gone")
	}
	if s.Find("gamma") == nil {
		t.Error("expected gamma to exist")
	}
	if s.Find("beta") == nil {
		t.Error("expected beta to remain")
	}
}

func TestRenameNotFound(t *testing.T) {
	s := &State{}
	err := s.Rename("nonexistent", "new")
	if err == nil {
		t.Fatal("expected error for nonexistent workspace")
	}
}

func TestRenameAlreadyExists(t *testing.T) {
	s := &State{}
	s.Add(Workspace{Name: "alpha"})
	s.Add(Workspace{Name: "beta"})

	err := s.Rename("alpha", "beta")
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}
}

func TestRenameSameName(t *testing.T) {
	s := &State{}
	s.Add(Workspace{Name: "alpha"})

	err := s.Rename("alpha", "alpha")
	if err == nil {
		t.Fatal("expected error for same name")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC)

	original := &State{
		Workspaces: []Workspace{
			{Name: "ws1", Path: "/tmp/ws1", Branch: "main", Port: 5000, CreatedAt: now},
		},
	}

	if err := original.Save(dir); err != nil {
		t.Fatal(err)
	}

	// Verify file exists
	if _, err := os.Stat(filepath.Join(dir, "fr8.json")); err != nil {
		t.Fatal("state file not created")
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(loaded.Workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(loaded.Workspaces))
	}
	ws := loaded.Workspaces[0]
	if ws.Name != "ws1" || ws.Port != 5000 || ws.Branch != "main" {
		t.Errorf("loaded workspace = %+v, want ws1/5000/main", ws)
	}
}

func TestLoadMissing(t *testing.T) {
	s, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Workspaces) != 0 {
		t.Errorf("expected empty state, got %d workspaces", len(s.Workspaces))
	}
}

func TestLoadMalformed(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "fr8.json"), []byte(`{broken`), 0644)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}
