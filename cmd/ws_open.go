package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/jsonout"
	"github.com/thomascarr/fr8/internal/opener"
)

var wsOpenOpener string

func init() {
	wsOpenCmd.Flags().StringVar(&wsOpenOpener, "opener", "", "opener to use (see: fr8 opener list)")
	wsOpenCmd.RegisterFlagCompletionFunc("opener", openerNameCompletion)
	workspaceCmd.AddCommand(wsOpenCmd)
}

var wsOpenCmd = &cobra.Command{
	Use:   "open [name]",
	Short: "Open a workspace with a configured opener",
	Example: `  fr8 ws open
  fr8 ws open my-feature
  fr8 ws open --opener vscode`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: workspaceNameCompletion,
	RunE:              runWsOpen,
}

func runWsOpen(cmd *cobra.Command, args []string) error {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	ws, _, _, err := resolveWorkspace(name)
	if err != nil {
		return err
	}

	path, err := opener.DefaultPath()
	if err != nil {
		return err
	}

	openers, err := opener.Load(path)
	if err != nil {
		return fmt.Errorf("loading openers: %w", err)
	}

	if len(openers) == 0 {
		return fmt.Errorf("no openers configured — add one with: fr8 opener add <name> [executable]")
	}

	var o *opener.Opener
	if wsOpenOpener != "" {
		o = opener.Find(openers, wsOpenOpener)
		if o == nil {
			return fmt.Errorf("opener %q not found (see: fr8 opener list)", wsOpenOpener)
		}
	} else if len(openers) == 1 {
		o = &openers[0]
	} else if d := opener.FindDefault(openers); d != nil {
		o = d
	} else {
		if jsonout.Enabled {
			return fmt.Errorf("multiple openers configured; specify one with --opener <name> (or set a default with: fr8 opener set-default <name>)")
		}
		fmt.Println("Multiple openers configured:")
		for _, op := range openers {
			fmt.Printf("  - %s\n", op.Name)
		}
		return fmt.Errorf("specify one with --opener <name> (or set a default with: fr8 opener set-default <name>)")
	}

	if err := opener.Run(*o, ws.Path); err != nil {
		return fmt.Errorf("running opener %q: %w", o.Name, err)
	}

	if jsonout.Enabled {
		return jsonout.Write(struct {
			Action    string `json:"action"`
			Workspace string `json:"workspace"`
			Opener    string `json:"opener"`
			Path      string `json:"path"`
		}{Action: "opened", Workspace: ws.Name, Opener: o.Name, Path: ws.Path})
	}

	fmt.Printf("Opened %q with %s\n", ws.Name, o.Name)
	return nil
}
