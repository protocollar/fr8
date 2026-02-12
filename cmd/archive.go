package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/config"
	"github.com/thomascarr/fr8/internal/env"
	"github.com/thomascarr/fr8/internal/git"
	"github.com/thomascarr/fr8/internal/state"
)

var archiveForce bool

func init() {
	archiveCmd.Flags().BoolVarP(&archiveForce, "force", "f", false, "skip confirmation and uncommitted changes check")
	workspaceCmd.AddCommand(archiveCmd)
}

var archiveCmd = &cobra.Command{
	Use:               "archive [name]",
	Short:             "Tear down a workspace",
	Long:              "Runs the archive script, removes the git worktree, and frees the port allocation.",
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

	// Safety checks
	if !archiveForce {
		dirty, _ := git.HasUncommittedChanges(ws.Path)
		if dirty {
			return fmt.Errorf("workspace %q has uncommitted changes (use --force to override)", ws.Name)
		}

		fmt.Printf("Archive workspace %q? This will:\n", ws.Name)
		fmt.Printf("  - Run archive script\n")
		fmt.Printf("  - Remove worktree at %s\n", ws.Path)
		fmt.Printf("  - Free port %d\n", ws.Port)
		fmt.Printf("\nContinue? [y/N] ")

		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Run archive script
	defaultBranch, _ := git.DefaultBranch(rootPath)
	if cfg.Scripts.Archive != "" {
		fmt.Printf("Running archive script: %s\n", cfg.Scripts.Archive)
		envVars := env.Build(ws, rootPath, defaultBranch)
		if err := runScript(cfg.Scripts.Archive, ws.Path, envVars); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: archive script failed: %v\n", err)
		}
	}

	// Remove worktree
	fmt.Println("Removing worktree...")
	if err := git.WorktreeRemove(rootPath, ws.Path); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to remove worktree: %v\n", err)
		fmt.Fprintln(os.Stderr, "You may need to remove it manually: git worktree remove", ws.Path)
	}

	// Update state
	st.Remove(ws.Name)
	if err := st.Save(commonDir); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Printf("Workspace %q archived.\n", ws.Name)
	return nil
}
