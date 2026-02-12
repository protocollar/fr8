package registry

import (
	"os"
	"path/filepath"
	"testing"
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
	r.Add(Repo{Name: "myapp", Path: "/home/user/myapp"})
	r.Add(Repo{Name: "other", Path: "/home/user/other"})

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
	r.Add(Repo{Name: "myapp", Path: "/home/user/myapp"})

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
	r.Add(Repo{Name: "alpha", Path: "/alpha"})
	r.Add(Repo{Name: "beta", Path: "/beta"})

	names := r.Names()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("unexpected names: %v", names)
	}
}

func TestDefaultPath(t *testing.T) {
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", "fr8", "repos.json")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}
