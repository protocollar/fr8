package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/protocollar/fr8/internal/config"
	"github.com/protocollar/fr8/internal/exitcode"
	"github.com/protocollar/fr8/internal/git"
	"github.com/protocollar/fr8/internal/jsonout"
)

var doctorFix bool

func init() {
	configDoctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "auto-fix correctable issues")
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configDoctorCmd)
	configCmd.AddCommand(configValidateCmd) // alias
	configCmd.AddCommand(configOpenCmd)
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and manage configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show resolved configuration",
	Args:  cobra.NoArgs,
	RunE:  runConfigShow,
}

var configDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check configuration health and optionally fix issues",
	Args:  cobra.NoArgs,
	RunE:  runConfigDoctor,
}

var configValidateCmd = &cobra.Command{
	Use:    "validate",
	Short:  "Check configuration health and optionally fix issues",
	Args:   cobra.NoArgs,
	RunE:   runConfigDoctor,
	Hidden: true, // deprecated alias for doctor
}

var configOpenCmd = &cobra.Command{
	Use:   "open",
	Short: "Open the fr8 config directory in the file manager",
	Args:  cobra.NoArgs,
	RunE:  runConfigOpen,
}

// configDir returns the fr8 global config directory (~/.config/fr8).
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".config", "fr8"), nil
}

func runConfigOpen(cmd *cobra.Command, args []string) error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	if jsonout.Enabled {
		return jsonout.Write(struct {
			Path string `json:"path"`
		}{Path: dir})
	}

	return openBrowser(dir)
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
		"port_range":             cfg.PortRange,
		"base_port":              cfg.BasePort,
		"worktree_path":          cfg.WorktreePath,
		"resolved_worktree_path": config.ResolveWorktreePath(cfg, rootPath),
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

func runConfigDoctor(cmd *cobra.Command, args []string) error {
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
	var fixableFiles []string // config files with legacy keys

	// Check for deprecated camelCase keys
	for _, name := range []string{"fr8.json", "conductor.json"} {
		p := filepath.Join(rootPath, name)
		if legacy := config.HasLegacyKeys(p); len(legacy) > 0 {
			fixableFiles = append(fixableFiles, p)
			for _, key := range legacy {
				warnings = append(warnings, fmt.Sprintf("%s: deprecated key %q — rename to %q (fixable)", name, key, config.LegacyKeyReplacement(key)))
			}
		}
	}

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
			configErrors = append(configErrors, fmt.Sprintf("worktree_path: %q exists but is not a directory", wtPath))
		}
	}
	// Parent must exist or be creatable — not an error if it doesn't exist yet

	// Check port ranges
	if cfg.BasePort < 1024 {
		warnings = append(warnings, fmt.Sprintf("base_port: %d is a privileged port (< 1024)", cfg.BasePort))
	}
	if cfg.BasePort > 65535 {
		configErrors = append(configErrors, fmt.Sprintf("base_port: %d is out of range (> 65535)", cfg.BasePort))
	}
	if cfg.PortRange < 1 {
		configErrors = append(configErrors, fmt.Sprintf("port_range: %d must be at least 1", cfg.PortRange))
	}
	if cfg.BasePort+cfg.PortRange*100 > 65535 {
		warnings = append(warnings, fmt.Sprintf("base_port %d + port_range %d may exhaust available ports with many workspaces", cfg.BasePort, cfg.PortRange))
	}

	// Handle --fix
	var fixed []string
	if doctorFix && len(fixableFiles) > 0 {
		if isInteractive() {
			fmt.Println("The following config files have deprecated camelCase keys:")
			for _, f := range fixableFiles {
				fmt.Printf("  %s\n", filepath.Base(f))
			}
			fmt.Printf("\nMigrate to snake_case? [y/N] ")

			var response string
			_, _ = fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Skipped.")
				fixableFiles = nil
			}
		} else if jsonout.Enabled {
			// In JSON mode --fix applies without prompting
		} else {
			return exitcode.New("interactive_only", exitcode.InteractiveOnly,
				"--fix requires an interactive terminal or --json mode")
		}

		for _, f := range fixableFiles {
			migrated, err := config.MigrateKeys(f)
			if err != nil {
				return fmt.Errorf("migrating %s: %w", filepath.Base(f), err)
			}
			for _, key := range migrated {
				fixed = append(fixed, fmt.Sprintf("%s: %s -> %s", filepath.Base(f), key, config.LegacyKeyReplacement(key)))
			}
		}
	}

	if jsonout.Enabled {
		return jsonout.Write(struct {
			Valid    bool     `json:"valid"`
			Errors   []string `json:"errors"`
			Warnings []string `json:"warnings"`
			Fixed    []string `json:"fixed"`
		}{
			Valid:    len(configErrors) == 0,
			Errors:   orEmpty(configErrors),
			Warnings: orEmpty(warnings),
			Fixed:    orEmpty(fixed),
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
	if len(fixed) > 0 {
		fmt.Println("\nFixed:")
		for _, f := range fixed {
			fmt.Printf("  ✓ %s\n", f)
		}
	}

	switch {
	case len(fixed) > 0 && len(configErrors) == 0:
		fmt.Println("\nFixed issues. Configuration is valid.")
	case len(configErrors) == 0 && len(warnings) == 0:
		fmt.Println("Configuration is valid.")
	case len(configErrors) == 0:
		if len(fixableFiles) > 0 && !doctorFix {
			fmt.Println("\nRun with --fix to auto-correct fixable issues.")
		}
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
