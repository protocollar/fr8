package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/config"
	"github.com/thomascarr/fr8/internal/env"
	"github.com/thomascarr/fr8/internal/git"
	"github.com/thomascarr/fr8/internal/tmux"
)

func init() {
	workspaceCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:               "start [name]",
	Short:             "Start the dev server in a background tmux session",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: workspaceNameCompletion,
	RunE:              runStart,
}

func runStart(cmd *cobra.Command, args []string) error {
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

	cfg, err := config.Load(rootPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.Scripts.Run == "" {
		return fmt.Errorf("no run script configured in fr8.json")
	}

	defaultBranch, _ := git.DefaultBranch(rootPath)
	envVars := env.BuildFr8Only(ws, rootPath, defaultBranch)

	sessionName := tmux.SessionName(tmux.RepoName(rootPath), ws.Name)
	if err := tmux.Start(sessionName, ws.Path, cfg.Scripts.Run, envVars); err != nil {
		return err
	}

	fmt.Printf("Started %q in background.\n", ws.Name)
	fmt.Printf("  Attach with: fr8 ws attach %s\n", ws.Name)
	fmt.Printf("  Logs:        fr8 ws logs %s\n", ws.Name)
	fmt.Printf("  Stop:        fr8 ws stop %s\n", ws.Name)
	return nil
}
