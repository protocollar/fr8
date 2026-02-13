package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thomascarr/fr8/internal/flock"
)

// Repo is a registered repository.
type Repo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Registry holds all registered repositories.
type Registry struct {
	Repos []Repo `json:"repos"`
}

// DefaultPath returns the default registry file path (~/.config/fr8/repos.json).
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".config", "fr8", "repos.json"), nil
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
