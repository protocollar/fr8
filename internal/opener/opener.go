package opener

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// Opener defines a named command for opening a workspace in an external tool.
type Opener struct {
	Name    string `json:"name"`
	Command string `json:"command"`
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

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("acquiring lock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

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

// Run resolves the opener's command to an executable and opens the workspace path.
// Returns an error if the executable is not found in $PATH.
func Run(o Opener, workspacePath string) error {
	binPath, err := exec.LookPath(o.Command)
	if err != nil {
		return fmt.Errorf("%s: executable not found in $PATH", o.Command)
	}
	cmd := exec.Command(binPath, workspacePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}
