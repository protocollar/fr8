package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const stateFile = "fr8.json"

// State holds all tracked workspaces for a repository.
type State struct {
	Workspaces []Workspace `json:"workspaces"`
}

// Workspace represents a single managed worktree.
type Workspace struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Branch    string    `json:"branch"`
	Port      int       `json:"port"`
	CreatedAt time.Time `json:"created_at"`
}

// Load reads the state file from the git common directory.
// Returns an empty state if the file doesn't exist.
func Load(gitCommonDir string) (*State, error) {
	p := statePath(gitCommonDir)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{}, nil
		}
		return nil, fmt.Errorf("reading state: %w", err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing state: %w", err)
	}
	return &s, nil
}

// Save writes the state file to the git common directory.
// Uses advisory file locking to prevent concurrent modifications.
func (s *State) Save(gitCommonDir string) error {
	p := statePath(gitCommonDir)
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}
	data = append(data, '\n')

	f, err := os.OpenFile(p+".lock", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("creating lock file: %w", err)
	}
	defer f.Close()
	defer os.Remove(p + ".lock")

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("acquiring lock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	return os.WriteFile(p, data, 0644)
}

// Add appends a workspace to the state. Returns an error if the name already exists.
func (s *State) Add(w Workspace) error {
	if s.Find(w.Name) != nil {
		return fmt.Errorf("workspace %q already exists", w.Name)
	}
	s.Workspaces = append(s.Workspaces, w)
	return nil
}

// Remove deletes a workspace by name.
func (s *State) Remove(name string) error {
	for i, w := range s.Workspaces {
		if w.Name == name {
			s.Workspaces = append(s.Workspaces[:i], s.Workspaces[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("workspace %q not found (see available: fr8 ws list)", name)
}

// Find returns the workspace with the given name, or nil.
func (s *State) Find(name string) *Workspace {
	for i := range s.Workspaces {
		if s.Workspaces[i].Name == name {
			return &s.Workspaces[i]
		}
	}
	return nil
}

// FindByPath returns the workspace whose path contains the given directory.
func (s *State) FindByPath(dir string) *Workspace {
	dir = filepath.Clean(dir)
	for i := range s.Workspaces {
		wsPath := filepath.Clean(s.Workspaces[i].Path)
		if dir == wsPath || strings.HasPrefix(dir, wsPath+string(filepath.Separator)) {
			return &s.Workspaces[i]
		}
	}
	return nil
}

// AllocatedPorts returns all ports currently allocated.
func (s *State) AllocatedPorts() []int {
	ports := make([]int, len(s.Workspaces))
	for i, w := range s.Workspaces {
		ports[i] = w.Port
	}
	return ports
}

// Names returns all workspace names.
func (s *State) Names() []string {
	names := make([]string, len(s.Workspaces))
	for i, w := range s.Workspaces {
		names[i] = w.Name
	}
	return names
}

// Rename changes a workspace's name. Returns an error if old doesn't exist or new already does.
func (s *State) Rename(oldName, newName string) error {
	if oldName == newName {
		return fmt.Errorf("old and new names are the same")
	}
	if s.Find(newName) != nil {
		return fmt.Errorf("workspace %q already exists", newName)
	}
	ws := s.Find(oldName)
	if ws == nil {
		return fmt.Errorf("workspace %q not found (see available: fr8 ws list)", oldName)
	}
	ws.Name = newName
	return nil
}

func statePath(gitCommonDir string) string {
	return filepath.Join(gitCommonDir, stateFile)
}
