package env

import (
	"strings"
	"testing"
	"time"

	"github.com/thomascarr/fr8/internal/state"
)

func TestBuildContainsAllVars(t *testing.T) {
	ws := &state.Workspace{
		Name: "test-ws",
		Path: "/tmp/ws/test-ws",
		Port: 5000,
	}

	result := Build(ws, "/Users/me/project", "main")

	expected := map[string]string{
		"FR8_WORKSPACE_NAME":       "test-ws",
		"FR8_WORKSPACE_PATH":       "/tmp/ws/test-ws",
		"FR8_ROOT_PATH":            "/Users/me/project",
		"FR8_DEFAULT_BRANCH":       "main",
		"FR8_PORT":                 "5000",
		"CONDUCTOR_WORKSPACE_NAME": "test-ws",
		"CONDUCTOR_WORKSPACE_PATH": "/tmp/ws/test-ws",
		"CONDUCTOR_ROOT_PATH":      "/Users/me/project",
		"CONDUCTOR_DEFAULT_BRANCH": "main",
		"CONDUCTOR_PORT":           "5000",
	}

	envMap := toMap(result)

	for k, want := range expected {
		got, ok := envMap[k]
		if !ok {
			t.Errorf("missing env var %s", k)
			continue
		}
		if got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
}

func TestBuildPreservesExistingEnv(t *testing.T) {
	ws := &state.Workspace{Name: "ws", Path: "/tmp/ws", Port: 5000, CreatedAt: time.Now()}
	result := Build(ws, "/root", "main")

	envMap := toMap(result)

	// PATH should be preserved from the current environment
	if _, ok := envMap["PATH"]; !ok {
		t.Error("expected PATH to be preserved from current environment")
	}
}

func TestBuildFr8OverridesConductor(t *testing.T) {
	ws := &state.Workspace{Name: "ws", Path: "/tmp/ws", Port: 5000}
	result := Build(ws, "/root", "main")

	envMap := toMap(result)

	// FR8 and CONDUCTOR should have the same values
	if envMap["FR8_PORT"] != envMap["CONDUCTOR_PORT"] {
		t.Errorf("FR8_PORT=%q != CONDUCTOR_PORT=%q", envMap["FR8_PORT"], envMap["CONDUCTOR_PORT"])
	}
}

func toMap(environ []string) map[string]string {
	m := make(map[string]string)
	for _, e := range environ {
		if i := strings.IndexByte(e, '='); i >= 0 {
			m[e[:i]] = e[i+1:]
		}
	}
	return m
}
