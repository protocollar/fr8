package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/thomascarr/fr8/internal/git"
	"github.com/thomascarr/fr8/internal/state"
	"github.com/thomascarr/fr8/internal/workspace"
)

// resolveWorkspace tries CWD-based resolution, falling back to global registry lookup.
// If --repo is set, resolves directly from that registered repo.
// Returns (workspace, rootPath, commonDir, error).
func resolveWorkspace(name string) (*state.Workspace, string, string, error) {
	// If --repo is specified, bypass CWD and resolve from that repo
	if resolveRepo != "" {
		if name == "" {
			return nil, "", "", fmt.Errorf("workspace name is required when using --repo")
		}
		ws, rootPath, commonDir, err := workspace.ResolveFromRepo(name, resolveRepo)
		if err != nil {
			return nil, "", "", err
		}
		return ws, rootPath, commonDir, nil
	}

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

	ws, rootPath, commonDir, err := workspace.ResolveGlobal(name)
	if err != nil {
		return nil, "", "", err
	}

	fmt.Fprintf(os.Stderr, "(resolved from repo %q)\n", filepath.Base(rootPath))
	return ws, rootPath, commonDir, nil
}
