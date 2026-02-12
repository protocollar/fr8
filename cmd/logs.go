package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/tmux"
)

var logsLines int

func init() {
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 50, "number of lines to capture")
	workspaceCmd.AddCommand(logsCmd)
}

var logsCmd = &cobra.Command{
	Use:               "logs [name]",
	Short:             "Show recent output from a workspace's background session",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: workspaceNameCompletion,
	RunE:              runLogs,
}

func runLogs(cmd *cobra.Command, args []string) error {
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
	output, err := tmux.CapturePanes(sessionName, logsLines)
	if err != nil {
		return err
	}

	fmt.Print(output)
	return nil
}
