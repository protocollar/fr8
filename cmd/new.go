package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/config"
	"github.com/thomascarr/fr8/internal/env"
	"github.com/thomascarr/fr8/internal/filesync"
	"github.com/thomascarr/fr8/internal/git"
	"github.com/thomascarr/fr8/internal/names"
	"github.com/thomascarr/fr8/internal/port"
	"github.com/thomascarr/fr8/internal/registry"
	"github.com/thomascarr/fr8/internal/state"
)

var newBranch string
var newRemote string
var newPR string
var noSetup bool
var noShell bool

func init() {
	newCmd.Flags().StringVarP(&newBranch, "branch", "b", "", "branch name (creates new branch if it doesn't exist)")
	newCmd.Flags().StringVarP(&newRemote, "remote", "r", "", "track an existing remote branch (fetches and creates local tracking branch)")
	newCmd.Flags().StringVarP(&newPR, "pull-request", "p", "", "create workspace from a GitHub pull request number (requires gh CLI)")
	newCmd.Flags().BoolVar(&noSetup, "no-setup", false, "skip running the setup script")
	newCmd.Flags().BoolVar(&noShell, "no-shell", false, "skip dropping into a workspace shell after creation")
	newCmd.MarkFlagsMutuallyExclusive("branch", "remote", "pull-request")
	workspaceCmd.AddCommand(newCmd)
}

var newCmd = &cobra.Command{
	Use:   "new [name]",
	Short: "Create a new workspace",
	Long:  "Creates a git worktree, allocates a port range, syncs files, and runs the setup script.",
	Example: `  fr8 ws new my-feature
  fr8 ws new my-feature -b feature/auth
  fr8 ws new -r feature/existing-branch
  fr8 ws new -p 42
  fr8 ws new --no-shell
  fr8 ws new --repo myapp my-feature`,
	Args: cobra.MaximumNArgs(1),
	RunE: runNew,
}

func runNew(cmd *cobra.Command, args []string) error {
	var rootPath, commonDir string

	if resolveRepo != "" {
		// Resolve from registry
		regPath, err := registry.DefaultPath()
		if err != nil {
			return err
		}
		reg, err := registry.Load(regPath)
		if err != nil {
			return fmt.Errorf("loading registry: %w", err)
		}
		repo := reg.Find(resolveRepo)
		if repo == nil {
			return fmt.Errorf("repo %q not found in registry (see: fr8 repo list)", resolveRepo)
		}
		rootPath = repo.Path
		commonDir, err = git.CommonDir(rootPath)
		if err != nil {
			return fmt.Errorf("finding git common dir: %w", err)
		}
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		if !git.IsInsideWorkTree(cwd) {
			return fmt.Errorf("not inside a git repository")
		}

		rootPath, err = git.RootWorktreePath(cwd)
		if err != nil {
			return fmt.Errorf("finding root worktree: %w", err)
		}

		commonDir, err = git.CommonDir(cwd)
		if err != nil {
			return fmt.Errorf("finding git common dir: %w", err)
		}
	}

	// Determine branch and whether to track remote
	branch := newBranch
	trackRemote := false

	if newPR != "" {
		resolved, err := resolvePRBranch(rootPath, newPR)
		if err != nil {
			return err
		}
		fmt.Printf("PR #%s → branch %s\n", newPR, resolved)
		branch = resolved
		trackRemote = true
	} else if newRemote != "" {
		branch = newRemote
		trackRemote = true
	}

	ws, err := createWorkspace(rootPath, commonDir, nameFromArgs(args), branch, trackRemote, !noSetup, !noShell)
	if err != nil {
		return err
	}
	_ = ws
	return nil
}

func nameFromArgs(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
}

// resolvePRBranch uses the gh CLI to resolve a PR number to its head branch name.
func resolvePRBranch(dir, prNumber string) (string, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", fmt.Errorf("gh CLI is required for --pr (install from https://cli.github.com)")
	}

	c := exec.Command("gh", "pr", "view", prNumber, "--json", "headRefName", "-q", ".headRefName")
	c.Dir = dir
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("resolving PR #%s: %s", prNumber, strings.TrimSpace(string(out)))
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" {
		return "", fmt.Errorf("PR #%s: could not resolve branch name", prNumber)
	}
	return branch, nil
}

// createWorkspace is the shared workspace creation logic used by both the CLI
// (runNew) and the TUI dashboard loop. When trackRemote is true, the branch is
// expected to exist on origin and a local tracking branch will be created.
func createWorkspace(rootPath, commonDir, wsName, branch string, trackRemote, runSetup, enterShell bool) (*state.Workspace, error) {
	cfg, err := config.Load(rootPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	st, err := state.Load(commonDir)
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}

	// Workspace name
	if wsName != "" {
		if st.Find(wsName) != nil {
			return nil, fmt.Errorf("workspace %q already exists", wsName)
		}
	} else {
		wsName = names.Generate(st.Names())
	}

	// Determine default branch and fetch latest from origin
	defaultBranch, _ := git.DefaultBranch(rootPath)
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	startPoint := ""
	remoteRef := "origin/" + defaultBranch
	fmt.Printf("Fetching latest from origin...\n")
	if err := git.Fetch(rootPath, "origin"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: git fetch failed: %v\n", err)
	}
	if git.RemoteRefExists(rootPath, remoteRef) {
		startPoint = remoteRef
	}

	// Branch resolution
	createBranch := false
	if branch == "" {
		branch = wsName
		createBranch = true
	} else if trackRemote {
		// --remote or --pr: create local tracking branch from origin/<branch>
		remoteBranch := "origin/" + branch
		if !git.RemoteRefExists(rootPath, remoteBranch) {
			return nil, fmt.Errorf("remote branch %s not found (did you forget to push?)", remoteBranch)
		}
		if !git.BranchExists(rootPath, branch) {
			fmt.Printf("Creating local branch %s tracking %s\n", branch, remoteBranch)
			if err := git.CreateTrackingBranch(rootPath, branch, remoteBranch); err != nil {
				return nil, fmt.Errorf("creating tracking branch: %w", err)
			}
		}
		startPoint = ""
		createBranch = false
	} else {
		// -b: create new local branch if it doesn't exist
		if !git.BranchExists(rootPath, branch) {
			createBranch = true
		}
	}

	// Port — collect ports from all registered repos to avoid cross-repo conflicts
	globalPorts := allAllocatedPorts()
	localPorts := st.AllocatedPorts()
	allocatedPort, err := port.Allocate(mergePorts(globalPorts, localPorts), cfg.BasePort, cfg.PortRange)
	if err != nil {
		return nil, fmt.Errorf("allocating port: %w", err)
	}

	// Worktree path
	wtBase := config.ResolveWorktreePath(cfg, rootPath)
	wsPath := filepath.Join(wtBase, wsName)

	// Create worktree
	fmt.Printf("Creating workspace %q...\n", wsName)
	if err := os.MkdirAll(wtBase, 0755); err != nil {
		return nil, fmt.Errorf("creating worktree directory: %w", err)
	}

	if err := git.WorktreeAdd(rootPath, wsPath, branch, createBranch, startPoint); err != nil {
		return nil, fmt.Errorf("creating worktree: %w", err)
	}

	ws := state.Workspace{
		Name:      wsName,
		Path:      wsPath,
		Branch:    branch,
		Port:      allocatedPort,
		CreatedAt: time.Now().UTC(),
	}

	if err := st.Add(ws); err != nil {
		// Clean up worktree on state failure
		git.WorktreeRemove(rootPath, wsPath)
		return nil, fmt.Errorf("saving workspace: %w", err)
	}
	if err := st.Save(commonDir); err != nil {
		git.WorktreeRemove(rootPath, wsPath)
		return nil, fmt.Errorf("saving state: %w", err)
	}

	// Auto-register repo in global registry
	autoRegisterRepo(rootPath)

	// Sync files
	fmt.Println("Syncing files...")
	if err := filesync.Sync(rootPath, wsPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: file sync failed: %v\n", err)
	}

	// Run setup script
	if runSetup && cfg.Scripts.Setup != "" {
		fmt.Printf("Running setup script: %s\n", cfg.Scripts.Setup)
		envVars := env.Build(&ws, rootPath, defaultBranch)
		if err := runScript(cfg.Scripts.Setup, wsPath, envVars); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: setup script failed: %v\n", err)
			fmt.Fprintln(os.Stderr, "The workspace was created but setup did not complete.")
			fmt.Fprintf(os.Stderr, "You can re-run setup with: cd %s && %s\n", wsPath, cfg.Scripts.Setup)
		}
	}

	// Print summary
	fmt.Println()
	fmt.Printf("Workspace created:\n")
	fmt.Printf("  Name:   %s\n", ws.Name)
	fmt.Printf("  Branch: %s\n", ws.Branch)
	fmt.Printf("  Ports:  %d-%d (%d ports)\n", ws.Port, ws.Port+cfg.PortRange-1, cfg.PortRange)
	fmt.Printf("  Path:   %s\n", shortenHomePath(ws.Path))

	// Drop into a subshell in the new workspace
	if enterShell {
		fmt.Println()
		fmt.Printf("Entering workspace %q...\n", ws.Name)
		fmt.Println("Type 'exit' to leave the workspace shell.")
		fmt.Println()

		envVars := env.Build(&ws, rootPath, defaultBranch)

		userShell := os.Getenv("SHELL")
		if userShell == "" {
			userShell = "/bin/bash"
		}

		c := exec.Command(userShell)
		c.Dir = ws.Path
		c.Env = envVars
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Stdin = os.Stdin

		if err := c.Run(); err != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				return &ws, err
			}
		}

		fmt.Printf("\nLeft workspace %q.\n", ws.Name)
	}

	return &ws, nil
}

// allAllocatedPorts collects every allocated port across all repos in the
// global registry. Failures are silently skipped so this never blocks
// workspace creation.
func allAllocatedPorts() []int {
	regPath, err := registry.DefaultPath()
	if err != nil {
		return nil
	}
	reg, err := registry.Load(regPath)
	if err != nil {
		return nil
	}
	var ports []int
	for _, repo := range reg.Repos {
		commonDir, err := git.CommonDir(repo.Path)
		if err != nil {
			continue
		}
		st, err := state.Load(commonDir)
		if err != nil {
			continue
		}
		ports = append(ports, st.AllocatedPorts()...)
	}
	return ports
}

// mergePorts returns the union of two port slices, deduplicating entries from b
// that already appear in a.
func mergePorts(a, b []int) []int {
	seen := make(map[int]bool, len(a))
	for _, p := range a {
		seen[p] = true
	}
	merged := append([]int{}, a...)
	for _, p := range b {
		if !seen[p] {
			merged = append(merged, p)
		}
	}
	return merged
}

func shortenHomePath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

func runScript(script, dir string, environ []string) error {
	c := exec.Command("sh", "-c", script)
	c.Dir = dir
	c.Env = environ
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}
