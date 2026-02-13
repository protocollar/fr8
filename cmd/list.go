package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/git"
	"github.com/thomascarr/fr8/internal/registry"
	"github.com/thomascarr/fr8/internal/state"
	"github.com/thomascarr/fr8/internal/tmux"
)

var listAll bool
var listRunning bool
var listDirty bool
var listMerged bool

func init() {
	listCmd.Flags().BoolVarP(&listAll, "all", "a", false, "list workspaces across all registered repos")
	listCmd.Flags().BoolVar(&listRunning, "running", false, "only show running workspaces")
	listCmd.Flags().BoolVar(&listDirty, "dirty", false, "only show workspaces with uncommitted changes")
	listCmd.Flags().BoolVar(&listMerged, "merged", false, "only show workspaces whose branch is merged")
	workspaceCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all workspaces",
	Example: `  fr8 ws list
  fr8 ws list --all
  fr8 ws list --running
  fr8 ws list --dirty
  fr8 ws list --merged`,
	Args: cobra.NoArgs,
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
	if listAll {
		return runListAll()
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	commonDir, err := git.CommonDir(cwd)
	if err != nil {
		// Not inside a git repo — fall back to listing all registered repos
		return runListAll()
	}

	st, err := state.Load(commonDir)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	// Reconcile: remove workspaces whose paths no longer exist
	reconcile(st, cwd)

	if len(st.Workspaces) == 0 {
		fmt.Println("No workspaces. Create one with: fr8 ws new")
		return nil
	}

	// Determine repo name for tmux session lookup
	hasTmux := tmux.Available() == nil
	rootPath, _ := git.RootWorktreePath(cwd)
	repoName := filepath.Base(rootPath)
	defaultBranch, _ := git.DefaultBranch(rootPath)
	hasFilters := listRunning || listDirty || listMerged

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tBRANCH\tPORT\tRUNNING\tPATH")
	for _, ws := range st.Workspaces {
		running := false
		if hasTmux {
			sessionName := tmux.SessionName(repoName, ws.Name)
			running = tmux.IsRunning(sessionName)
		}

		if hasFilters {
			if listRunning && !running {
				continue
			}
			if listDirty {
				dirty, _ := git.HasUncommittedChanges(ws.Path)
				if !dirty {
					continue
				}
			}
			if listMerged && defaultBranch != "" {
				merged, _ := git.IsMerged(ws.Path, ws.Branch, defaultBranch)
				if !merged {
					continue
				}
			}
		}

		runMark := ""
		if running {
			runMark = "●"
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n", ws.Name, ws.Branch, ws.Port, runMark, ws.Path)
	}
	w.Flush()

	// Save reconciled state
	st.Save(commonDir)

	return nil
}

func runListAll() error {
	regPath, err := registry.DefaultPath()
	if err != nil {
		return err
	}

	reg, err := registry.Load(regPath)
	if err != nil {
		return fmt.Errorf("loading registry: %w", err)
	}

	if len(reg.Repos) == 0 {
		fmt.Println("No repos registered. Add one with: fr8 repo add")
		return nil
	}

	hasTmux := tmux.Available() == nil
	hasFilters := listRunning || listDirty || listMerged

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "REPO\tNAME\tBRANCH\tPORT\tRUNNING\tPATH")

	for _, repo := range reg.Repos {
		commonDir, err := git.CommonDir(repo.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: unable to read %s: %v\n", repo.Name, err)
			continue
		}

		st, err := state.Load(commonDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: unable to load state for %s: %v\n", repo.Name, err)
			continue
		}

		rootPath, _ := git.RootWorktreePath(repo.Path)
		defaultBranch, _ := git.DefaultBranch(rootPath)

		for _, ws := range st.Workspaces {
			running := false
			if hasTmux {
				sessionName := tmux.SessionName(repo.Name, ws.Name)
				running = tmux.IsRunning(sessionName)
			}

			if hasFilters {
				if listRunning && !running {
					continue
				}
				if listDirty {
					dirty, _ := git.HasUncommittedChanges(ws.Path)
					if !dirty {
						continue
					}
				}
				if listMerged && defaultBranch != "" {
					merged, _ := git.IsMerged(ws.Path, ws.Branch, defaultBranch)
					if !merged {
						continue
					}
				}
			}

			runMark := ""
			if running {
				runMark = "●"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n", repo.Name, ws.Name, ws.Branch, ws.Port, runMark, ws.Path)
		}
	}

	w.Flush()
	return nil
}

func reconcile(st *state.State, cwd string) {
	gitWorktrees, err := git.WorktreeList(cwd)
	if err != nil {
		return // can't reconcile, leave state as-is
	}

	wtPaths := make(map[string]bool, len(gitWorktrees))
	for _, wt := range gitWorktrees {
		wtPaths[wt.Path] = true
	}

	var remaining []state.Workspace
	for _, ws := range st.Workspaces {
		if wtPaths[ws.Path] {
			remaining = append(remaining, ws)
		}
	}
	st.Workspaces = remaining
}
