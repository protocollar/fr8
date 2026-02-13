package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/env"
	"github.com/thomascarr/fr8/internal/git"
)

func init() {
	workspaceCmd.AddCommand(shellCmd)
}

var shellCmd = &cobra.Command{
	Use:   "shell [name]",
	Short: "Open a subshell with workspace environment",
	Example: `  fr8 ws shell
  fr8 ws shell my-feature`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: workspaceNameCompletion,
	RunE:              runShell,
}

func runShell(cmd *cobra.Command, args []string) error {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	ws, rootPath, _, err := resolveWorkspace(name)
	if err != nil {
		return err
	}

	defaultBranch, _ := git.DefaultBranch(rootPath)
	envVars := env.Build(ws, rootPath, defaultBranch)

	// Use the user's preferred shell
	userShell := os.Getenv("SHELL")
	if userShell == "" {
		userShell = "/bin/bash"
	}

	fmt.Printf("Entering workspace %q (%s)\n", ws.Name, ws.Path)
	fmt.Printf("Type 'exit' to leave the workspace shell.\n\n")

	c := exec.Command(userShell)
	c.Dir = ws.Path
	c.Env = envVars
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin

	if err := c.Run(); err != nil {
		// Non-zero exit from the shell is normal (user typed exit)
		if _, ok := err.(*exec.ExitError); ok {
			return nil
		}
		return err
	}

	fmt.Printf("\nLeft workspace %q.\n", ws.Name)
	return nil
}
