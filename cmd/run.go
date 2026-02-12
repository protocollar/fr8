package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/config"
	"github.com/thomascarr/fr8/internal/env"
	"github.com/thomascarr/fr8/internal/git"
)

func init() {
	workspaceCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:               "run [name]",
	Short:             "Start the dev server in a workspace",
	Long:              "Execs into the run script, replacing the fr8 process for clean signal handling.",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: workspaceNameCompletion,
	RunE:              runRun,
}

func runRun(cmd *cobra.Command, args []string) error {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	ws, rootPath, _, err := resolveWorkspace(name)
	if err != nil {
		return err
	}

	cfg, err := config.Load(rootPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.Scripts.Run == "" {
		return fmt.Errorf("no run script configured in fr8.json")
	}

	defaultBranch, _ := git.DefaultBranch(rootPath)
	envVars := env.Build(ws, rootPath, defaultBranch)

	// Change to workspace directory before exec
	if err := os.Chdir(ws.Path); err != nil {
		return fmt.Errorf("changing to workspace directory: %w", err)
	}

	// Exec replaces the current process - signals go directly to the script
	shell, err := shellPath()
	if err != nil {
		return err
	}
	return syscall.Exec(shell, []string{"sh", "-c", cfg.Scripts.Run}, envVars)
}

func shellPath() (string, error) {
	p, err := exec.LookPath("sh")
	if err != nil {
		return "", fmt.Errorf("sh not found: %w", err)
	}
	return p, nil
}
