package workspace

import (
	"testing"

	"github.com/protocollar/fr8/internal/state"
)

func TestResolveByName(t *testing.T) {
	st := &state.State{
		Workspaces: []state.Workspace{
			{Name: "alpha", Path: "/tmp/alpha"},
			{Name: "beta", Path: "/tmp/beta"},
		},
	}

	ws, err := Resolve("beta", st)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Name != "beta" {
		t.Errorf("Name = %q, want beta", ws.Name)
	}
}

func TestResolveByNameNotFound(t *testing.T) {
	st := &state.State{}

	_, err := Resolve("nonexistent", st)
	if err == nil {
		t.Fatal("expected error for nonexistent workspace")
	}
}
