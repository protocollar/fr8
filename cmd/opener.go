package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/opener"
)

func init() {
	openerCmd.AddCommand(openerListCmd)
	openerCmd.AddCommand(openerAddCmd)
	openerCmd.AddCommand(openerRemoveCmd)
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
	Use:   "add <name> [executable]",
	Short: "Add a workspace opener",
	Long: `Add a named opener. The executable must be in $PATH.
If omitted, the name is used as the executable.

Examples:
  fr8 opener add rubymine
  fr8 opener add vscode code
  fr8 opener add cursor`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runOpenerAdd,
}

var openerRemoveCmd = &cobra.Command{
	Use:               "remove <name>",
	Aliases:           []string{"rm"},
	Short:             "Remove a workspace opener",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: openerNameCompletion,
	RunE:              runOpenerRemove,
}

func runOpenerList(cmd *cobra.Command, args []string) error {
	path, err := opener.DefaultPath()
	if err != nil {
		return err
	}

	openers, err := opener.Load(path)
	if err != nil {
		return fmt.Errorf("loading openers: %w", err)
	}

	if len(openers) == 0 {
		fmt.Println("No openers configured. Add one with: fr8 opener add <name> [executable]")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tCOMMAND")
	for _, o := range openers {
		fmt.Fprintf(w, "%s\t%s\n", o.Name, o.Command)
	}
	w.Flush()
	return nil
}

func runOpenerAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	command := name
	if len(args) > 1 {
		command = args[1]
	}

	path, err := opener.DefaultPath()
	if err != nil {
		return err
	}

	openers, err := opener.Load(path)
	if err != nil {
		return fmt.Errorf("loading openers: %w", err)
	}

	if opener.Find(openers, name) != nil {
		return fmt.Errorf("opener %q already exists (remove it first with: fr8 opener remove %s)", name, name)
	}

	openers = append(openers, opener.Opener{Name: name, Command: command})

	if err := opener.Save(path, openers); err != nil {
		return fmt.Errorf("saving openers: %w", err)
	}

	fmt.Printf("Added opener %q (%s)\n", name, command)
	return nil
}

func runOpenerRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	path, err := opener.DefaultPath()
	if err != nil {
		return err
	}

	openers, err := opener.Load(path)
	if err != nil {
		return fmt.Errorf("loading openers: %w", err)
	}

	found := false
	for i, o := range openers {
		if o.Name == name {
			openers = append(openers[:i], openers[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("opener %q not found", name)
	}

	if err := opener.Save(path, openers); err != nil {
		return fmt.Errorf("saving openers: %w", err)
	}

	fmt.Printf("Removed opener %q.\n", name)
	return nil
}

func openerNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	path, err := opener.DefaultPath()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	openers, err := opener.Load(path)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	names := make([]string, len(openers))
	for i, o := range openers {
		names[i] = o.Name
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
