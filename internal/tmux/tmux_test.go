package tmux

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestSessionName(t *testing.T) {
	tests := []struct {
		repo string
		ws   string
		want string
	}{
		{"myrepo", "cool-workspace", "fr8/myrepo/cool-workspace"},
		{"fr8", "test", "fr8/fr8/test"},
		{"repo", "ws-with-dashes", "fr8/repo/ws-with-dashes"},
	}

	for _, tt := range tests {
		got := SessionName(tt.repo, tt.ws)
		if got != tt.want {
			t.Errorf("SessionName(%q, %q) = %q, want %q", tt.repo, tt.ws, got, tt.want)
		}
	}
}

func TestRepoName(t *testing.T) {
	tests := []struct {
		rootPath string
		want     string
	}{
		{"/Users/me/code/myproject", "myproject"},
		{"/a/b/c", "c"},
		{"single", "single"},
	}

	for _, tt := range tests {
		got := RepoName(tt.rootPath)
		if got != tt.want {
			t.Errorf("RepoName(%q) = %q, want %q", tt.rootPath, got, tt.want)
		}
	}
}

func TestShellescape(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"KEY=value", "KEY='value'"},
		{"KEY=hello world", "KEY='hello world'"},
		{"KEY=it's a test", "KEY='it'\"'\"'s a test'"},
		{"KEY=", "KEY=''"},
		{"NOEQUALS", "NOEQUALS"},
		{"FR8_PORT=5000", "FR8_PORT='5000'"},
		{"KEY=path/to/dir", "KEY='path/to/dir'"},
		{"KEY=val with \"quotes\"", "KEY='val with \"quotes\"'"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := shellescape(tt.input)
			if got != tt.want {
				t.Errorf("shellescape(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestAvailable(t *testing.T) {
	_, err := exec.LookPath("tmux")
	if err != nil {
		// tmux not installed — Available should return an error
		if Available() == nil {
			t.Error("Expected error when tmux is not installed")
		}
	} else {
		// tmux installed — Available should succeed
		if err := Available(); err != nil {
			t.Errorf("Expected nil error when tmux is installed, got: %v", err)
		}
	}
}

func tmuxInstalled() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

func TestIsRunningNonexistent(t *testing.T) {
	if !tmuxInstalled() {
		t.Skip("tmux not installed")
	}

	if IsRunning("fr8/nonexistent/session-that-does-not-exist-12345") {
		t.Error("IsRunning should return false for nonexistent session")
	}
}

func TestStopNonexistent(t *testing.T) {
	if !tmuxInstalled() {
		t.Skip("tmux not installed")
	}

	// Stop on a nonexistent session should be idempotent (no error)
	if err := Stop("fr8/nonexistent/session-that-does-not-exist-12345"); err != nil {
		t.Errorf("Stop on nonexistent session should return nil, got: %v", err)
	}
}

func TestStartStopLifecycle(t *testing.T) {
	if !tmuxInstalled() {
		t.Skip("tmux not installed")
	}

	name := "fr8/test-repo/test-lifecycle-ws"

	// Ensure clean state
	Stop(name)

	// Start a simple session
	err := Start(name, "/tmp", "sleep 60", nil)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer Stop(name)

	// Should be running
	if !IsRunning(name) {
		t.Error("session should be running after Start")
	}

	// Starting again should fail
	err = Start(name, "/tmp", "sleep 60", nil)
	if err == nil {
		t.Error("expected error when starting already-running session")
	}

	// Stop it
	if err := Stop(name); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Should no longer be running
	if IsRunning(name) {
		t.Error("session should not be running after Stop")
	}
}

func TestStartWithEnvVars(t *testing.T) {
	if !tmuxInstalled() {
		t.Skip("tmux not installed")
	}

	name := "fr8/test-repo/test-env-ws"

	// Ensure clean state
	Stop(name)

	envVars := []string{
		"FR8_PORT=5000",
		"FR8_WORKSPACE_NAME=test-env-ws",
	}

	err := Start(name, "/tmp", "sleep 60", envVars)
	if err != nil {
		t.Fatalf("Start with env vars failed: %v", err)
	}
	defer Stop(name)

	if !IsRunning(name) {
		t.Error("session should be running")
	}
}

func TestCapturePanes(t *testing.T) {
	if !tmuxInstalled() {
		t.Skip("tmux not installed")
	}

	name := "fr8/test-repo/test-capture-ws"

	// Ensure clean state
	Stop(name)

	// Use a long-running command so the session stays alive
	err := Start(name, "/tmp", "sleep 60", nil)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer Stop(name)

	// Give tmux a moment to initialize the session
	time.Sleep(200 * time.Millisecond)

	// Capture should not error
	_, err = CapturePanes(name, 50)
	if err != nil {
		t.Errorf("CapturePanes failed: %v", err)
	}
}

func TestCapturePanesNotRunning(t *testing.T) {
	if !tmuxInstalled() {
		t.Skip("tmux not installed")
	}

	_, err := CapturePanes("fr8/nonexistent/no-such-session", 50)
	if err == nil {
		t.Error("expected error when capturing from nonexistent session")
	}
}

func TestListFr8Sessions(t *testing.T) {
	if !tmuxInstalled() {
		t.Skip("tmux not installed")
	}

	name := "fr8/test-repo/test-list-ws"

	// Ensure clean state
	Stop(name)

	// Start a session so there's something to find
	err := Start(name, "/tmp", "sleep 60", nil)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer Stop(name)

	sessions, err := ListFr8Sessions()
	if err != nil {
		t.Fatalf("ListFr8Sessions failed: %v", err)
	}

	found := false
	for _, s := range sessions {
		if s.Name == name {
			found = true
			if s.Repo != "test-repo" {
				t.Errorf("session.Repo = %q, want test-repo", s.Repo)
			}
			if s.Workspace != "test-list-ws" {
				t.Errorf("session.Workspace = %q, want test-list-ws", s.Workspace)
			}
		}
	}
	if !found {
		t.Errorf("expected to find session %q in list, got %v", name, sessions)
	}
}

func TestListFr8SessionsFiltersNonFr8(t *testing.T) {
	if !tmuxInstalled() {
		t.Skip("tmux not installed")
	}

	// Create a non-fr8 session
	nonFr8 := "not-fr8-session-test"
	cmd := exec.Command("tmux", "new-session", "-d", "-s", nonFr8, "sleep 60")
	if err := cmd.Run(); err != nil {
		t.Skipf("could not create test session: %v", err)
	}
	defer exec.Command("tmux", "kill-session", "-t", nonFr8).Run()

	sessions, err := ListFr8Sessions()
	if err != nil {
		t.Fatalf("ListFr8Sessions failed: %v", err)
	}

	for _, s := range sessions {
		if s.Name == nonFr8 {
			t.Errorf("ListFr8Sessions should not include non-fr8 session %q", nonFr8)
		}
	}
}

func TestStartStopMultiple(t *testing.T) {
	if !tmuxInstalled() {
		t.Skip("tmux not installed")
	}

	name1 := "fr8/test-repo/test-multi-ws-1"
	name2 := "fr8/test-repo/test-multi-ws-2"

	// Ensure clean state
	Stop(name1)
	Stop(name2)

	// Start both sessions
	if err := Start(name1, "/tmp", "sleep 60", nil); err != nil {
		t.Fatalf("Start(%q) failed: %v", name1, err)
	}
	defer Stop(name1)

	if err := Start(name2, "/tmp", "sleep 60", nil); err != nil {
		t.Fatalf("Start(%q) failed: %v", name2, err)
	}
	defer Stop(name2)

	// Both should appear in ListFr8Sessions
	sessions, err := ListFr8Sessions()
	if err != nil {
		t.Fatalf("ListFr8Sessions failed: %v", err)
	}

	found1, found2 := false, false
	for _, s := range sessions {
		if s.Name == name1 {
			found1 = true
		}
		if s.Name == name2 {
			found2 = true
		}
	}
	if !found1 {
		t.Errorf("expected to find session %q in list", name1)
	}
	if !found2 {
		t.Errorf("expected to find session %q in list", name2)
	}

	// Stop both
	if err := Stop(name1); err != nil {
		t.Errorf("Stop(%q) failed: %v", name1, err)
	}
	if err := Stop(name2); err != nil {
		t.Errorf("Stop(%q) failed: %v", name2, err)
	}

	// Neither should be running
	if IsRunning(name1) {
		t.Errorf("session %q should not be running after Stop", name1)
	}
	if IsRunning(name2) {
		t.Errorf("session %q should not be running after Stop", name2)
	}
}

func TestAttachPassesEnvironment(t *testing.T) {
	if !tmuxInstalled() {
		t.Skip("tmux not installed")
	}

	name := "fr8/test-repo/test-attach-env-ws"
	Stop(name)

	if err := Start(name, "/tmp", "sleep 60", nil); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer Stop(name)

	var capturedEnv []string
	origExec := execFunc
	execFunc = func(argv0 string, argv []string, envv []string) error {
		capturedEnv = envv
		return nil
	}
	defer func() { execFunc = origExec }()

	if err := Attach(name); err != nil {
		t.Fatalf("Attach failed: %v", err)
	}

	if len(capturedEnv) == 0 {
		t.Fatal("Attach passed empty environment to exec; expected os.Environ()")
	}

	// TERM must be present for tmux to work
	found := false
	for _, e := range capturedEnv {
		if len(e) >= 5 && e[:5] == "TERM=" {
			found = true
			break
		}
	}
	if !found {
		// TERM might not be set in CI, so just check we got the real env
		if len(capturedEnv) != len(os.Environ()) {
			t.Errorf("expected environ length %d, got %d", len(os.Environ()), len(capturedEnv))
		}
	}
}

func TestAttachNotRunning(t *testing.T) {
	if !tmuxInstalled() {
		t.Skip("tmux not installed")
	}

	err := Attach("fr8/nonexistent/no-such-session")
	if err == nil {
		t.Error("expected error when attaching to nonexistent session")
	}
}
