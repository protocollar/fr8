package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/port"
	"github.com/thomascarr/fr8/internal/state"
)

func init() {
	workspaceCmd.AddCommand(browserCmd)
}

var browserCmd = &cobra.Command{
	Use:               "browser [name]",
	Short:             "Open workspace dev server in the browser",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: workspaceNameCompletion,
	RunE:              runBrowser,
}

func runBrowser(cmd *cobra.Command, args []string) error {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	ws, _, _, err := resolveWorkspace(name)
	if err != nil {
		return err
	}

	return openWorkspaceBrowser(ws)
}

func openWorkspaceBrowser(ws *state.Workspace) error {
	if port.IsFree(ws.Port) {
		fmt.Printf("Warning: nothing seems to be listening on :%d\n", ws.Port)
	}

	url := fmt.Sprintf("http://localhost:%d", ws.Port)
	fmt.Printf("Opening %s for workspace %q\n", url, ws.Name)
	return openBrowser(url)
}
