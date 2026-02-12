package cmd

import (
	"fmt"
	"os"

	"github.com/thomascarr/fr8/internal/git"
	"github.com/thomascarr/fr8/internal/state"
	"github.com/thomascarr/fr8/internal/workspace"
)

// resolveWorkspace tries CWD-based resolution, falling back to global registry lookup.
// Returns (workspace, rootPath, commonDir, error).
func resolveWorkspace(name string) (*state.Workspace, string, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", "", err
	}

	commonDir, cdErr := git.CommonDir(cwd)
	if cdErr == nil {
		// Inside a git repo — use local state
		st, err := state.Load(commonDir)
		if err != nil {
			return nil, "", "", fmt.Errorf("loading state: %w", err)
		}

		ws, err := workspace.Resolve(name, st)
		if err != nil {
			return nil, "", "", err
		}

		rootPath, err := git.RootWorktreePath(cwd)
		if err != nil {
			return nil, "", "", fmt.Errorf("finding root worktree: %w", err)
		}

		return ws, rootPath, commonDir, nil
	}

	// Not inside a git repo — try global registry if a name was given
	if name == "" {
		return nil, "", "", fmt.Errorf("not inside a git repository (specify a workspace name or run from inside a repo)")
	}

	return workspace.ResolveGlobal(name)
}
