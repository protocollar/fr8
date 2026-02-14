package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/protocollar/fr8/internal/env"
	"github.com/protocollar/fr8/internal/git"
	"github.com/protocollar/fr8/internal/jsonout"
)

func init() {
	workspaceCmd.AddCommand(envCmd)
}

var envCmd = &cobra.Command{
	Use:               "env [name]",
	Short:             "Print workspace environment variables as export statements",
	Long:              "Prints FR8_* and CONDUCTOR_* variables suitable for eval \"$(fr8 ws env)\".",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: workspaceNameCompletion,
	RunE:              runEnv,
}

func runEnv(cmd *cobra.Command, args []string) error {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	ws, rootPath, _, err := resolveWorkspace(name)
	if err != nil {
		return err
	}

	defaultBranch, _ := git.DefaultBranch(rootPath)

	vars := env.BuildFr8Only(ws, rootPath, defaultBranch)

	if jsonout.Enabled {
		// Output FR8_* vars only as a map (skip CONDUCTOR_* compat vars)
		envMap := make(map[string]string)
		for _, v := range vars {
			parts := strings.SplitN(v, "=", 2)
			if len(parts) == 2 && strings.HasPrefix(parts[0], "FR8_") {
				envMap[parts[0]] = parts[1]
			}
		}
		return jsonout.Write(envMap)
	}

	for _, v := range vars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			fmt.Printf("export %s=%q\n", parts[0], parts[1])
		}
	}

	return nil
}
