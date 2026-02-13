package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/git"
	"github.com/thomascarr/fr8/internal/jsonout"
	"github.com/thomascarr/fr8/internal/state"
	"github.com/thomascarr/fr8/internal/tmux"
)

func init() {
	workspaceCmd.AddCommand(renameCmd)
}

var renameCmd = &cobra.Command{
	Use:               "rename <old> <new>",
	Short:             "Rename a workspace",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: workspaceNameCompletion,
	RunE:              runRename,
}

func runRename(cmd *cobra.Command, args []string) error {
	oldName := args[0]
	newName := args[1]

	ws, rootPath, commonDir, err := resolveWorkspace(oldName)
	if err != nil {
		return err
	}

	st, err := state.Load(commonDir)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	// Move the worktree directory (e.g. ~/fr8/myapp/old-name → ~/fr8/myapp/new-name)
	oldPath := ws.Path
	newPath := filepath.Join(filepath.Dir(oldPath), newName)
	if err := git.WorktreeMove(rootPath, oldPath, newPath); err != nil {
		return fmt.Errorf("moving worktree: %w", err)
	}

	// Update state: name and path
	if err := st.Rename(oldName, newName); err != nil {
		return err
	}
	renamed := st.Find(newName)
	renamed.Path = newPath

	if err := st.Save(commonDir); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	// Rename tmux session if running
	if tmux.Available() == nil {
		repoName := tmux.RepoName(rootPath)
		oldSession := tmux.SessionName(repoName, oldName)
		if tmux.IsRunning(oldSession) {
			newSession := tmux.SessionName(repoName, newName)
			_ = tmux.RenameSession(oldSession, newSession)
		}
	}

	if jsonout.Enabled {
		return jsonout.Write(struct {
			Action  string `json:"action"`
			OldName string `json:"old_name"`
			NewName string `json:"new_name"`
			Path    string `json:"path"`
		}{Action: "renamed", OldName: oldName, NewName: newName, Path: newPath})
	}

	fmt.Printf("Renamed %q → %q\n", oldName, newName)
	fmt.Printf("  Path: %s\n", newPath)
	return nil
}
