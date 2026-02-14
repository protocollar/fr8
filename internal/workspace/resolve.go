package workspace

import (
	"fmt"
	"os"
	"strings"

	"github.com/protocollar/fr8/internal/git"
	"github.com/protocollar/fr8/internal/registry"
	"github.com/protocollar/fr8/internal/state"
)

// Resolve finds a workspace by name, or detects it from the current directory.
// If name is empty, uses CWD to find the matching workspace.
func Resolve(name string, st *state.State) (*state.Workspace, error) {
	if name != "" {
		ws := st.Find(name)
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

	ws := st.FindByPath(cwd)
	if ws == nil {
		return nil, fmt.Errorf("not inside a managed workspace (run from a workspace directory or specify a name)")
	}
	return ws, nil
}

// globalMatch holds a workspace match found during global resolution.
type globalMatch struct {
	Workspace *state.Workspace
	RootPath  string
	CommonDir string
	RepoName  string
}

// ResolveGlobal searches all registered repos for a workspace by name.
// Returns an error listing the matching repos if more than one is found.
func ResolveGlobal(name string) (*state.Workspace, string, string, error) {
	regPath, err := registry.DefaultPath()
	if err != nil {
		return nil, "", "", err
	}

	reg, err := registry.Load(regPath)
	if err != nil {
		return nil, "", "", fmt.Errorf("loading registry: %w", err)
	}

	var matches []globalMatch

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
			matches = append(matches, globalMatch{
				Workspace: ws,
				RootPath:  rootPath,
				CommonDir: commonDir,
				RepoName:  repo.Name,
			})
		}
	}

	switch len(matches) {
	case 0:
		return nil, "", "", fmt.Errorf("workspace %q not found in any registered repo (see repos: fr8 repo list)", name)
	case 1:
		m := matches[0]
		return m.Workspace, m.RootPath, m.CommonDir, nil
	default:
		var repoNames []string
		for _, m := range matches {
			repoNames = append(repoNames, m.RepoName)
		}
		return nil, "", "", fmt.Errorf(
			"workspace %q found in multiple repos: %s\nUse --repo to disambiguate: fr8 ws <cmd> --repo <reponame> %s",
			name, strings.Join(repoNames, ", "), name,
		)
	}
}

// ResolveFromRepo resolves a workspace by name from a specific registered repo.
func ResolveFromRepo(name, repoName string) (*state.Workspace, string, string, error) {
	regPath, err := registry.DefaultPath()
	if err != nil {
		return nil, "", "", err
	}

	reg, err := registry.Load(regPath)
	if err != nil {
		return nil, "", "", fmt.Errorf("loading registry: %w", err)
	}

	repo := reg.Find(repoName)
	if repo == nil {
		return nil, "", "", fmt.Errorf("repo %q not found in registry (see: fr8 repo list)", repoName)
	}

	commonDir, err := git.CommonDir(repo.Path)
	if err != nil {
		return nil, "", "", fmt.Errorf("reading git data for %s: %w", repoName, err)
	}

	st, err := state.Load(commonDir)
	if err != nil {
		return nil, "", "", fmt.Errorf("loading state for %s: %w", repoName, err)
	}

	ws := st.Find(name)
	if ws == nil {
		return nil, "", "", fmt.Errorf("workspace %q not found in repo %q (see available: fr8 ws list --repo %s)", name, repoName, repoName)
	}

	rootPath, err := git.RootWorktreePath(repo.Path)
	if err != nil {
		rootPath = repo.Path
	}

	return ws, rootPath, commonDir, nil
}
