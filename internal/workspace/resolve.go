package workspace

import (
	"fmt"
	"os"

	"github.com/thomascarr/fr8/internal/git"
	"github.com/thomascarr/fr8/internal/registry"
	"github.com/thomascarr/fr8/internal/state"
)

// Resolve finds a workspace by name, or detects it from the current directory.
// If name is empty, uses CWD to find the matching workspace.
func Resolve(name string, st *state.State) (*state.Workspace, error) {
	if name != "" {
		ws := st.Find(name)
		if ws == nil {
			return nil, fmt.Errorf("workspace %q not found", name)
		}
		return ws, nil
	}

	// Auto-detect from CWD
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}

	ws := st.FindByPath(cwd)
	if ws == nil {
		return nil, fmt.Errorf("not inside a managed workspace (run from a workspace directory or specify a name)")
	}
	return ws, nil
}

// ResolveGlobal searches all registered repos for a workspace by name.
// Returns the workspace, the repo root path, and the git common dir.
func ResolveGlobal(name string) (*state.Workspace, string, string, error) {
	regPath, err := registry.DefaultPath()
	if err != nil {
		return nil, "", "", err
	}

	reg, err := registry.Load(regPath)
	if err != nil {
		return nil, "", "", fmt.Errorf("loading registry: %w", err)
	}

	for _, repo := range reg.Repos {
		commonDir, err := git.CommonDir(repo.Path)
		if err != nil {
			continue
		}

		st, err := state.Load(commonDir)
		if err != nil {
			continue
		}

		ws := st.Find(name)
		if ws != nil {
			rootPath, err := git.RootWorktreePath(repo.Path)
			if err != nil {
				rootPath = repo.Path
			}
			return ws, rootPath, commonDir, nil
		}
	}

	return nil, "", "", fmt.Errorf("workspace %q not found in any registered repo", name)
}
