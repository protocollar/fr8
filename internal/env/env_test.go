package env

import (
	"strings"
	"testing"
	"time"

	"github.com/protocollar/fr8/internal/registry"
)

func TestBuildContainsAllVars(t *testing.T) {
	ws := &registry.Workspace{
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
	ws := &registry.Workspace{Name: "ws", Path: "/tmp/ws", Port: 5000, CreatedAt: time.Now()}
	result := Build(ws, "/root", "main")

	envMap := toMap(result)

	// PATH should be preserved from the current environment
	if _, ok := envMap["PATH"]; !ok {
		t.Error("expected PATH to be preserved from current environment")
	}
}

func TestBuildFr8OverridesConductor(t *testing.T) {
	ws := &registry.Workspace{Name: "ws", Path: "/tmp/ws", Port: 5000}
	result := Build(ws, "/root", "main")

	envMap := toMap(result)

	// FR8 and CONDUCTOR should have the same values
	if envMap["FR8_PORT"] != envMap["CONDUCTOR_PORT"] {
		t.Errorf("FR8_PORT=%q != CONDUCTOR_PORT=%q", envMap["FR8_PORT"], envMap["CONDUCTOR_PORT"])
	}
}

func TestBuildFr8OnlyContainsOnlyFr8Vars(t *testing.T) {
	ws := &registry.Workspace{
		Name: "test-ws",
		Path: "/tmp/ws/test-ws",
		Port: 5000,
	}

	result := BuildFr8Only(ws, "/Users/me/project", "main")

	// Should have exactly 10 vars (5 FR8 + 5 CONDUCTOR)
	if len(result) != 10 {
		t.Errorf("BuildFr8Only returned %d vars, want 10", len(result))
	}

	envMap := toMap(result)

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

func TestBuildFr8OnlyExcludesProcessEnv(t *testing.T) {
	ws := &registry.Workspace{Name: "ws", Path: "/tmp/ws", Port: 5000}
	result := BuildFr8Only(ws, "/root", "main")

	envMap := toMap(result)

	// Should NOT contain PATH or HOME from the process environment
	if _, ok := envMap["PATH"]; ok {
		t.Error("BuildFr8Only should not include PATH from process environment")
	}
	if _, ok := envMap["HOME"]; ok {
		t.Error("BuildFr8Only should not include HOME from process environment")
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
