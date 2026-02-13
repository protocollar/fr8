package cmd

import "github.com/spf13/cobra"

var resolveRepo string

var workspaceCmd = &cobra.Command{
	Use:     "workspace",
	Aliases: []string{"ws"},
	Short:   "Manage workspaces",
}

func init() {
	workspaceCmd.PersistentFlags().StringVar(&resolveRepo, "repo", "", "target a specific registered repo")
	_ = workspaceCmd.RegisterFlagCompletionFunc("repo", repoNameCompletion)
	rootCmd.AddCommand(workspaceCmd)
}
