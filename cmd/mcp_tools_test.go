package cmd

import (
	"encoding/json"
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

func TestMcpResolveRepoRequiresParam(t *testing.T) {
	_, _, err := mcpResolveRepo("")
	if err == nil {
		t.Fatal("expected error for empty repo param")
	}
	want := "repo parameter is required"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
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
		"config_doctor",
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
