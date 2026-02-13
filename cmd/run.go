package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/config"
	"github.com/thomascarr/fr8/internal/env"
	"github.com/thomascarr/fr8/internal/git"
	"github.com/thomascarr/fr8/internal/state"
	"github.com/thomascarr/fr8/internal/tmux"
)

var runAll bool

func init() {
	runCmd.Flags().BoolVarP(&runAll, "all", "A", false, "Start all workspaces in the current repo")
	workspaceCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run [name]",
	Short: "Run the dev server in a background tmux session",
	Example: `  fr8 ws run
  fr8 ws run my-feature
  fr8 ws run --all`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: workspaceNameCompletion,
	RunE:              runRun,
}

func runRun(cmd *cobra.Command, args []string) error {
	if runAll {
		if len(args) > 0 {
			return fmt.Errorf("cannot use --all with a workspace name")
		}
		return runRunAll()
	}

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

func runRunAll() error {
	if err := tmux.Available(); err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	commonDir, err := git.CommonDir(cwd)
	if err != nil {
		return fmt.Errorf("not inside a git repository")
	}

	st, err := state.Load(commonDir)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	if len(st.Workspaces) == 0 {
		fmt.Println("No workspaces found.")
		return nil
	}

	rootPath, err := git.RootWorktreePath(cwd)
	if err != nil {
		return fmt.Errorf("finding root worktree: %w", err)
	}

	cfg, err := config.Load(rootPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.Scripts.Run == "" {
		return fmt.Errorf("no run script configured in fr8.json")
	}

	defaultBranch, _ := git.DefaultBranch(rootPath)
	repoName := tmux.RepoName(rootPath)

	var started, skipped int
	for i := range st.Workspaces {
		ws := &st.Workspaces[i]
		sessionName := tmux.SessionName(repoName, ws.Name)

		if tmux.IsRunning(sessionName) {
			skipped++
			continue
		}

		envVars := env.BuildFr8Only(ws, rootPath, defaultBranch)
		if err := tmux.Start(sessionName, ws.Path, cfg.Scripts.Run, envVars); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to start %q: %v\n", ws.Name, err)
			continue
		}

		fmt.Printf("Started %q\n", ws.Name)
		started++
	}

	fmt.Printf("Started %d workspace(s)", started)
	if skipped > 0 {
		fmt.Printf(", %d already running", skipped)
	}
	fmt.Println()
	return nil
}

func shellPath() (string, error) {
	p, err := exec.LookPath("sh")
	if err != nil {
		return "", fmt.Errorf("sh not found: %w", err)
	}
	return p, nil
}
