package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/tmux"
)

func init() {
	workspaceCmd.AddCommand(psCmd)
}

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List all running fr8 workspace sessions",
	Args:  cobra.NoArgs,
	RunE:  runPS,
}

func runPS(cmd *cobra.Command, args []string) error {
	if err := tmux.Available(); err != nil {
		return err
	}

	sessions, err := tmux.ListFr8Sessions()
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		fmt.Println("No running workspaces.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "REPO\tWORKSPACE\tSESSION")
	for _, s := range sessions {
		fmt.Fprintf(w, "%s\t%s\t%s\n", s.Repo, s.Workspace, s.Name)
	}
	w.Flush()
	return nil
}
