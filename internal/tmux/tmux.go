package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// execFunc is the function used to replace the current process.
// Defaults to syscall.Exec; overridden in tests.
var execFunc = syscall.Exec

// Session represents a running fr8 tmux session.
type Session struct {
	Name      string // full session name, e.g. "fr8/myrepo/cool-workspace"
	Repo      string
	Workspace string
}

// Available checks whether tmux is installed and runnable.
func Available() error {
	_, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux is not installed (brew install tmux / apt install tmux)")
	}
	return nil
}

// SessionName returns the fr8 session name for a workspace.
// Format: fr8/<repo>/<workspace>
func SessionName(repoName, wsName string) string {
	return fmt.Sprintf("fr8/%s/%s", repoName, wsName)
}

// RepoName extracts the repo directory name from a root path.
func RepoName(rootPath string) string {
	return filepath.Base(rootPath)
}

// Start creates a new detached tmux session running the given command.
// envVars should be only the FR8_*/CONDUCTOR_* key=value pairs to export;
// the user's shell environment is inherited by tmux automatically.
func Start(name, dir, command string, envVars []string) error {
	if IsRunning(name) {
		return fmt.Errorf("session %q is already running", name)
	}

	// Build export commands for FR8/CONDUCTOR env vars only
	var exports []string
	for _, e := range envVars {
		exports = append(exports, fmt.Sprintf("export %s", shellescape(e)))
	}

	// Combine exports + exec into a single shell command
	var sessionCmd string
	if len(exports) > 0 {
		sessionCmd = strings.Join(exports, "; ") + "; exec " + command
	} else {
		sessionCmd = "exec " + command
	}

	cmd := exec.Command("tmux", "new-session", "-d", "-s", name, "-c", dir, sessionCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("starting tmux session: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Stop kills a tmux session. Returns nil if the session doesn't exist.
func Stop(name string) error {
	if !IsRunning(name) {
		return nil
	}
	cmd := exec.Command("tmux", "kill-session", "-t", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("stopping tmux session: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// IsRunning returns true if a tmux session with the given name exists.
func IsRunning(name string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", name)
	return cmd.Run() == nil
}

// Attach replaces the current process with tmux attach.
// This mirrors the syscall.Exec pattern used by fr8 ws shell.
func Attach(name string) error {
	if !IsRunning(name) {
		return fmt.Errorf("session %q is not running", name)
	}

	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found: %w", err)
	}

	return execFunc(tmuxPath, []string{"tmux", "attach-session", "-t", name}, os.Environ())
}

// CapturePanes captures recent output from a tmux session's pane.
// lines controls how many lines of scrollback to capture.
func CapturePanes(name string, lines int) (string, error) {
	if !IsRunning(name) {
		return "", fmt.Errorf("session %q is not running", name)
	}

	startLine := fmt.Sprintf("-%d", lines)
	cmd := exec.Command("tmux", "capture-pane", "-t", name, "-p", "-S", startLine)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("capturing pane output: %w", err)
	}
	return string(out), nil
}

// ListFr8Sessions returns all tmux sessions with the "fr8/" prefix.
func ListFr8Sessions() ([]Session, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	out, err := cmd.Output()
	if err != nil {
		// tmux returns error when no server is running — that's fine, no sessions
		return nil, nil
	}

	var sessions []Session
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "fr8/") {
			continue
		}

		// Parse fr8/<repo>/<workspace>
		parts := strings.SplitN(line, "/", 3)
		if len(parts) != 3 {
			continue
		}
		sessions = append(sessions, Session{
			Name:      line,
			Repo:      parts[1],
			Workspace: parts[2],
		})
	}
	return sessions, nil
}

// shellescape wraps a KEY=VALUE string for safe use in a shell export statement.
func shellescape(kv string) string {
	idx := strings.IndexByte(kv, '=')
	if idx < 0 {
		return kv
	}
	key := kv[:idx]
	val := kv[idx+1:]
	// Single-quote the value, escaping any embedded single quotes
	val = strings.ReplaceAll(val, "'", "'\"'\"'")
	return fmt.Sprintf("%s='%s'", key, val)
}
