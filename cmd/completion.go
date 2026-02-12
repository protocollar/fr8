package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/git"
	"github.com/thomascarr/fr8/internal/registry"
	"github.com/thomascarr/fr8/internal/state"
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

	commonDir, err := git.CommonDir(cwd)
	if err != nil {
		// Not inside a git repo — search all registered repos
		return allRegistryWorkspaceNames(), cobra.ShellCompDirectiveNoFileComp
	}

	st, err := state.Load(commonDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return st.Names(), cobra.ShellCompDirectiveNoFileComp
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

	var names []string
	for _, repo := range reg.Repos {
		commonDir, err := git.CommonDir(repo.Path)
		if err != nil {
			continue
		}

		st, err := state.Load(commonDir)
		if err != nil {
			continue
		}

		names = append(names, st.Names()...)
	}
	return names
}
