package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/protocollar/fr8/internal/jsonout"
	"github.com/protocollar/fr8/internal/userconfig"
)

func init() {
	openerCmd.AddCommand(openerListCmd)
	openerCmd.AddCommand(openerAddCmd)
	openerCmd.AddCommand(openerRemoveCmd)
	openerCmd.AddCommand(openerSetDefaultCmd)
	rootCmd.AddCommand(openerCmd)
}

var openerCmd = &cobra.Command{
	Use:   "opener",
	Short: "Manage workspace openers",
	Long:  "Configure external tools for opening workspaces (e.g., VSCode, Cursor, terminal).",
}

var openerListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List configured openers",
	Args:    cobra.NoArgs,
	RunE:    runOpenerList,
}

var openerAddCmd = &cobra.Command{
	Use:   "add <name> [command...]",
	Short: "Add a workspace opener",
	Long: `Add a named opener. The executable must be in $PATH.
If no command is given, the name is used as the executable.
Commands can include arguments (e.g. "code --new-window").

Examples:
  fr8 opener add rubymine
  fr8 opener add vscode code
  fr8 opener add vscode-nw code --new-window
  fr8 opener add cursor`,
	Args: cobra.MinimumNArgs(1),
	RunE: runOpenerAdd,
}

var openerSetDefaultCmd = &cobra.Command{
	Use:               "set-default <name>",
	Short:             "Set the default opener",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: openerNameCompletion,
	RunE:              runOpenerSetDefault,
}

var openerRemoveCmd = &cobra.Command{
	Use:               "remove <name>",
	Aliases:           []string{"rm"},
	Short:             "Remove a workspace opener",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: openerNameCompletion,
	RunE:              runOpenerRemove,
}

func loadUserConfig() (*userconfig.Config, string, error) {
	path, err := userconfig.DefaultPath()
	if err != nil {
		return nil, "", err
	}
	cfg, err := userconfig.Load(path)
	if err != nil {
		return nil, "", fmt.Errorf("loading config: %w", err)
	}
	return cfg, path, nil
}

func runOpenerList(cmd *cobra.Command, args []string) error {
	cfg, _, err := loadUserConfig()
	if err != nil {
		return err
	}

	if jsonout.Enabled {
		openers := cfg.Openers
		if openers == nil {
			openers = []userconfig.Opener{}
		}
		return jsonout.Write(openers)
	}

	if len(cfg.Openers) == 0 {
		fmt.Println("No openers configured. Add one with: fr8 opener add <name> [command...]")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tCOMMAND\tDEFAULT")
	for _, o := range cfg.Openers {
		def := ""
		if o.Default {
			def = "(default)"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", o.Name, o.Command, def)
	}
	_ = w.Flush()
	return nil
}

func runOpenerAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	command := name
	if len(args) > 1 {
		command = strings.Join(args[1:], " ")
	}

	cfg, path, err := loadUserConfig()
	if err != nil {
		return err
	}

	o := userconfig.Opener{Name: name, Command: command}
	if err := cfg.AddOpener(o); err != nil {
		return err
	}

	if err := cfg.Save(path); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	if jsonout.Enabled {
		return jsonout.Write(struct {
			Action string          `json:"action"`
			Opener userconfig.Opener `json:"opener"`
		}{Action: "added", Opener: o})
	}

	fmt.Printf("Added opener %q (%s)\n", name, command)

	// Validate the executable exists in $PATH
	parts := strings.Fields(command)
	if _, err := exec.LookPath(parts[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %q not found in $PATH\n", parts[0])
	}

	return nil
}

func runOpenerSetDefault(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, path, err := loadUserConfig()
	if err != nil {
		return err
	}

	if err := cfg.SetDefaultOpener(name); err != nil {
		return err
	}

	if err := cfg.Save(path); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	if jsonout.Enabled {
		return jsonout.Write(struct {
			Action string `json:"action"`
			Name   string `json:"name"`
		}{Action: "set_default", Name: name})
	}

	fmt.Printf("Set %q as default opener.\n", name)
	return nil
}

func runOpenerRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, path, err := loadUserConfig()
	if err != nil {
		return err
	}

	if err := cfg.RemoveOpener(name); err != nil {
		return err
	}

	if err := cfg.Save(path); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	if jsonout.Enabled {
		return jsonout.Write(struct {
			Action string `json:"action"`
			Name   string `json:"name"`
		}{Action: "removed", Name: name})
	}

	fmt.Printf("Removed opener %q.\n", name)
	return nil
}

func openerNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cfg, _, err := loadUserConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return cfg.OpenerNames(), cobra.ShellCompDirectiveNoFileComp
}
