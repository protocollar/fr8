package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/protocollar/fr8/internal/git"
	"github.com/protocollar/fr8/internal/jsonout"
	"github.com/protocollar/fr8/internal/registry"
	"github.com/protocollar/fr8/internal/tmux"
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

	ws, rootPath, err := resolveWorkspace(oldName)
	if err != nil {
		return err
	}

	// Move the worktree directory (e.g. ~/fr8/myapp/old-name → ~/fr8/myapp/new-name)
	oldPath := ws.Path
	newPath := filepath.Join(filepath.Dir(oldPath), newName)
	if err := git.WorktreeMove(rootPath, oldPath, newPath); err != nil {
		return fmt.Errorf("moving worktree: %w", err)
	}

	// Update state: name and path
	regPath, err := registry.DefaultPath()
	if err != nil {
		return err
	}
	reg, err := registry.Load(regPath)
	if err != nil {
		return fmt.Errorf("loading registry: %w", err)
	}
	repo := reg.FindByPath(rootPath)
	if repo == nil {
		return fmt.Errorf("repo not found in registry for path: %s", rootPath)
	}
	if err := repo.RenameWorkspace(oldName, newName); err != nil {
		return err
	}
	renamed := repo.FindWorkspace(newName)
	renamed.Path = newPath

	if err := reg.Save(regPath); err != nil {
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
