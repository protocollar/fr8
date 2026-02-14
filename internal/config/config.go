package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config represents the fr8.json (or conductor.json) configuration.
type Config struct {
	Scripts      Scripts `json:"scripts"`
	PortRange    int     `json:"port_range"`
	BasePort     int     `json:"base_port"`
	WorktreePath string  `json:"worktree_path"`
}

// UnmarshalJSON supports both snake_case (preferred) and legacy camelCase keys.
func (c *Config) UnmarshalJSON(data []byte) error {
	// Decode into a raw map to handle both key styles.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if v, ok := raw["scripts"]; ok {
		if err := json.Unmarshal(v, &c.Scripts); err != nil {
			return fmt.Errorf("parsing scripts: %w", err)
		}
	}

	// port_range (preferred) or portRange (legacy)
	if v, ok := raw["port_range"]; ok {
		if err := json.Unmarshal(v, &c.PortRange); err != nil {
			return fmt.Errorf("parsing port_range: %w", err)
		}
	} else if v, ok := raw["portRange"]; ok {
		if err := json.Unmarshal(v, &c.PortRange); err != nil {
			return fmt.Errorf("parsing portRange: %w", err)
		}
	}

	// base_port (preferred) or basePort (legacy)
	if v, ok := raw["base_port"]; ok {
		if err := json.Unmarshal(v, &c.BasePort); err != nil {
			return fmt.Errorf("parsing base_port: %w", err)
		}
	} else if v, ok := raw["basePort"]; ok {
		if err := json.Unmarshal(v, &c.BasePort); err != nil {
			return fmt.Errorf("parsing basePort: %w", err)
		}
	}

	// worktree_path (preferred) or worktreePath (legacy)
	if v, ok := raw["worktree_path"]; ok {
		if err := json.Unmarshal(v, &c.WorktreePath); err != nil {
			return fmt.Errorf("parsing worktree_path: %w", err)
		}
	} else if v, ok := raw["worktreePath"]; ok {
		if err := json.Unmarshal(v, &c.WorktreePath); err != nil {
			return fmt.Errorf("parsing worktreePath: %w", err)
		}
	}

	return nil
}

// Scripts defines the lifecycle commands.
type Scripts struct {
	Setup   string `json:"setup"`
	Run     string `json:"run"`
	Archive string `json:"archive"`
}

// legacyKeys are the deprecated camelCase config keys and their snake_case replacements.
var legacyKeys = map[string]string{
	"portRange":    "port_range",
	"basePort":     "base_port",
	"worktreePath": "worktree_path",
}

// LegacyKeyReplacement returns the snake_case replacement for a legacy camelCase key.
func LegacyKeyReplacement(key string) string {
	return legacyKeys[key]
}

// HasLegacyKeys checks whether a config file at path uses deprecated camelCase keys.
// Returns the list of legacy keys found.
func HasLegacyKeys(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	var found []string
	for old := range legacyKeys {
		if _, ok := raw[old]; ok {
			found = append(found, old)
		}
	}
	return found
}

// MigrateKeys rewrites a config file, replacing deprecated camelCase keys with snake_case.
// Returns the list of keys that were migrated.
func MigrateKeys(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filepath.Base(path), err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", filepath.Base(path), err)
	}

	var migrated []string
	for old, new := range legacyKeys {
		v, ok := raw[old]
		if !ok {
			continue
		}
		// Only migrate if the new key doesn't already exist
		if _, exists := raw[new]; !exists {
			raw[new] = v
		}
		delete(raw, old)
		migrated = append(migrated, old)
	}

	if len(migrated) == 0 {
		return nil, nil
	}

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}
	out = append(out, '\n')

	if err := os.WriteFile(path, out, 0644); err != nil {
		return nil, fmt.Errorf("writing %s: %w", filepath.Base(path), err)
	}

	return migrated, nil
}

// Load reads fr8.json from rootPath, falling back to conductor.json.
// Returns config with defaults applied.
func Load(rootPath string) (*Config, error) {
	cfg := &Config{}

	for _, name := range []string{"fr8.json", "conductor.json"} {
		p := filepath.Join(rootPath, name)
		data, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("reading %s: %w", name, err)
		}
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", name, err)
		}
		applyDefaults(cfg)
		return cfg, nil
	}

	// No config file found — use defaults
	applyDefaults(cfg)
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.PortRange == 0 {
		cfg.PortRange = 10
	}
	if cfg.BasePort == 0 {
		cfg.BasePort = 8000
	}
	if cfg.WorktreePath == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			cfg.WorktreePath = filepath.Join(home, "fr8")
		} else {
			cfg.WorktreePath = "../fr8"
		}
	}
}

// ResolveWorktreePath resolves the worktree base directory relative to rootPath.
// The result includes the repo name as a subdirectory.
func ResolveWorktreePath(cfg *Config, rootPath string) string {
	base := cfg.WorktreePath
	if strings.HasPrefix(base, "~/") || base == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			base = filepath.Join(home, base[2:])
		}
	}
	if !filepath.IsAbs(base) {
		base = filepath.Join(rootPath, base)
	}
	repoName := filepath.Base(rootPath)
	return filepath.Clean(filepath.Join(base, repoName))
}
