package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/protocollar/fr8/internal/jsonout"
	"github.com/protocollar/fr8/internal/opener"
	"github.com/protocollar/fr8/internal/userconfig"
)

var wsOpenOpener string

func init() {
	wsOpenCmd.Flags().StringVar(&wsOpenOpener, "opener", "", "opener to use (see: fr8 opener list)")
	_ = wsOpenCmd.RegisterFlagCompletionFunc("opener", openerNameCompletion)
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

	ws, _, err := resolveWorkspace(name)
	if err != nil {
		return err
	}

	cfg, _, err := loadUserConfig()
	if err != nil {
		return err
	}

	if len(cfg.Openers) == 0 {
		return fmt.Errorf("no openers configured — add one with: fr8 opener add <name> [executable]")
	}

	var o *userconfig.Opener
	if wsOpenOpener != "" {
		o = cfg.FindOpener(wsOpenOpener)
		if o == nil {
			return fmt.Errorf("opener %q not found (see: fr8 opener list)", wsOpenOpener)
		}
	} else if len(cfg.Openers) == 1 {
		o = &cfg.Openers[0]
	} else if d := cfg.FindDefaultOpener(); d != nil {
		o = d
	} else {
		if jsonout.Enabled {
			return fmt.Errorf("multiple openers configured; specify one with --opener <name> (or set a default with: fr8 opener set-default <name>)")
		}
		fmt.Println("Multiple openers configured:")
		for _, op := range cfg.Openers {
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
