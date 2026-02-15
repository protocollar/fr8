package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/protocollar/fr8/internal/config"
	"github.com/protocollar/fr8/internal/env"
	"github.com/protocollar/fr8/internal/git"
	"github.com/protocollar/fr8/internal/jsonout"
	"github.com/protocollar/fr8/internal/registry"
	"github.com/protocollar/fr8/internal/tmux"
)

var runAll bool
var runIfNotRunning bool

func init() {
	runCmd.Flags().BoolVarP(&runAll, "all", "A", false, "Start all workspaces in the current repo")
	runCmd.Flags().BoolVar(&runIfNotRunning, "if-not-running", false, "succeed silently if already running")
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

	ws, rootPath, err := resolveWorkspace(name)
	if err != nil {
		return err
	}

	cfg, err := config.Load(rootPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.Scripts.Run == "" {
		return fmt.Errorf("no run script configured (add \"scripts.run\" to fr8.json)")
	}

	defaultBranch, _ := git.DefaultBranch(rootPath)
	envVars := env.BuildFr8Only(ws, rootPath, defaultBranch)

	sessionName := tmux.SessionName(tmux.RepoName(rootPath), ws.Name)

	if tmux.IsRunning(sessionName) {
		if runIfNotRunning {
			if jsonout.Enabled {
				return jsonout.Write(struct {
					Action    string `json:"action"`
					Workspace string `json:"workspace"`
					Session   string `json:"session"`
				}{Action: "already_running", Workspace: ws.Name, Session: sessionName})
			}
			fmt.Printf("Workspace %q is already running.\n", ws.Name)
			return nil
		}
		return fmt.Errorf("session %q is already running (use fr8 ws attach to connect)", sessionName)
	}

	if err := tmux.Start(sessionName, ws.Path, cfg.Scripts.Run, envVars); err != nil {
		return err
	}

	if jsonout.Enabled {
		return jsonout.Write(struct {
			Action    string `json:"action"`
			Workspace string `json:"workspace"`
			Session   string `json:"session"`
		}{Action: "started", Workspace: ws.Name, Session: sessionName})
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

	regPath, err := registry.DefaultPath()
	if err != nil {
		return fmt.Errorf("finding state path: %w", err)
	}
	reg, err := registry.Load(regPath)
	if err != nil {
		return fmt.Errorf("loading registry: %w", err)
	}

	// Find repo in registry by CWD
	rootPath, err := git.RootWorktreePath(cwd)
	if err != nil {
		return fmt.Errorf("finding root worktree: %w", err)
	}

	repo := reg.FindByPath(rootPath)
	if repo == nil {
		return fmt.Errorf("repo not found in registry (run fr8 repo add first)")
	}

	if len(repo.Workspaces) == 0 {
		if jsonout.Enabled {
			return jsonout.Write(struct {
				Started        []string `json:"started"`
				AlreadyRunning []string `json:"already_running"`
				Failed         []any    `json:"failed"`
			}{Started: []string{}, AlreadyRunning: []string{}, Failed: []any{}})
		}
		fmt.Println("No workspaces found.")
		return nil
	}

	cfg, err := config.Load(rootPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.Scripts.Run == "" {
		return fmt.Errorf("no run script configured (add \"scripts.run\" to fr8.json)")
	}

	defaultBranch, _ := git.DefaultBranch(rootPath)
	repoName := tmux.RepoName(rootPath)

	var started, skipped int
	var startedNames, alreadyRunning []string
	var failed []runFailedItem

	for i := range repo.Workspaces {
		ws := &repo.Workspaces[i]
		sessionName := tmux.SessionName(repoName, ws.Name)

		if tmux.IsRunning(sessionName) {
			skipped++
			alreadyRunning = append(alreadyRunning, ws.Name)
			continue
		}

		envVars := env.BuildFr8Only(ws, rootPath, defaultBranch)
		if err := tmux.Start(sessionName, ws.Path, cfg.Scripts.Run, envVars); err != nil {
			if !jsonout.Enabled {
				fmt.Fprintf(os.Stderr, "Warning: failed to start %q: %v\n", ws.Name, err)
			}
			failed = append(failed, runFailedItem{Workspace: ws.Name, Error: err.Error()})
			continue
		}

		if !jsonout.Enabled {
			fmt.Printf("Started %q\n", ws.Name)
		}
		startedNames = append(startedNames, ws.Name)
		started++
	}

	if jsonout.Enabled {
		if failed == nil {
			failed = []runFailedItem{}
		}
		return jsonout.Write(struct {
			Started        []string        `json:"started"`
			AlreadyRunning []string        `json:"already_running"`
			Failed         []runFailedItem `json:"failed"`
		}{
			Started:        orEmpty(startedNames),
			AlreadyRunning: orEmpty(alreadyRunning),
			Failed:         failed,
		})
	}

	fmt.Printf("Started %d workspace(s)", started)
	if skipped > 0 {
		fmt.Printf(", %d already running", skipped)
	}
	fmt.Println()
	return nil
}

type runFailedItem struct {
	Workspace string `json:"workspace"`
	Error     string `json:"error"`
}

func shellPath() (string, error) {
	p, err := exec.LookPath("sh")
	if err != nil {
		return "", fmt.Errorf("sh not found: %w", err)
	}
	return p, nil
}
