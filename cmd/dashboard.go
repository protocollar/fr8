package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/env"
	"github.com/thomascarr/fr8/internal/exitcode"
	"github.com/thomascarr/fr8/internal/git"
	"github.com/thomascarr/fr8/internal/jsonout"
	"github.com/thomascarr/fr8/internal/opener"
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
	if jsonout.Enabled {
		return &exitcode.ExitError{
			Err:      fmt.Errorf("dashboard requires an interactive terminal and cannot be used with --json"),
			ExitCode: exitcode.InteractiveOnly,
			Code:     "interactive_only",
		}
	}

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
				var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
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

		if result.OpenWorkspace != nil {
			ws := result.OpenWorkspace
			openerPath, err := opener.DefaultPath()
			if err != nil {
				return err
			}
			openers, err := opener.Load(openerPath)
			if err != nil {
				return fmt.Errorf("loading openers: %w", err)
			}
			o := opener.Find(openers, result.OpenerName)
			if o == nil {
				return fmt.Errorf("opener %q not found", result.OpenerName)
			}
			if err := opener.Run(*o, ws.Path); err != nil {
				fmt.Fprintf(os.Stderr, "Error running opener %q: %v\n", o.Name, err)
			} else {
				fmt.Printf("Opened %q with %s\n", ws.Name, o.Name)
			}
			continue
		}

		if result.CreateRequested {
			ws, err := createWorkspace(result.RootPath, result.CommonDir, result.CreateName, "", false, true, false)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating workspace: %v\n", err)
			} else {
				fmt.Printf("Created workspace %q\n", ws.Name)
			}
			continue
		}

		// No action requested (user quit) — exit the loop
		return nil
	}
}
