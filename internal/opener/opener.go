package opener

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/thomascarr/fr8/internal/flock"
)

// Opener defines a named command for opening a workspace in an external tool.
type Opener struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Default bool   `json:"default,omitempty"`
}

// DefaultPath returns the default openers config path (~/.config/fr8/openers.json).
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".config", "fr8", "openers.json"), nil
}

// Load reads the opener list from path. Returns an empty slice if the file doesn't exist.
func Load(path string) ([]Opener, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading openers: %w", err)
	}
	var openers []Opener
	if err := json.Unmarshal(data, &openers); err != nil {
		return nil, fmt.Errorf("parsing openers: %w", err)
	}
	return openers, nil
}

// Save writes the opener list to path.
// Uses advisory file locking to prevent concurrent modifications.
func Save(path string, openers []Opener) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating openers directory: %w", err)
	}

	data, err := json.MarshalIndent(openers, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling openers: %w", err)
	}
	data = append(data, '\n')

	f, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("creating lock file: %w", err)
	}
	defer f.Close()
	defer os.Remove(path + ".lock")

	if err := flock.Lock(f.Fd()); err != nil {
		return fmt.Errorf("acquiring lock: %w", err)
	}
	defer flock.Unlock(f.Fd())

	return os.WriteFile(path, data, 0644)
}

// Find returns the opener with the given name, or nil.
func Find(openers []Opener, name string) *Opener {
	for i := range openers {
		if openers[i].Name == name {
			return &openers[i]
		}
	}
	return nil
}

// FindDefault returns the opener marked as default, or nil if none.
func FindDefault(openers []Opener) *Opener {
	for i := range openers {
		if openers[i].Default {
			return &openers[i]
		}
	}
	return nil
}

// SetDefault marks the named opener as default and clears the flag on all others.
func SetDefault(openers []Opener, name string) error {
	found := false
	for i := range openers {
		if openers[i].Name == name {
			openers[i].Default = true
			found = true
		} else {
			openers[i].Default = false
		}
	}
	if !found {
		return fmt.Errorf("opener %q not found (see available: fr8 opener list)", name)
	}
	return nil
}

// Run resolves the opener's command to an executable and opens the workspace path.
// The Command field may contain arguments (e.g. "code --new-window").
// Returns an error if the executable is not found in $PATH.
func Run(o Opener, workspacePath string) error {
	parts := strings.Fields(o.Command)
	if len(parts) == 0 {
		return fmt.Errorf("opener %q has an empty command", o.Name)
	}
	binPath, err := exec.LookPath(parts[0])
	if err != nil {
		return fmt.Errorf("%s: executable not found in $PATH (check that it is installed and on your PATH)", parts[0])
	}
	args := append(parts[1:], workspacePath)
	cmd := exec.Command(binPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}
