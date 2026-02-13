package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/exitcode"
	"github.com/thomascarr/fr8/internal/jsonout"
	"github.com/thomascarr/fr8/internal/tmux"
)

func init() {
	workspaceCmd.AddCommand(attachCmd)
}

var attachCmd = &cobra.Command{
	Use:               "attach [name]",
	Short:             "Attach to a workspace's background tmux session",
	Long:              "Replaces the fr8 process with tmux attach. Detach with Ctrl-B d.",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: workspaceNameCompletion,
	RunE:              runAttach,
}

func runAttach(cmd *cobra.Command, args []string) error {
	if jsonout.Enabled {
		return &exitcode.ExitError{
			Err:      fmt.Errorf("attach requires an interactive terminal and cannot be used with --json"),
			ExitCode: exitcode.InteractiveOnly,
			Code:     "interactive_only",
		}
	}

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
	return tmux.Attach(sessionName)
}
