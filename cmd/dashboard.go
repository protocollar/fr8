package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/env"
	"github.com/thomascarr/fr8/internal/git"
	"github.com/thomascarr/fr8/internal/tmux"
	"github.com/thomascarr/fr8/internal/tui"
)

func init() {
	rootCmd.AddCommand(dashboardCmd)
}

var dashboardCmd = &cobra.Command{
	Use:     "dashboard",
	Aliases: []string{"dash"},
	Short:   "Interactive TUI for browsing repos and workspaces",
	Args:    cobra.NoArgs,
	RunE:    runDashboard,
}

func runDashboard(cmd *cobra.Command, args []string) error {
	for {
		result, err := tui.RunDashboard()
		if err != nil {
			return err
		}

		if result.ShellWorkspace != nil {
			ws := result.ShellWorkspace
			rootPath := result.RootPath
			defaultBranch, _ := git.DefaultBranch(rootPath)
			envVars := env.Build(ws, rootPath, defaultBranch)

			userShell := os.Getenv("SHELL")
			if userShell == "" {
				userShell = "/bin/bash"
			}

			fmt.Printf("Entering workspace %q (%s)\n", ws.Name, ws.Path)
			fmt.Printf("Type 'exit' to return to the dashboard.\n\n")

			c := exec.Command(userShell)
			c.Dir = ws.Path
			c.Env = envVars
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Stdin = os.Stdin

			if err := c.Run(); err != nil {
				if _, ok := err.(*exec.ExitError); !ok {
					return err
				}
			}

			fmt.Printf("\nLeft workspace %q.\n", ws.Name)
			continue
		}

		if result.AttachWorkspace != nil {
			ws := result.AttachWorkspace
			rootPath := result.RootPath
			sessionName := tmux.SessionName(tmux.RepoName(rootPath), ws.Name)
			if err := tmux.AttachRun(sessionName); err != nil {
				return err
			}
			continue
		}

		// No action requested (user quit) — exit the loop
		return nil
	}
}
