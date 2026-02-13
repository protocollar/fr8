package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestMcpResult(t *testing.T) {
	v := map[string]string{"key": "value"}
	result, err := mcpResult(v)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Error("IsError should be false")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	tc, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}

	var got map[string]string
	if err := json.Unmarshal([]byte(tc.Text), &got); err != nil {
		t.Fatalf("invalid JSON in content: %v", err)
	}
	if got["key"] != "value" {
		t.Errorf("got %v, want key=value", got)
	}
}

func TestMcpResultMarshalError(t *testing.T) {
	_, err := mcpResult(func() {})
	if err == nil {
		t.Error("expected error when marshaling a function")
	}
}

func TestMcpError(t *testing.T) {
	result, err := mcpError("something went wrong")
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("IsError should be true")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	tc, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}
	if tc.Text != "something went wrong" {
		t.Errorf("text = %q, want %q", tc.Text, "something went wrong")
	}
}

func TestMcpResolveWorkspaceEmptyName(t *testing.T) {
	_, _, _, err := mcpResolveWorkspace("", "")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if err.Error() != "workspace name is required" {
		t.Errorf("error = %q, want %q", err.Error(), "workspace name is required")
	}
}

func TestMcpResolveWorkspaceWithRepoNoName(t *testing.T) {
	_, _, _, err := mcpResolveWorkspace("", "some-repo")
	if err == nil {
		t.Fatal("expected error for empty name even with repo")
	}
	if err.Error() != "workspace name is required" {
		t.Errorf("error = %q, want %q", err.Error(), "workspace name is required")
	}
}

func TestMcpResolveRepoFromCWD(t *testing.T) {
	dir := initTestRepo(t)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	rootPath, commonDir, err := mcpResolveRepo("")
	if err != nil {
		t.Fatalf("mcpResolveRepo from git repo: %v", err)
	}
	if rootPath == "" {
		t.Error("rootPath should not be empty")
	}
	if commonDir == "" {
		t.Error("commonDir should not be empty")
	}
}

func TestMcpResolveRepoNotGitRepo(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	_, _, err = mcpResolveRepo("")
	if err == nil {
		t.Fatal("expected error outside git repo")
	}
	want := "not inside a git repository"
	if !contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err.Error(), want)
	}
}

func TestRegisterMCPTools(t *testing.T) {
	s := server.NewMCPServer("test", "0.0.0")
	registerMCPTools(s)

	tools := s.ListTools()
	expectedTools := []string{
		"workspace_list",
		"workspace_status",
		"workspace_create",
		"workspace_archive",
		"workspace_run",
		"workspace_stop",
		"workspace_env",
		"workspace_logs",
		"workspace_rename",
		"repo_list",
		"config_show",
		"config_validate",
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("got %d tools, want %d", len(tools), len(expectedTools))
	}
	for _, name := range expectedTools {
		if _, ok := tools[name]; !ok {
			t.Errorf("missing tool %q", name)
		}
	}
}

// contains checks if s contains substr (same as exitcode.contains but local).
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// initTestRepo creates a temporary git repo for integration tests.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s", args, out)
		}
	}
	return dir
}
