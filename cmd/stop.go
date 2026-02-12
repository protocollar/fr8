package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/tmux"
)

func init() {
	workspaceCmd.AddCommand(stopCmd)
}

var stopCmd = &cobra.Command{
	Use:               "stop [name]",
	Short:             "Stop a workspace's background tmux session",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: workspaceNameCompletion,
	RunE:              runStop,
}

func runStop(cmd *cobra.Command, args []string) error {
	if err := tmux.Available(); err != nil {
		return err
	}

	var name string
	if len(args) > 0 {
		name = args[0]
	}

	ws, rootPath, _, err := resolveWorkspace(name)
	if err != nil {
		return err
	}

	sessionName := tmux.SessionName(tmux.RepoName(rootPath), ws.Name)
	if !tmux.IsRunning(sessionName) {
		fmt.Printf("Workspace %q is not running.\n", ws.Name)
		return nil
	}

	if err := tmux.Stop(sessionName); err != nil {
		return err
	}

	fmt.Printf("Stopped %q.\n", ws.Name)
	return nil
}
