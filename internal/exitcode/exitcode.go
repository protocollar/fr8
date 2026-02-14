package exitcode

import "fmt"

const (
	Success         = 0
	GeneralError    = 1
	NotFound        = 2
	AlreadyExists   = 3
	NotInRepo       = 4
	DirtyWorkspace  = 5
	InteractiveOnly = 6
	TmuxUnavailable = 7
	ConfigError     = 8
)

// ExitError wraps an error with a semantic exit code and machine-readable code string.
type ExitError struct {
	Err      error
	ExitCode int
	Code     string
}

func (e *ExitError) Error() string {
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

// New creates an ExitError with the given code, exit code, and message.
func New(code string, exitCode int, msg string) *ExitError {
	return &ExitError{
		Err:      fmt.Errorf("%s", msg),
		ExitCode: exitCode,
		Code:     code,
	}
}

// Wrap creates an ExitError wrapping an existing error.
func Wrap(code string, exitCode int, err error) *ExitError {
	return &ExitError{
		Err:      err,
		ExitCode: exitCode,
		Code:     code,
	}
}

// ClassifyError returns (code, exitCode) by matching common error patterns.
func ClassifyError(err error) (string, int) {
	msg := err.Error()

	switch {
	case contains(msg, "not found"):
		if contains(msg, "workspace") {
			return "workspace_not_found", NotFound
		}
		if contains(msg, "repo") {
			return "repo_not_found", NotFound
		}
		if contains(msg, "opener") {
			return "opener_not_found", NotFound
		}
		return "not_found", NotFound
	case contains(msg, "already exists"):
		return "already_exists", AlreadyExists
	case contains(msg, "not inside a git repository"):
		return "not_in_repo", NotInRepo
	case contains(msg, "tmux is not installed"):
		return "tmux_not_available", TmuxUnavailable
	case contains(msg, "cannot be used with --json"):
		return "interactive_only", InteractiveOnly
	case contains(msg, "uncommitted changes"):
		return "dirty_workspace", DirtyWorkspace
	default:
		return "error", GeneralError
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
