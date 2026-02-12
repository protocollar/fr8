package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	workspaceCmd.AddCommand(cdCmd)
}

var cdCmd = &cobra.Command{
	Use:   "cd [name]",
	Short: "Print workspace path (for use with cd)",
	Long: `Prints the workspace path to stdout for use with shell cd:
  cd $(fr8 ws cd myws)

Or add a shell function to ~/.zshrc:
  fr8cd() { cd "$(fr8 ws cd "$@")"; }`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: workspaceNameCompletion,
	RunE:              runCd,
}

func runCd(cmd *cobra.Command, args []string) error {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	ws, _, _, err := resolveWorkspace(name)
	if err != nil {
		return err
	}

	fmt.Print(ws.Path)
	return nil
}
