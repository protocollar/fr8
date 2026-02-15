package userconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/protocollar/fr8/internal/flock"
	"github.com/protocollar/fr8/internal/registry"
)

// Opener defines a named command for opening a workspace in an external tool.
type Opener struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Default bool   `json:"default,omitempty"`
}

// Config holds user-level preferences stored in ~/.config/fr8/config.json.
type Config struct {
	Openers []Opener `json:"openers,omitempty"`
}

// DefaultPath returns the path to the user config file (~/.config/fr8/config.json).
func DefaultPath() (string, error) {
	dir, err := registry.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads the config from path. Returns an empty config if the file doesn't exist.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &c, nil
}

// Save writes the config to path.
// Uses advisory file locking to prevent concurrent modifications.
func (c *Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
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

// FindOpener returns the opener with the given name, or nil.
func (c *Config) FindOpener(name string) *Opener {
	for i := range c.Openers {
		if c.Openers[i].Name == name {
			return &c.Openers[i]
		}
	}
	return nil
}

// FindDefaultOpener returns the opener marked as default, or nil if none.
func (c *Config) FindDefaultOpener() *Opener {
	for i := range c.Openers {
		if c.Openers[i].Default {
			return &c.Openers[i]
		}
	}
	return nil
}

// AddOpener appends an opener. Returns an error if the name already exists.
func (c *Config) AddOpener(o Opener) error {
	if c.FindOpener(o.Name) != nil {
		return fmt.Errorf("opener %q already exists (remove it first with: fr8 opener remove %s)", o.Name, o.Name)
	}
	c.Openers = append(c.Openers, o)
	return nil
}

// RemoveOpener removes an opener by name.
func (c *Config) RemoveOpener(name string) error {
	for i, o := range c.Openers {
		if o.Name == name {
			c.Openers = append(c.Openers[:i], c.Openers[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("opener %q not found", name)
}

// SetDefaultOpener marks the named opener as default and clears the flag on all others.
func (c *Config) SetDefaultOpener(name string) error {
	found := false
	for i := range c.Openers {
		if c.Openers[i].Name == name {
			c.Openers[i].Default = true
			found = true
		} else {
			c.Openers[i].Default = false
		}
	}
	if !found {
		return fmt.Errorf("opener %q not found (see available: fr8 opener list)", name)
	}
	return nil
}

// OpenerNames returns all opener names.
func (c *Config) OpenerNames() []string {
	names := make([]string, len(c.Openers))
	for i, o := range c.Openers {
		names[i] = o.Name
	}
	return names
}

