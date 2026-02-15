package workspace

import (
	"fmt"
	"os"
	"strings"

	"github.com/protocollar/fr8/internal/git"
	"github.com/protocollar/fr8/internal/registry"
)

// Resolve finds a workspace by name within a specific repo.
// If name is empty, uses CWD to find the matching workspace.
func Resolve(name string, repo *registry.Repo) (*registry.Workspace, error) {
	if name != "" {
		ws := repo.FindWorkspace(name)
		if ws == nil {
			return nil, fmt.Errorf("workspace %q not found (see available: fr8 ws list)", name)
		}
		return ws, nil
	}

	// Auto-detect from CWD
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}

	ws := repo.FindWorkspaceByPath(cwd)
	if ws == nil {
		return nil, fmt.Errorf("not inside a managed workspace (run from a workspace directory or specify a name)")
	}
	return ws, nil
}

// ResolveGlobal searches all registered repos for a workspace by name.
// Returns an error listing the matching repos if more than one is found.
func ResolveGlobal(name string) (*registry.Workspace, *registry.Repo, string, error) {
	regPath, err := registry.DefaultPath()
	if err != nil {
		return nil, nil, "", err
	}

	reg, err := registry.Load(regPath)
	if err != nil {
		return nil, nil, "", fmt.Errorf("loading registry: %w", err)
	}

	ws, repo, err := reg.FindWorkspaceGlobal(name)
	if err != nil {
		// Reformat multi-repo ambiguity error with suggestion
		if strings.Contains(err.Error(), "multiple repos") {
			return nil, nil, "", err
		}
		return nil, nil, "", fmt.Errorf("workspace %q not found in any registered repo (see repos: fr8 repo list)", name)
	}

	rootPath, err := git.RootWorktreePath(repo.Path)
	if err != nil {
		rootPath = repo.Path
	}

	return ws, repo, rootPath, nil
}

// ResolveFromRepo resolves a workspace by name from a specific registered repo.
func ResolveFromRepo(name, repoName string) (*registry.Workspace, *registry.Repo, string, error) {
	regPath, err := registry.DefaultPath()
	if err != nil {
		return nil, nil, "", err
	}

	reg, err := registry.Load(regPath)
	if err != nil {
		return nil, nil, "", fmt.Errorf("loading registry: %w", err)
	}

	repo := reg.Find(repoName)
	if repo == nil {
		return nil, nil, "", fmt.Errorf("repo %q not found in registry (see: fr8 repo list)", repoName)
	}

	ws := repo.FindWorkspace(name)
	if ws == nil {
		return nil, nil, "", fmt.Errorf("workspace %q not found in repo %q (see available: fr8 ws list --repo %s)", name, repoName, repoName)
	}

	rootPath, err := git.RootWorktreePath(repo.Path)
	if err != nil {
		rootPath = repo.Path
	}

	return ws, repo, rootPath, nil
}
