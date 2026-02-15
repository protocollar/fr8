package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/protocollar/fr8/internal/flock"
)

// Workspace represents a single managed worktree within a repo.
type Workspace struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Port      int       `json:"port"`
	CreatedAt time.Time `json:"created_at"`
}

// Repo is a registered repository.
type Repo struct {
	Name       string      `json:"name"`
	Path       string      `json:"path"`
	Workspaces []Workspace `json:"workspaces,omitempty"`
}

// Registry holds all registered repositories.
type Registry struct {
	Repos []Repo `json:"repos"`
}

// DefaultPath returns the path to the unified state file (~/.local/state/fr8/repos.json).
// Respects FR8_STATE_DIR to override the directory.
func DefaultPath() (string, error) {
	if dir := os.Getenv("FR8_STATE_DIR"); dir != "" {
		return filepath.Join(dir, "repos.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".local", "state", "fr8", "repos.json"), nil
}

// ConfigDir returns the fr8 config directory (~/.config/fr8).
// Respects FR8_CONFIG_DIR to override the directory.
func ConfigDir() (string, error) {
	if dir := os.Getenv("FR8_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".config", "fr8"), nil
}

// Load reads the registry from path. Returns an empty registry if the file doesn't exist.
func Load(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Registry{}, nil
		}
		return nil, fmt.Errorf("reading registry: %w", err)
	}
	var r Registry
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parsing registry: %w", err)
	}
	return &r, nil
}

// Save writes the registry to path.
// Uses advisory file locking to prevent concurrent modifications.
func (r *Registry) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating registry directory: %w", err)
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling registry: %w", err)
	}
	data = append(data, '\n')

	f, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("creating lock file: %w", err)
	}
	defer func() { _ = f.Close() }()
	defer func() { _ = os.Remove(path + ".lock") }()

	if err := flock.Lock(f.Fd()); err != nil {
		return fmt.Errorf("acquiring lock: %w", err)
	}
	defer func() { _ = flock.Unlock(f.Fd()) }()

	return os.WriteFile(path, data, 0644)
}

// Add appends a repo to the registry. Returns an error if the name already exists.
func (r *Registry) Add(repo Repo) error {
	if r.Find(repo.Name) != nil {
		return fmt.Errorf("repo %q already registered (use fr8 repo remove %s first to re-register)", repo.Name, repo.Name)
	}
	if r.FindByPath(repo.Path) != nil {
		return fmt.Errorf("path %q already registered", repo.Path)
	}
	r.Repos = append(r.Repos, repo)
	return nil
}

// Remove deletes a repo by name.
func (r *Registry) Remove(name string) error {
	for i, repo := range r.Repos {
		if repo.Name == name {
			r.Repos = append(r.Repos[:i], r.Repos[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("repo %q not found (see available: fr8 repo list)", name)
}

// Find returns the repo with the given name, or nil.
func (r *Registry) Find(name string) *Repo {
	for i := range r.Repos {
		if r.Repos[i].Name == name {
			return &r.Repos[i]
		}
	}
	return nil
}

// FindByPath returns the repo with the given path, or nil.
func (r *Registry) FindByPath(path string) *Repo {
	for i := range r.Repos {
		if r.Repos[i].Path == path {
			return &r.Repos[i]
		}
	}
	return nil
}

// Names returns all repo names.
func (r *Registry) Names() []string {
	names := make([]string, len(r.Repos))
	for i, repo := range r.Repos {
		names[i] = repo.Name
	}
	return names
}

// --- Workspace methods on Repo ---

// FindWorkspace returns the workspace with the given name, or nil.
func (r *Repo) FindWorkspace(name string) *Workspace {
	for i := range r.Workspaces {
		if r.Workspaces[i].Name == name {
			return &r.Workspaces[i]
		}
	}
	return nil
}

// FindWorkspaceByPath returns the workspace whose path matches or contains dir.
func (r *Repo) FindWorkspaceByPath(dir string) *Workspace {
	dir = filepath.Clean(dir)
	for i := range r.Workspaces {
		wsPath := filepath.Clean(r.Workspaces[i].Path)
		if dir == wsPath || strings.HasPrefix(dir, wsPath+string(filepath.Separator)) {
			return &r.Workspaces[i]
		}
	}
	return nil
}

// AddWorkspace appends a workspace. Returns an error if the name already exists.
func (r *Repo) AddWorkspace(ws Workspace) error {
	if r.FindWorkspace(ws.Name) != nil {
		return fmt.Errorf("workspace %q already exists", ws.Name)
	}
	r.Workspaces = append(r.Workspaces, ws)
	return nil
}

// RemoveWorkspace deletes a workspace by name.
func (r *Repo) RemoveWorkspace(name string) error {
	for i, ws := range r.Workspaces {
		if ws.Name == name {
			r.Workspaces = append(r.Workspaces[:i], r.Workspaces[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("workspace %q not found (see available: fr8 ws list)", name)
}

// RenameWorkspace changes a workspace's name. Returns an error if old doesn't exist or new already does.
func (r *Repo) RenameWorkspace(oldName, newName string) error {
	if oldName == newName {
		return fmt.Errorf("old and new names are the same")
	}
	if r.FindWorkspace(newName) != nil {
		return fmt.Errorf("workspace %q already exists", newName)
	}
	ws := r.FindWorkspace(oldName)
	if ws == nil {
		return fmt.Errorf("workspace %q not found (see available: fr8 ws list)", oldName)
	}
	ws.Name = newName
	return nil
}

// WorkspaceNames returns all workspace names in this repo.
func (r *Repo) WorkspaceNames() []string {
	names := make([]string, len(r.Workspaces))
	for i, ws := range r.Workspaces {
		names[i] = ws.Name
	}
	return names
}

// AllocatedPorts returns all ports currently allocated in this repo.
func (r *Repo) AllocatedPorts() []int {
	ports := make([]int, len(r.Workspaces))
	for i, ws := range r.Workspaces {
		ports[i] = ws.Port
	}
	return ports
}

// --- Global workspace methods on Registry ---

// AllAllocatedPorts returns every allocated port across all repos.
func (r *Registry) AllAllocatedPorts() []int {
	var ports []int
	for _, repo := range r.Repos {
		ports = append(ports, repo.AllocatedPorts()...)
	}
	return ports
}

// AllWorkspaceNames returns every workspace name across all repos.
func (r *Registry) AllWorkspaceNames() []string {
	var names []string
	for _, repo := range r.Repos {
		names = append(names, repo.WorkspaceNames()...)
	}
	return names
}

// globalMatch holds a workspace match found during global resolution.
type globalMatch struct {
	Workspace *Workspace
	Repo      *Repo
}

// FindWorkspaceGlobal searches all repos for a workspace by name.
// Returns an error if more than one repo contains a matching workspace.
func (r *Registry) FindWorkspaceGlobal(name string) (*Workspace, *Repo, error) {
	var matches []globalMatch
	for i := range r.Repos {
		ws := r.Repos[i].FindWorkspace(name)
		if ws != nil {
			matches = append(matches, globalMatch{
				Workspace: ws,
				Repo:      &r.Repos[i],
			})
		}
	}

	switch len(matches) {
	case 0:
		return nil, nil, fmt.Errorf("workspace %q not found in any registered repo (see repos: fr8 repo list)", name)
	case 1:
		return matches[0].Workspace, matches[0].Repo, nil
	default:
		var repoNames []string
		for _, m := range matches {
			repoNames = append(repoNames, m.Repo.Name)
		}
		return nil, nil, fmt.Errorf(
			"workspace %q found in multiple repos: %s\nUse --repo to disambiguate: fr8 ws <cmd> --repo <reponame> %s",
			name, strings.Join(repoNames, ", "), name,
		)
	}
}

// FindRepoByWorkspacePath walks all repos and returns the one containing a
// workspace whose path matches or contains dir.
func (r *Registry) FindRepoByWorkspacePath(dir string) *Repo {
	for i := range r.Repos {
		if r.Repos[i].FindWorkspaceByPath(dir) != nil {
			return &r.Repos[i]
		}
	}
	return nil
}
