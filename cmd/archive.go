package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/config"
	"github.com/thomascarr/fr8/internal/env"
	"github.com/thomascarr/fr8/internal/exitcode"
	"github.com/thomascarr/fr8/internal/git"
	"github.com/thomascarr/fr8/internal/jsonout"
	"github.com/thomascarr/fr8/internal/state"
	"github.com/thomascarr/fr8/internal/tmux"
)

var archiveForce bool
var archiveIfExists bool
var archiveDryRun bool

func init() {
	archiveCmd.Flags().BoolVarP(&archiveForce, "force", "f", false, "skip confirmation and uncommitted changes check")
	archiveCmd.Flags().BoolVar(&archiveIfExists, "if-exists", false, "succeed silently if workspace not found")
	archiveCmd.Flags().BoolVar(&archiveDryRun, "dry-run", false, "show what would be done without doing it")
	workspaceCmd.AddCommand(archiveCmd)
}

var archiveCmd = &cobra.Command{
	Use:   "archive [name]",
	Short: "Tear down a workspace",
	Long:  "Runs the archive script, removes the git worktree, and frees the port allocation.",
	Example: `  fr8 ws archive
  fr8 ws archive my-feature
  fr8 ws archive --force`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: workspaceNameCompletion,
	RunE:              runArchive,
}

func runArchive(cmd *cobra.Command, args []string) error {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	ws, rootPath, commonDir, err := resolveWorkspace(name)
	if err != nil {
		if archiveIfExists {
			if jsonout.Enabled {
				return jsonout.Write(struct {
					Action string `json:"action"`
				}{Action: "not_found"})
			}
			return nil
		}
		return err
	}

	cfg, err := config.Load(rootPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	st, err := state.Load(commonDir)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	// Dry run: just report what would happen
	if archiveDryRun {
		dirty, _ := git.HasUncommittedChanges(ws.Path)
		result := struct {
			Action    string `json:"action"`
			Workspace struct {
				Name   string `json:"name"`
				Branch string `json:"branch"`
				Port   int    `json:"port"`
				Path   string `json:"path"`
			} `json:"workspace"`
			Dirty      bool   `json:"dirty"`
			HasScript  bool   `json:"has_archive_script"`
			ScriptName string `json:"archive_script,omitempty"`
		}{Action: "dry_run"}
		result.Workspace.Name = ws.Name
		result.Workspace.Branch = ws.Branch
		result.Workspace.Port = ws.Port
		result.Workspace.Path = ws.Path
		result.Dirty = dirty
		result.HasScript = cfg.Scripts.Archive != ""
		if cfg.Scripts.Archive != "" {
			result.ScriptName = cfg.Scripts.Archive
		}
		if jsonout.Enabled {
			return jsonout.Write(result)
		}
		fmt.Printf("Dry run — would archive workspace %q:\n", ws.Name)
		fmt.Printf("  Path:     %s\n", ws.Path)
		fmt.Printf("  Branch:   %s\n", ws.Branch)
		fmt.Printf("  Port:     %d\n", ws.Port)
		if dirty {
			fmt.Printf("  Status:   dirty (uncommitted changes)\n")
		}
		if cfg.Scripts.Archive != "" {
			fmt.Printf("  Script:   %s\n", cfg.Scripts.Archive)
		}
		return nil
	}

	// Safety checks
	if !archiveForce {
		dirty, _ := git.HasUncommittedChanges(ws.Path)
		if dirty {
			if jsonout.Enabled {
				return &exitcode.ExitError{
					Err:      fmt.Errorf("workspace %q has uncommitted changes (use --force to override)", ws.Name),
					ExitCode: exitcode.DirtyWorkspace,
					Code:     "dirty_workspace",
				}
			}
			return fmt.Errorf("workspace %q has uncommitted changes (use --force to override)", ws.Name)
		}

		// Skip interactive confirmation when --json or non-TTY
		if isInteractive() {
			fmt.Printf("Archive workspace %q? This will:\n", ws.Name)
			fmt.Printf("  - Run archive script\n")
			fmt.Printf("  - Remove worktree at %s\n", ws.Path)
			fmt.Printf("  - Free port %d\n", ws.Port)
			fmt.Printf("\nContinue? [y/N] ")

			var response string
			_, _ = fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}
	}

	// Auto-stop tmux session if running
	if tmux.Available() == nil {
		sessionName := tmux.SessionName(tmux.RepoName(rootPath), ws.Name)
		if tmux.IsRunning(sessionName) {
			_, _ = fmt.Fprintf(jsonout.MsgOut(), "Stopping background session...\n")
			if err := tmux.Stop(sessionName); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to stop tmux session: %v\n", err)
			}
		}
	}

	// Run archive script
	defaultBranch, _ := git.DefaultBranch(rootPath)
	if cfg.Scripts.Archive != "" {
		_, _ = fmt.Fprintf(jsonout.MsgOut(), "Running archive script: %s\n", cfg.Scripts.Archive)
		envVars := env.Build(ws, rootPath, defaultBranch)
		if err := runScript(cfg.Scripts.Archive, ws.Path, envVars); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: archive script failed: %v\n", err)
		}
	}

	// Remove worktree
	_, _ = fmt.Fprintf(jsonout.MsgOut(), "Removing worktree...\n")
	if err := git.WorktreeRemove(rootPath, ws.Path); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to remove worktree: %v\n", err)
		fmt.Fprintln(os.Stderr, "You may need to remove it manually: git worktree remove", ws.Path)
	}

	// Update state
	_ = st.Remove(ws.Name)
	if err := st.Save(commonDir); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	if jsonout.Enabled {
		return jsonout.Write(struct {
			Action    string `json:"action"`
			Workspace struct {
				Name   string `json:"name"`
				Branch string `json:"branch"`
				Port   int    `json:"port"`
				Path   string `json:"path"`
			} `json:"workspace"`
		}{
			Action: "archived",
			Workspace: struct {
				Name   string `json:"name"`
				Branch string `json:"branch"`
				Port   int    `json:"port"`
				Path   string `json:"path"`
			}{Name: ws.Name, Branch: ws.Branch, Port: ws.Port, Path: ws.Path},
		})
	}

	fmt.Printf("Workspace %q archived.\n", ws.Name)
	return nil
}
