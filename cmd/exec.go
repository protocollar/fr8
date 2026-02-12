package cmd

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/env"
	"github.com/thomascarr/fr8/internal/git"
)

func init() {
	workspaceCmd.AddCommand(execCmd)
}

var execCmd = &cobra.Command{
	Use:   "exec [name] -- <command>",
	Short: "Run a command with workspace environment",
	Long: `Runs an arbitrary command with workspace environment variables set.
The workspace name is optional if you're inside a workspace directory.

Examples:
  fr8 exec myws -- bundle exec rails c
  fr8 exec -- npm test
  cd /path/to/workspace && fr8 exec -- make build`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	RunE:               runExec,
}

func runExec(cmd *cobra.Command, args []string) error {
	// Parse: either "exec -- <cmd>" or "exec <name> -- <cmd>"
	var wsName string
	var command []string

	dashIdx := -1
	for i, a := range args {
		if a == "--" {
			dashIdx = i
			break
		}
	}

	if dashIdx == -1 {
		return fmt.Errorf("usage: fr8 exec [name] -- <command>")
	}
	if dashIdx > 0 {
		wsName = args[0]
	}
	command = args[dashIdx+1:]
	if len(command) == 0 {
		return fmt.Errorf("no command specified after --")
	}

	ws, rootPath, _, err := resolveWorkspace(wsName)
	if err != nil {
		return err
	}

	defaultBranch, _ := git.DefaultBranch(rootPath)
	envVars := env.Build(ws, rootPath, defaultBranch)

	if err := os.Chdir(ws.Path); err != nil {
		return fmt.Errorf("changing to workspace directory: %w", err)
	}

	shell, err := shellPath()
	if err != nil {
		return err
	}
	return syscall.Exec(shell, []string{"sh", "-c", strings.Join(command, " ")}, envVars)
}
