package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var workspaceCmds = map[string]bool{
	"new": true, "list": true, "status": true, "run": true,
	"exec": true, "shell": true, "cd": true, "archive": true,
}

var rootCmd = &cobra.Command{
	Use:   "fr8",
	Short: "Manage git worktrees as isolated dev workspaces",
	Long: `fr8 creates and manages git worktrees as isolated development workspaces,
each with its own port range, database prefix, and environment variables.

Configure via fr8.json (or conductor.json) in your repo root.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Suggest "fr8 workspace <cmd>" for old top-level workspace commands
		if args := os.Args[1:]; len(args) > 0 && workspaceCmds[args[0]] {
			fmt.Fprintf(os.Stderr, "Did you mean: fr8 workspace %s\n", args[0])
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
