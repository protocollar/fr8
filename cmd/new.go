package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
var noSetup bool
var newRepo string

func init() {
	newCmd.Flags().StringVarP(&newBranch, "branch", "b", "", "branch name (creates new branch if it doesn't exist)")
	newCmd.Flags().BoolVar(&noSetup, "no-setup", false, "skip running the setup script")
	newCmd.Flags().StringVar(&newRepo, "repo", "", "create workspace in a registered repo (by name)")
	newCmd.RegisterFlagCompletionFunc("repo", repoNameCompletion)
	workspaceCmd.AddCommand(newCmd)
}

var newCmd = &cobra.Command{
	Use:   "new [name]",
	Short: "Create a new workspace",
	Long:  "Creates a git worktree, allocates a port range, syncs files, and runs the setup script.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runNew,
}

func runNew(cmd *cobra.Command, args []string) error {
	var rootPath, commonDir string

	if newRepo != "" {
		// Resolve from registry
		regPath, err := registry.DefaultPath()
		if err != nil {
			return err
		}
		reg, err := registry.Load(regPath)
		if err != nil {
			return fmt.Errorf("loading registry: %w", err)
		}
		repo := reg.Find(newRepo)
		if repo == nil {
			return fmt.Errorf("repo %q not found in registry (see: fr8 repo list)", newRepo)
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

	cfg, err := config.Load(rootPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	st, err := state.Load(commonDir)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	// Workspace name
	var wsName string
	if len(args) > 0 {
		wsName = args[0]
		if st.Find(wsName) != nil {
			return fmt.Errorf("workspace %q already exists", wsName)
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

	// Branch
	branch := newBranch
	createBranch := false
	if branch == "" {
		branch = wsName
		createBranch = true
	} else {
		if !git.BranchExists(rootPath, branch) {
			createBranch = true
		}
	}

	// Port — collect ports from all registered repos to avoid cross-repo conflicts
	globalPorts := allAllocatedPorts()
	localPorts := st.AllocatedPorts()
	allocatedPort, err := port.Allocate(mergePorts(globalPorts, localPorts), cfg.BasePort, cfg.PortRange)
	if err != nil {
		return fmt.Errorf("allocating port: %w", err)
	}

	// Worktree path
	wtBase := config.ResolveWorktreePath(cfg, rootPath)
	wsPath := filepath.Join(wtBase, wsName)

	// Create worktree
	fmt.Printf("Creating workspace %q...\n", wsName)
	if err := os.MkdirAll(wtBase, 0755); err != nil {
		return fmt.Errorf("creating worktree directory: %w", err)
	}

	if err := git.WorktreeAdd(rootPath, wsPath, branch, createBranch, startPoint); err != nil {
		return fmt.Errorf("creating worktree: %w", err)
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
		return fmt.Errorf("saving workspace: %w", err)
	}
	if err := st.Save(commonDir); err != nil {
		git.WorktreeRemove(rootPath, wsPath)
		return fmt.Errorf("saving state: %w", err)
	}

	// Auto-register repo in global registry
	autoRegisterRepo(rootPath)

	// Sync files
	fmt.Println("Syncing files...")
	if err := filesync.Sync(rootPath, wsPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: file sync failed: %v\n", err)
	}

	// Run setup script
	if !noSetup && cfg.Scripts.Setup != "" {
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
	fmt.Printf("  Port:   %d\n", ws.Port)
	fmt.Printf("  Path:   %s\n", ws.Path)

	return nil
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

func runScript(script, dir string, environ []string) error {
	c := exec.Command("sh", "-c", script)
	c.Dir = dir
	c.Env = environ
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}
