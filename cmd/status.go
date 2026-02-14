package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/protocollar/fr8/internal/env"
	"github.com/protocollar/fr8/internal/git"
	"github.com/protocollar/fr8/internal/jsonout"
	"github.com/protocollar/fr8/internal/tmux"
)

func init() {
	workspaceCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show workspace details",
	Example: `  fr8 ws status
  fr8 ws status my-feature`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: workspaceNameCompletion,
	RunE:              runStatus,
}

type workspaceStatusJSON struct {
	Name      string            `json:"name"`
	Path      string            `json:"path"`
	Branch    string            `json:"branch"`
	Port      int               `json:"port"`
	PortEnd   int               `json:"port_end"`
	Dirty     bool              `json:"dirty"`
	Running   bool              `json:"running"`
	CreatedAt time.Time         `json:"created_at"`
	Env       map[string]string `json:"env"`
}

func (w workspaceStatusJSON) Concise() any {
	return struct {
		Name    string `json:"name"`
		Port    int    `json:"port"`
		Running bool   `json:"running"`
		Dirty   bool   `json:"dirty"`
		Path    string `json:"path"`
	}{Name: w.Name, Port: w.Port, Running: w.Running, Dirty: w.Dirty, Path: w.Path}
}

func runStatus(cmd *cobra.Command, args []string) error {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	ws, rootPath, _, err := resolveWorkspace(name)
	if err != nil {
		return err
	}

	defaultBranch, _ := git.DefaultBranch(rootPath)

	branch, _ := git.CurrentBranch(ws.Path)
	if branch == "" {
		branch = ws.Branch
	}

	dirty, _ := git.HasUncommittedChanges(ws.Path)

	running := false
	if tmux.Available() == nil {
		sessionName := tmux.SessionName(tmux.RepoName(rootPath), ws.Name)
		running = tmux.IsRunning(sessionName)
	}

	if jsonout.Enabled {
		vars := env.BuildFr8Only(ws, rootPath, defaultBranch)
		envMap := make(map[string]string)
		for _, v := range vars {
			parts := strings.SplitN(v, "=", 2)
			if len(parts) == 2 && strings.HasPrefix(parts[0], "FR8_") {
				envMap[parts[0]] = parts[1]
			}
		}
		return jsonout.Write(workspaceStatusJSON{
			Name:      ws.Name,
			Path:      ws.Path,
			Branch:    branch,
			Port:      ws.Port,
			PortEnd:   ws.Port + 9,
			Dirty:     dirty,
			Running:   running,
			CreatedAt: ws.CreatedAt,
			Env:       envMap,
		})
	}

	fmt.Printf("Workspace: %s\n", ws.Name)
	fmt.Printf("  Path:           %s\n", ws.Path)
	fmt.Printf("  Branch:         %s\n", branch)
	if dirty {
		fmt.Printf("  Status:         dirty (uncommitted changes)\n")
	} else {
		fmt.Printf("  Status:         clean\n")
	}
	fmt.Printf("  Port:           %d (range %d-%d)\n", ws.Port, ws.Port, ws.Port+9)
	fmt.Printf("  Created:        %s\n", ws.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Println()
	fmt.Printf("Environment:\n")
	fmt.Printf("  FR8_WORKSPACE_NAME  %s\n", ws.Name)
	fmt.Printf("  FR8_WORKSPACE_PATH  %s\n", ws.Path)
	fmt.Printf("  FR8_ROOT_PATH       %s\n", rootPath)
	fmt.Printf("  FR8_DEFAULT_BRANCH  %s\n", defaultBranch)
	fmt.Printf("  FR8_PORT            %d\n", ws.Port)

	// Process status
	fmt.Println()
	if tmux.Available() == nil {
		if running {
			fmt.Printf("Process: running (fr8 ws attach %s)\n", ws.Name)
		} else {
			fmt.Printf("Process: not running (fr8 ws run %s)\n", ws.Name)
		}
	}

	return nil
}
