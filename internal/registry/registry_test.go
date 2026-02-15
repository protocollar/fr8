package registry

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadMissingFile(t *testing.T) {
	r, err := Load("/nonexistent/path/repos.json")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(r.Repos) != 0 {
		t.Fatalf("expected empty registry, got %d repos", len(r.Repos))
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fr8", "repos.json")

	r := &Registry{
		Repos: []Repo{
			{Name: "myapp", Path: "/home/user/myapp"},
			{Name: "other", Path: "/home/user/other"},
		},
	}

	if err := r.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded.Repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(loaded.Repos))
	}
	if loaded.Repos[0].Name != "myapp" || loaded.Repos[0].Path != "/home/user/myapp" {
		t.Errorf("unexpected first repo: %+v", loaded.Repos[0])
	}
	if loaded.Repos[1].Name != "other" || loaded.Repos[1].Path != "/home/user/other" {
		t.Errorf("unexpected second repo: %+v", loaded.Repos[1])
	}
}

func TestAddAndFind(t *testing.T) {
	r := &Registry{}

	if err := r.Add(Repo{Name: "myapp", Path: "/home/user/myapp"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	found := r.Find("myapp")
	if found == nil {
		t.Fatal("expected to find myapp")
	}
	if found.Path != "/home/user/myapp" {
		t.Errorf("expected path /home/user/myapp, got %s", found.Path)
	}

	if r.Find("nonexistent") != nil {
		t.Error("expected nil for nonexistent repo")
	}
}

func TestAddDuplicateName(t *testing.T) {
	r := &Registry{}

	if err := r.Add(Repo{Name: "myapp", Path: "/home/user/myapp"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	err := r.Add(Repo{Name: "myapp", Path: "/home/user/other"})
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}
}

func TestAddDuplicatePath(t *testing.T) {
	r := &Registry{}

	if err := r.Add(Repo{Name: "myapp", Path: "/home/user/myapp"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	err := r.Add(Repo{Name: "other", Path: "/home/user/myapp"})
	if err == nil {
		t.Fatal("expected error for duplicate path")
	}
}

func TestRemove(t *testing.T) {
	r := &Registry{}
	if err := r.Add(Repo{Name: "myapp", Path: "/home/user/myapp"}); err != nil {
		t.Fatal(err)
	}
	if err := r.Add(Repo{Name: "other", Path: "/home/user/other"}); err != nil {
		t.Fatal(err)
	}

	if err := r.Remove("myapp"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if r.Find("myapp") != nil {
		t.Error("expected myapp to be removed")
	}
	if r.Find("other") == nil {
		t.Error("expected other to still exist")
	}

	if err := r.Remove("nonexistent"); err == nil {
		t.Error("expected error for nonexistent repo")
	}
}

func TestFindByPath(t *testing.T) {
	r := &Registry{}
	if err := r.Add(Repo{Name: "myapp", Path: "/home/user/myapp"}); err != nil {
		t.Fatal(err)
	}

	found := r.FindByPath("/home/user/myapp")
	if found == nil {
		t.Fatal("expected to find repo by path")
	}
	if found.Name != "myapp" {
		t.Errorf("expected name myapp, got %s", found.Name)
	}

	if r.FindByPath("/home/user/other") != nil {
		t.Error("expected nil for nonexistent path")
	}
}

func TestNames(t *testing.T) {
	r := &Registry{}
	if err := r.Add(Repo{Name: "alpha", Path: "/alpha"}); err != nil {
		t.Fatal(err)
	}
	if err := r.Add(Repo{Name: "beta", Path: "/beta"}); err != nil {
		t.Fatal(err)
	}

	names := r.Names()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("unexpected names: %v", names)
	}
}

// --- DefaultPath and ConfigDir tests ---

func TestDefaultPath(t *testing.T) {
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".local", "state", "fr8", "repos.json")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestDefaultPathEnvOverride(t *testing.T) {
	t.Setenv("FR8_STATE_DIR", "/tmp/custom-state")
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	expected := "/tmp/custom-state/repos.json"
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestConfigDir(t *testing.T) {
	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", "fr8")
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestConfigDirEnvOverride(t *testing.T) {
	t.Setenv("FR8_CONFIG_DIR", "/tmp/custom-config")
	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	if dir != "/tmp/custom-config" {
		t.Errorf("expected /tmp/custom-config, got %s", dir)
	}
}

// --- Workspace CRUD tests ---

func TestWorkspaceCRUD(t *testing.T) {
	repo := &Repo{Name: "myapp", Path: "/home/user/myapp"}

	ws := Workspace{Name: "feature-1", Path: "/tmp/ws/feature-1", Port: 5000, CreatedAt: time.Now()}
	if err := repo.AddWorkspace(ws); err != nil {
		t.Fatalf("AddWorkspace: %v", err)
	}

	found := repo.FindWorkspace("feature-1")
	if found == nil {
		t.Fatal("expected to find workspace")
	}
	if found.Port != 5000 {
		t.Errorf("Port = %d, want 5000", found.Port)
	}

	// Duplicate add
	if err := repo.AddWorkspace(ws); err == nil {
		t.Fatal("expected error for duplicate workspace name")
	}

	// Remove
	if err := repo.RemoveWorkspace("feature-1"); err != nil {
		t.Fatalf("RemoveWorkspace: %v", err)
	}
	if repo.FindWorkspace("feature-1") != nil {
		t.Error("expected workspace to be removed")
	}

	// Remove nonexistent
	if err := repo.RemoveWorkspace("nonexistent"); err == nil {
		t.Error("expected error for nonexistent workspace")
	}
}

func TestWorkspaceRename(t *testing.T) {
	repo := &Repo{Name: "myapp", Path: "/home/user/myapp"}
	if err := repo.AddWorkspace(Workspace{Name: "alpha"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddWorkspace(Workspace{Name: "beta"}); err != nil {
		t.Fatal(err)
	}

	if err := repo.RenameWorkspace("alpha", "gamma"); err != nil {
		t.Fatal(err)
	}
	if repo.FindWorkspace("alpha") != nil {
		t.Error("expected alpha to be gone")
	}
	if repo.FindWorkspace("gamma") == nil {
		t.Error("expected gamma to exist")
	}

	// Same name
	if err := repo.RenameWorkspace("gamma", "gamma"); err == nil {
		t.Error("expected error for same name")
	}
	// Duplicate
	if err := repo.RenameWorkspace("gamma", "beta"); err == nil {
		t.Error("expected error for duplicate name")
	}
	// Not found
	if err := repo.RenameWorkspace("nonexistent", "foo"); err == nil {
		t.Error("expected error for nonexistent workspace")
	}
}

func TestWorkspaceNames(t *testing.T) {
	repo := &Repo{Name: "myapp", Path: "/home/user/myapp"}
	if err := repo.AddWorkspace(Workspace{Name: "alpha"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddWorkspace(Workspace{Name: "beta"}); err != nil {
		t.Fatal(err)
	}

	names := repo.WorkspaceNames()
	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("WorkspaceNames = %v, want [alpha beta]", names)
	}
}

func TestAllocatedPorts(t *testing.T) {
	repo := &Repo{Name: "myapp", Path: "/home/user/myapp"}
	if err := repo.AddWorkspace(Workspace{Name: "a", Port: 5000}); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddWorkspace(Workspace{Name: "b", Port: 5010}); err != nil {
		t.Fatal(err)
	}

	ports := repo.AllocatedPorts()
	if len(ports) != 2 || ports[0] != 5000 || ports[1] != 5010 {
		t.Errorf("AllocatedPorts = %v, want [5000 5010]", ports)
	}
}

func TestFindWorkspaceByPath(t *testing.T) {
	repo := &Repo{Name: "myapp", Path: "/home/user/myapp"}
	if err := repo.AddWorkspace(Workspace{Name: "ws1", Path: "/tmp/workspaces/ws1"}); err != nil {
		t.Fatal(err)
	}

	// Exact match
	if ws := repo.FindWorkspaceByPath("/tmp/workspaces/ws1"); ws == nil || ws.Name != "ws1" {
		t.Error("expected to find ws1 by exact path")
	}
	// Subdirectory
	if ws := repo.FindWorkspaceByPath("/tmp/workspaces/ws1/app/models"); ws == nil || ws.Name != "ws1" {
		t.Error("expected to find ws1 by subdirectory")
	}
	// No match
	if repo.FindWorkspaceByPath("/tmp/other") != nil {
		t.Error("expected nil for non-matching path")
	}
	// False positive (ws10 should not match ws1)
	if repo.FindWorkspaceByPath("/tmp/workspaces/ws10") != nil {
		t.Error("expected nil — ws10 should not match ws1 prefix")
	}
}

func TestAllAllocatedPorts(t *testing.T) {
	r := &Registry{
		Repos: []Repo{
			{Name: "a", Path: "/a", Workspaces: []Workspace{{Name: "w1", Port: 5000}, {Name: "w2", Port: 5010}}},
			{Name: "b", Path: "/b", Workspaces: []Workspace{{Name: "w3", Port: 6000}}},
		},
	}
	ports := r.AllAllocatedPorts()
	if len(ports) != 3 {
		t.Fatalf("expected 3 ports, got %d", len(ports))
	}
}

func TestAllWorkspaceNames(t *testing.T) {
	r := &Registry{
		Repos: []Repo{
			{Name: "a", Path: "/a", Workspaces: []Workspace{{Name: "w1"}, {Name: "w2"}}},
			{Name: "b", Path: "/b", Workspaces: []Workspace{{Name: "w3"}}},
		},
	}
	names := r.AllWorkspaceNames()
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}
}

func TestFindWorkspaceGlobal(t *testing.T) {
	r := &Registry{
		Repos: []Repo{
			{Name: "a", Path: "/a", Workspaces: []Workspace{{Name: "unique"}}},
			{Name: "b", Path: "/b", Workspaces: []Workspace{{Name: "other"}}},
		},
	}

	ws, repo, err := r.FindWorkspaceGlobal("unique")
	if err != nil {
		t.Fatal(err)
	}
	if ws.Name != "unique" || repo.Name != "a" {
		t.Errorf("got ws=%q repo=%q, want unique/a", ws.Name, repo.Name)
	}

	// Not found
	_, _, err = r.FindWorkspaceGlobal("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent workspace")
	}
}

func TestFindWorkspaceGlobalAmbiguous(t *testing.T) {
	r := &Registry{
		Repos: []Repo{
			{Name: "a", Path: "/a", Workspaces: []Workspace{{Name: "shared"}}},
			{Name: "b", Path: "/b", Workspaces: []Workspace{{Name: "shared"}}},
		},
	}

	_, _, err := r.FindWorkspaceGlobal("shared")
	if err == nil {
		t.Fatal("expected error for ambiguous workspace")
	}
}

func TestFindRepoByWorkspacePath(t *testing.T) {
	r := &Registry{
		Repos: []Repo{
			{Name: "a", Path: "/a", Workspaces: []Workspace{{Name: "w1", Path: "/tmp/ws/w1"}}},
			{Name: "b", Path: "/b", Workspaces: []Workspace{{Name: "w2", Path: "/tmp/ws/w2"}}},
		},
	}

	repo := r.FindRepoByWorkspacePath("/tmp/ws/w1/app")
	if repo == nil || repo.Name != "a" {
		t.Error("expected to find repo a")
	}

	if r.FindRepoByWorkspacePath("/tmp/other") != nil {
		t.Error("expected nil for non-matching path")
	}
}

func TestSaveAndLoadWithWorkspaces(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "repos.json")

	now := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	r := &Registry{
		Repos: []Repo{
			{
				Name: "myapp",
				Path: "/home/user/myapp",
				Workspaces: []Workspace{
					{Name: "ws1", Path: "/tmp/ws1", Port: 5000, CreatedAt: now},
				},
			},
		},
	}

	if err := r.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(loaded.Repos))
	}
	repo := loaded.Repos[0]
	if len(repo.Workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(repo.Workspaces))
	}
	ws := repo.Workspaces[0]
	if ws.Name != "ws1" || ws.Port != 5000 {
		t.Errorf("workspace = %+v, want ws1/5000", ws)
	}
}
