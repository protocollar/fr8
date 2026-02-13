package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/jsonout"
	"github.com/thomascarr/fr8/internal/tmux"
)

var stopAll bool
var stopIfRunning bool

func init() {
	stopCmd.Flags().BoolVarP(&stopAll, "all", "A", false, "Stop all running fr8 sessions")
	stopCmd.Flags().BoolVar(&stopIfRunning, "if-running", false, "succeed silently if not running")
	workspaceCmd.AddCommand(stopCmd)
}

var stopCmd = &cobra.Command{
	Use:               "stop [name]",
	Short:             "Stop a workspace's background tmux session",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: workspaceNameCompletion,
	RunE:              runStop,
}

func runStop(cmd *cobra.Command, args []string) error {
	if stopAll {
		if len(args) > 0 {
			return fmt.Errorf("cannot use --all with a workspace name")
		}
		return runStopAll()
	}

	if err := tmux.Available(); err != nil {
		return err
	}

	var name string
	if len(args) > 0 {
		name = args[0]
	}

	ws, rootPath, _, err := resolveWorkspace(name)
	if err != nil {
		return err
	}

	sessionName := tmux.SessionName(tmux.RepoName(rootPath), ws.Name)
	if !tmux.IsRunning(sessionName) {
		if stopIfRunning {
			if jsonout.Enabled {
				return jsonout.Write(struct {
					Action    string `json:"action"`
					Workspace string `json:"workspace"`
					Session   string `json:"session"`
				}{Action: "not_running", Workspace: ws.Name, Session: sessionName})
			}
			fmt.Printf("Workspace %q is not running.\n", ws.Name)
			return nil
		}
		if jsonout.Enabled {
			return jsonout.Write(struct {
				Action    string `json:"action"`
				Workspace string `json:"workspace"`
				Session   string `json:"session"`
			}{Action: "not_running", Workspace: ws.Name, Session: sessionName})
		}
		fmt.Printf("Workspace %q is not running.\n", ws.Name)
		return nil
	}

	if err := tmux.Stop(sessionName); err != nil {
		return err
	}

	if jsonout.Enabled {
		return jsonout.Write(struct {
			Action    string `json:"action"`
			Workspace string `json:"workspace"`
			Session   string `json:"session"`
		}{Action: "stopped", Workspace: ws.Name, Session: sessionName})
	}

	fmt.Printf("Stopped %q.\n", ws.Name)
	return nil
}

func runStopAll() error {
	if err := tmux.Available(); err != nil {
		return err
	}

	sessions, err := tmux.ListFr8Sessions()
	if err != nil {
		return fmt.Errorf("listing sessions: %w", err)
	}

	if len(sessions) == 0 {
		if jsonout.Enabled {
			return jsonout.Write(struct {
				Stopped []string        `json:"stopped"`
				Failed  []runFailedItem `json:"failed"`
			}{Stopped: []string{}, Failed: []runFailedItem{}})
		}
		fmt.Println("No running fr8 sessions.")
		return nil
	}

	var stopped []string
	var failed []runFailedItem
	for _, s := range sessions {
		if err := tmux.Stop(s.Name); err != nil {
			if !jsonout.Enabled {
				fmt.Fprintf(os.Stderr, "Warning: failed to stop %q: %v\n", s.Name, err)
			}
			failed = append(failed, runFailedItem{Workspace: s.Workspace, Error: err.Error()})
			continue
		}
		if !jsonout.Enabled {
			fmt.Printf("Stopped %q\n", s.Name)
		}
		stopped = append(stopped, s.Workspace)
	}

	if jsonout.Enabled {
		if failed == nil {
			failed = []runFailedItem{}
		}
		return jsonout.Write(struct {
			Stopped []string        `json:"stopped"`
			Failed  []runFailedItem `json:"failed"`
		}{Stopped: orEmpty(stopped), Failed: failed})
	}

	fmt.Printf("Stopped %d session(s).\n", len(stopped))
	return nil
}
