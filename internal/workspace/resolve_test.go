package workspace

import (
	"testing"

	"github.com/protocollar/fr8/internal/registry"
)

func TestResolveByName(t *testing.T) {
	repo := &registry.Repo{
		Name: "test",
		Path: "/tmp/repo",
		Workspaces: []registry.Workspace{
			{Name: "alpha", Path: "/tmp/alpha"},
			{Name: "beta", Path: "/tmp/beta"},
		},
	}

	ws, err := Resolve("beta", repo)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Name != "beta" {
		t.Errorf("Name = %q, want beta", ws.Name)
	}
}

func TestResolveByNameNotFound(t *testing.T) {
	repo := &registry.Repo{Name: "test", Path: "/tmp/repo"}

	_, err := Resolve("nonexistent", repo)
	if err == nil {
		t.Fatal("expected error for nonexistent workspace")
	}
}
