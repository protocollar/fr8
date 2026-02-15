package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/protocollar/fr8/internal/git"
	"github.com/protocollar/fr8/internal/registry"
)

func init() {
	rootCmd.AddCommand(completionCmd)
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completions for fr8.

  # Bash
  source <(fr8 completion bash)

  # Zsh
  fr8 completion zsh > "${fpath[1]}/_fr8"

  # Fish
  fr8 completion fish | source`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish"},
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		default:
			return cmd.Help()
		}
	},
}

// workspaceNameCompletion returns a ValidArgsFunction that completes workspace names.
func workspaceNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cwd, err := os.Getwd()
	if err != nil {
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

	// Try CWD match
	repo := reg.FindRepoByWorkspacePath(cwd)
	if repo == nil {
		if git.IsInsideWorkTree(cwd) {
			rootPath, err := git.RootWorktreePath(cwd)
			if err == nil {
				repo = reg.FindByPath(rootPath)
			}
		}
	}
	if repo != nil {
		return repo.WorkspaceNames(), cobra.ShellCompDirectiveNoFileComp
	}

	// Fall back to all registered workspace names
	return reg.AllWorkspaceNames(), cobra.ShellCompDirectiveNoFileComp
}

// allRegistryWorkspaceNames returns workspace names from all registered repos.
func allRegistryWorkspaceNames() []string {
	regPath, err := registry.DefaultPath()
	if err != nil {
		return nil
	}
	reg, err := registry.Load(regPath)
	if err != nil {
		return nil
	}
	return reg.AllWorkspaceNames()
}
