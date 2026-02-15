package env

import (
	"fmt"
	"os"

	"github.com/protocollar/fr8/internal/registry"
)

// Build returns a complete environment variable slice for running scripts
// in the given workspace. Includes both FR8_* and CONDUCTOR_* (compat) vars,
// merged with the current process environment.
func Build(ws *registry.Workspace, rootPath, defaultBranch string) []string {
	fr8Vars := map[string]string{
		"FR8_WORKSPACE_NAME": ws.Name,
		"FR8_WORKSPACE_PATH": ws.Path,
		"FR8_ROOT_PATH":      rootPath,
		"FR8_DEFAULT_BRANCH": defaultBranch,
		"FR8_PORT":           fmt.Sprintf("%d", ws.Port),
		// Conductor compatibility
		"CONDUCTOR_WORKSPACE_NAME": ws.Name,
		"CONDUCTOR_WORKSPACE_PATH": ws.Path,
		"CONDUCTOR_ROOT_PATH":      rootPath,
		"CONDUCTOR_DEFAULT_BRANCH": defaultBranch,
		"CONDUCTOR_PORT":           fmt.Sprintf("%d", ws.Port),
	}

	// Start with current env, then override with fr8 vars.
	envMap := make(map[string]string)
	for _, e := range os.Environ() {
		for i := range e {
			if e[i] == '=' {
				envMap[e[:i]] = e[i+1:]
				break
			}
		}
	}
	for k, v := range fr8Vars {
		envMap[k] = v
	}

	result := make([]string, 0, len(envMap))
	for k, v := range envMap {
		result = append(result, k+"="+v)
	}
	return result
}

// BuildFr8Only returns only the FR8_* and CONDUCTOR_* environment variables
// (not merged with the current process env). Used for tmux sessions where
// the user's shell environment is inherited automatically.
func BuildFr8Only(ws *registry.Workspace, rootPath, defaultBranch string) []string {
	return []string{
		"FR8_WORKSPACE_NAME=" + ws.Name,
		"FR8_WORKSPACE_PATH=" + ws.Path,
		"FR8_ROOT_PATH=" + rootPath,
		"FR8_DEFAULT_BRANCH=" + defaultBranch,
		fmt.Sprintf("FR8_PORT=%d", ws.Port),
		"CONDUCTOR_WORKSPACE_NAME=" + ws.Name,
		"CONDUCTOR_WORKSPACE_PATH=" + ws.Path,
		"CONDUCTOR_ROOT_PATH=" + rootPath,
		"CONDUCTOR_DEFAULT_BRANCH=" + defaultBranch,
		fmt.Sprintf("CONDUCTOR_PORT=%d", ws.Port),
	}
}
