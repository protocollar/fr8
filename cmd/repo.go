package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/git"
	"github.com/thomascarr/fr8/internal/registry"
	"github.com/thomascarr/fr8/internal/state"
)

var repoAddName string
var repoListWorkspaces bool

func init() {
	repoAddCmd.Flags().StringVar(&repoAddName, "name", "", "override the repo name (defaults to directory name)")
	repoListCmd.Flags().BoolVarP(&repoListWorkspaces, "workspaces", "w", false, "show workspaces for each repo")

	repoCmd.AddCommand(repoListCmd)
	repoCmd.AddCommand(repoAddCmd)
	repoCmd.AddCommand(repoRemoveCmd)
	rootCmd.AddCommand(repoCmd)
}

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage the global repo registry",
	Long:  "Register, list, and remove repos from the global fr8 registry.",
}

var repoListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List registered repos",
	Args:    cobra.NoArgs,
	RunE:    runRepoList,
}

var repoAddCmd = &cobra.Command{
	Use:   "add [path]",
	Short: "Register a repo",
	Long:  "Register a git repo in the global registry. Defaults to the current directory.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRepoAdd,
}

var repoRemoveCmd = &cobra.Command{
	Use:               "remove <name>",
	Aliases:           []string{"rm"},
	Short:             "Unregister a repo",
	Long:              "Remove a repo from the global registry. Does not touch git data.",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: repoNameCompletion,
	RunE:              runRepoRemove,
}

func runRepoList(cmd *cobra.Command, args []string) error {
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

	if !repoListWorkspaces {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tPATH")
		for _, repo := range reg.Repos {
			fmt.Fprintf(w, "%s\t%s\n", repo.Name, repo.Path)
		}
		w.Flush()
		return nil
	}

	// Show workspaces for each repo
	for _, repo := range reg.Repos {
		fmt.Printf("%s (%s)\n", repo.Name, repo.Path)

		commonDir, err := git.CommonDir(repo.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  (unable to read git data: %v)\n", err)
			continue
		}

		st, err := state.Load(commonDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  (unable to load state: %v)\n", err)
			continue
		}

		if len(st.Workspaces) == 0 {
			fmt.Println("  (no workspaces)")
			continue
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, ws := range st.Workspaces {
			fmt.Fprintf(w, "  %s\t%s\t%d\n", ws.Name, ws.Branch, ws.Port)
		}
		w.Flush()
	}

	return nil
}

func runRepoAdd(cmd *cobra.Command, args []string) error {
	var dir string
	if len(args) > 0 {
		dir = args[0]
	} else {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	// Resolve symlinks to canonicalize
	dir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolving absolute path: %w", err)
	}

	// Validate it's a git repo
	if !git.IsInsideWorkTree(dir) {
		return fmt.Errorf("%s is not a git repository", dir)
	}

	// Resolve to root worktree
	rootPath, err := git.RootWorktreePath(dir)
	if err != nil {
		return fmt.Errorf("finding root worktree: %w", err)
	}

	name := repoAddName
	if name == "" {
		name = filepath.Base(rootPath)
	}

	regPath, err := registry.DefaultPath()
	if err != nil {
		return err
	}

	reg, err := registry.Load(regPath)
	if err != nil {
		return fmt.Errorf("loading registry: %w", err)
	}

	if err := reg.Add(registry.Repo{Name: name, Path: rootPath}); err != nil {
		return err
	}

	if err := reg.Save(regPath); err != nil {
		return fmt.Errorf("saving registry: %w", err)
	}

	fmt.Printf("Registered %q → %s\n", name, rootPath)
	return nil
}

func runRepoRemove(cmd *cobra.Command, args []string) error {
	regPath, err := registry.DefaultPath()
	if err != nil {
		return err
	}

	reg, err := registry.Load(regPath)
	if err != nil {
		return fmt.Errorf("loading registry: %w", err)
	}

	if err := reg.Remove(args[0]); err != nil {
		return err
	}

	if err := reg.Save(regPath); err != nil {
		return fmt.Errorf("saving registry: %w", err)
	}

	fmt.Printf("Removed %q from registry.\n", args[0])
	return nil
}

// autoRegisterRepo silently registers a repo if not already present.
// Skips on name collision — never blocks other commands.
func autoRegisterRepo(rootPath string) {
	regPath, err := registry.DefaultPath()
	if err != nil {
		return
	}

	reg, err := registry.Load(regPath)
	if err != nil {
		return
	}

	// Already registered by path
	if reg.FindByPath(rootPath) != nil {
		return
	}

	name := filepath.Base(rootPath)

	// Name collision — skip silently
	if reg.Find(name) != nil {
		return
	}

	reg.Repos = append(reg.Repos, registry.Repo{Name: name, Path: rootPath})
	if err := reg.Save(regPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to auto-register repo: %v\n", err)
	}
}

// repoNameCompletion returns a ValidArgsFunction that completes repo names.
func repoNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	regPath, err := registry.DefaultPath()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	reg, err := registry.Load(regPath)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return reg.Names(), cobra.ShellCompDirectiveNoFileComp
}
