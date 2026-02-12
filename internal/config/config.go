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
	PortRange    int     `json:"portRange"`
	BasePort     int     `json:"basePort"`
	WorktreePath string  `json:"worktreePath"`
}

// Scripts defines the lifecycle commands.
type Scripts struct {
	Setup   string `json:"setup"`
	Run     string `json:"run"`
	Archive string `json:"archive"`
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
		cfg.BasePort = 5000
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
