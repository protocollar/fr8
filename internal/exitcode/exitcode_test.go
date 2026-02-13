package exitcode

import (
	"errors"
	"fmt"
	"testing"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode string
		wantExit int
	}{
		{
			name:     "workspace not found",
			err:      fmt.Errorf("workspace %q not found", "my-ws"),
			wantCode: "workspace_not_found",
			wantExit: NotFound,
		},
		{
			name:     "repo not found",
			err:      fmt.Errorf("repo %q not found in registry", "my-repo"),
			wantCode: "repo_not_found",
			wantExit: NotFound,
		},
		{
			name:     "opener not found",
			err:      fmt.Errorf("opener %q not found", "vscode"),
			wantCode: "opener_not_found",
			wantExit: NotFound,
		},
		{
			name:     "generic not found",
			err:      fmt.Errorf("session not found"),
			wantCode: "not_found",
			wantExit: NotFound,
		},
		{
			name:     "already exists",
			err:      fmt.Errorf("workspace %q already exists", "my-ws"),
			wantCode: "already_exists",
			wantExit: AlreadyExists,
		},
		{
			name:     "not in repo",
			err:      fmt.Errorf("not inside a git repository"),
			wantCode: "not_in_repo",
			wantExit: NotInRepo,
		},
		{
			name:     "dirty workspace",
			err:      fmt.Errorf("workspace has uncommitted changes"),
			wantCode: "dirty_workspace",
			wantExit: DirtyWorkspace,
		},
		{
			name:     "interactive only",
			err:      fmt.Errorf("dashboard cannot be used with --json"),
			wantCode: "interactive_only",
			wantExit: InteractiveOnly,
		},
		{
			name:     "tmux not available",
			err:      fmt.Errorf("tmux is not installed"),
			wantCode: "tmux_not_available",
			wantExit: TmuxUnavailable,
		},
		{
			name:     "default general error",
			err:      fmt.Errorf("something unexpected happened"),
			wantCode: "error",
			wantExit: GeneralError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, exit := ClassifyError(tt.err)
			if code != tt.wantCode {
				t.Errorf("code = %q, want %q", code, tt.wantCode)
			}
			if exit != tt.wantExit {
				t.Errorf("exit = %d, want %d", exit, tt.wantExit)
			}
		})
	}
}

func TestNew(t *testing.T) {
	e := New("test_code", 42, "test message")
	if e.Code != "test_code" {
		t.Errorf("Code = %q, want %q", e.Code, "test_code")
	}
	if e.ExitCode != 42 {
		t.Errorf("ExitCode = %d, want %d", e.ExitCode, 42)
	}
	if e.Error() != "test message" {
		t.Errorf("Error() = %q, want %q", e.Error(), "test message")
	}
}

func TestWrap(t *testing.T) {
	inner := fmt.Errorf("inner error")
	e := Wrap("wrap_code", 5, inner)
	if e.Code != "wrap_code" {
		t.Errorf("Code = %q, want %q", e.Code, "wrap_code")
	}
	if e.ExitCode != 5 {
		t.Errorf("ExitCode = %d, want %d", e.ExitCode, 5)
	}
	if e.Error() != "inner error" {
		t.Errorf("Error() = %q, want %q", e.Error(), "inner error")
	}
}

func TestUnwrap(t *testing.T) {
	inner := fmt.Errorf("inner error")
	e := Wrap("code", 1, inner)
	if e.Unwrap() != inner {
		t.Error("Unwrap() did not return the inner error")
	}
}

func TestErrorsAs(t *testing.T) {
	inner := fmt.Errorf("workspace %q not found", "test")
	e := Wrap("workspace_not_found", NotFound, inner)

	// Wrap in another error to test errors.As traversal
	wrapped := fmt.Errorf("resolving: %w", e)

	var exitErr *ExitError
	if !errors.As(wrapped, &exitErr) {
		t.Fatal("errors.As did not find ExitError")
	}
	if exitErr.Code != "workspace_not_found" {
		t.Errorf("Code = %q, want %q", exitErr.Code, "workspace_not_found")
	}
}
