package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/protocollar/fr8/internal/config"
	"github.com/protocollar/fr8/internal/git"
	"github.com/protocollar/fr8/internal/jsonout"
)

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configValidateCmd)
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and validate configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show resolved configuration",
	Args:  cobra.NoArgs,
	RunE:  runConfigShow,
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",
	Args:  cobra.NoArgs,
	RunE:  runConfigValidate,
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	rootPath, err := git.RootWorktreePath(cwd)
	if err != nil {
		return fmt.Errorf("not inside a git repository (run from a repo or use --repo <name>)")
	}

	cfg, err := config.Load(rootPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	resolved := map[string]interface{}{
		"scripts": map[string]string{
			"setup":   cfg.Scripts.Setup,
			"run":     cfg.Scripts.Run,
			"archive": cfg.Scripts.Archive,
		},
		"portRange":            cfg.PortRange,
		"basePort":             cfg.BasePort,
		"worktreePath":         cfg.WorktreePath,
		"resolvedWorktreePath": config.ResolveWorktreePath(cfg, rootPath),
	}

	if jsonout.Enabled {
		return jsonout.Write(resolved)
	}

	data, err := json.MarshalIndent(resolved, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	rootPath, err := git.RootWorktreePath(cwd)
	if err != nil {
		return fmt.Errorf("not inside a git repository (run from a repo or use --repo <name>)")
	}

	cfg, err := config.Load(rootPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	var warnings []string
	var configErrors []string

	// Check script paths
	for name, script := range map[string]string{
		"setup":   cfg.Scripts.Setup,
		"run":     cfg.Scripts.Run,
		"archive": cfg.Scripts.Archive,
	} {
		if script == "" {
			continue
		}
		parts := strings.Fields(script)
		if _, err := exec.LookPath(parts[0]); err != nil {
			// Check relative to rootPath
			if _, err := os.Stat(fmt.Sprintf("%s/%s", rootPath, parts[0])); err != nil {
				warnings = append(warnings, fmt.Sprintf("scripts.%s: %q not found in $PATH or repo", name, parts[0]))
			}
		}
	}

	// Check worktree path writable
	wtPath := config.ResolveWorktreePath(cfg, rootPath)
	if info, err := os.Stat(wtPath); err == nil {
		if !info.IsDir() {
			configErrors = append(configErrors, fmt.Sprintf("worktreePath: %q exists but is not a directory", wtPath))
		}
	}
	// Parent must exist or be creatable — not an error if it doesn't exist yet

	// Check port ranges
	if cfg.BasePort < 1024 {
		warnings = append(warnings, fmt.Sprintf("basePort: %d is a privileged port (< 1024)", cfg.BasePort))
	}
	if cfg.BasePort > 65535 {
		configErrors = append(configErrors, fmt.Sprintf("basePort: %d is out of range (> 65535)", cfg.BasePort))
	}
	if cfg.PortRange < 1 {
		configErrors = append(configErrors, fmt.Sprintf("portRange: %d must be at least 1", cfg.PortRange))
	}
	if cfg.BasePort+cfg.PortRange*100 > 65535 {
		warnings = append(warnings, fmt.Sprintf("basePort %d + portRange %d may exhaust available ports with many workspaces", cfg.BasePort, cfg.PortRange))
	}

	if jsonout.Enabled {
		return jsonout.Write(struct {
			Valid    bool     `json:"valid"`
			Errors   []string `json:"errors"`
			Warnings []string `json:"warnings"`
		}{
			Valid:    len(configErrors) == 0,
			Errors:   orEmpty(configErrors),
			Warnings: orEmpty(warnings),
		})
	}

	if len(configErrors) > 0 {
		fmt.Println("Errors:")
		for _, e := range configErrors {
			fmt.Printf("  ✗ %s\n", e)
		}
	}
	if len(warnings) > 0 {
		fmt.Println("Warnings:")
		for _, w := range warnings {
			fmt.Printf("  ⚠ %s\n", w)
		}
	}

	switch {
	case len(configErrors) == 0 && len(warnings) == 0:
		fmt.Println("Configuration is valid.")
	case len(configErrors) == 0:
		fmt.Println("\nConfiguration is valid with warnings.")
	default:
		return fmt.Errorf("configuration has %d error(s); fix the issues above in fr8.json", len(configErrors))
	}

	return nil
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
