package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/tmux"
)

var logsLines int
var logsFollow bool

func init() {
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 50, "number of lines to capture")
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "follow output (poll every 1s)")
	workspaceCmd.AddCommand(logsCmd)
}

var logsCmd = &cobra.Command{
	Use:   "logs [name]",
	Short: "Show recent output from a workspace's background session",
	Example: `  fr8 ws logs
  fr8 ws logs my-feature
  fr8 ws logs -n 100
  fr8 ws logs -f`,
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

	if !logsFollow {
		output, err := tmux.CapturePanes(sessionName, logsLines)
		if err != nil {
			return err
		}
		fmt.Print(output)
		return nil
	}

	// Follow mode: poll and redraw
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Initial capture
	output, err := tmux.CapturePanes(sessionName, logsLines)
	if err != nil {
		return err
	}
	fmt.Print("\033[2J\033[H") // clear screen
	fmt.Print(output)

	for {
		select {
		case <-sig:
			return nil
		case <-ticker.C:
			if !tmux.IsRunning(sessionName) {
				fmt.Fprintf(os.Stderr, "\nSession ended.\n")
				return nil
			}
			output, err := tmux.CapturePanes(sessionName, logsLines)
			if err != nil {
				return err
			}
			fmt.Print("\033[2J\033[H") // clear screen
			fmt.Print(output)
		}
	}
}
