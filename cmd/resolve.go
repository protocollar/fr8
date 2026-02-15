package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/protocollar/fr8/internal/git"
	"github.com/protocollar/fr8/internal/registry"
	"github.com/protocollar/fr8/internal/workspace"
)

// resolveWorkspace tries CWD-based resolution, falling back to global registry lookup.
// If --repo is set, resolves directly from that registered repo.
// Returns (workspace, rootPath, error).
func resolveWorkspace(name string) (*registry.Workspace, string, error) {
	// If --repo is specified, bypass CWD and resolve from that repo
	if resolveRepo != "" {
		if name == "" {
			return nil, "", fmt.Errorf("workspace name is required when using --repo")
		}
		ws, _, rootPath, err := workspace.ResolveFromRepo(name, resolveRepo)
		if err != nil {
			return nil, "", err
		}
		return ws, rootPath, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}

	// Load registry
	regPath, err := registry.DefaultPath()
	if err != nil {
		return nil, "", err
	}
	reg, err := registry.Load(regPath)
	if err != nil {
		return nil, "", fmt.Errorf("loading registry: %w", err)
	}

	// Try to find repo by CWD (workspace path match)
	repo := reg.FindRepoByWorkspacePath(cwd)
	if repo == nil {
		// Try rootPath match (CWD is inside a git repo registered in the registry)
		if git.IsInsideWorkTree(cwd) {
			rootPath, err := git.RootWorktreePath(cwd)
			if err == nil {
				repo = reg.FindByPath(rootPath)
			}
		}
	}

	if repo != nil {
		ws, err := workspace.Resolve(name, repo)
		if err != nil {
			return nil, "", err
		}
		rootPath, err := git.RootWorktreePath(repo.Path)
		if err != nil {
			rootPath = repo.Path
		}
		return ws, rootPath, nil
	}

	// Not found via CWD — try global if a name was given
	if name == "" {
		return nil, "", fmt.Errorf("not inside a git repository (specify a workspace name or run from inside a repo)")
	}

	ws, _, rootPath, err := workspace.ResolveGlobal(name)
	if err != nil {
		return nil, "", err
	}

	fmt.Fprintf(os.Stderr, "(resolved from repo %q)\n", filepath.Base(rootPath))
	return ws, rootPath, nil
}
