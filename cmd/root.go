package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/exitcode"
	"github.com/thomascarr/fr8/internal/jsonout"
)

var jsonOutput bool
var conciseOutput bool

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
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		jsonout.Enabled = jsonOutput
		jsonout.Concise = conciseOutput

		if conciseOutput && !jsonOutput {
			return fmt.Errorf("--concise requires --json")
		}

		if jsonOutput {
			jsonout.SetMsgOut(io.Discard)
		} else if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
			// Non-TTY: route human messages to stderr so piped stdout stays clean
			jsonout.SetMsgOut(os.Stderr)
		}

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output as JSON")
	rootCmd.PersistentFlags().BoolVar(&conciseOutput, "concise", false, "minimal JSON fields (use with --json)")
}

// RootCommand returns the top-level cobra.Command for testing.
func RootCommand() *cobra.Command { return rootCmd }

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		var exitErr *exitcode.ExitError
		if errors.As(err, &exitErr) {
			if jsonout.Enabled {
				jsonout.WriteError(exitErr.Code, exitErr.Error(), exitErr.ExitCode)
			} else {
				fmt.Fprintln(os.Stderr, err)
			}
			os.Exit(exitErr.ExitCode)
		}

		// Classify untyped errors
		code, exitCode := exitcode.ClassifyError(err)
		if jsonout.Enabled {
			jsonout.WriteError(code, err.Error(), exitCode)
		} else {
			// Suggest "fr8 workspace <cmd>" for old top-level workspace commands
			if args := os.Args[1:]; len(args) > 0 && workspaceCmds[args[0]] {
				fmt.Fprintf(os.Stderr, "Did you mean: fr8 workspace %s\n", args[0])
			} else {
				fmt.Fprintln(os.Stderr, err)
			}
		}
		os.Exit(exitCode)
	}
}

// isInteractive returns true if stdout is a TTY and --json is not active.
func isInteractive() bool {
	if jsonout.Enabled {
		return false
	}
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}
