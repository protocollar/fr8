package jsonout

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Enabled is set to true when --json flag is active.
var Enabled bool

// Concise is set to true when --concise flag is active (modifier for --json).
var Concise bool

// msgOut is where human progress messages are written.
// When --json is active, this is io.Discard.
// When stdout is not a TTY, this is os.Stderr.
// Otherwise, this is os.Stdout.
var msgOut io.Writer = os.Stdout

// SetMsgOut sets the writer for human progress messages.
func SetMsgOut(w io.Writer) {
	msgOut = w
}

// MsgOut returns the writer for human progress messages.
// Commands should use fmt.Fprintf(jsonout.MsgOut(), ...) instead of fmt.Printf
// for any human-readable output that should be suppressed in JSON mode.
func MsgOut() io.Writer {
	return msgOut
}

// Conciser is implemented by types that can return a minimal representation.
type Conciser interface {
	Concise() any
}

// Write marshals v as JSON to stdout. If Concise is enabled and v implements
// Conciser, the concise representation is used instead.
func Write(v any) error {
	if Concise {
		if c, ok := v.(Conciser); ok {
			v = c.Concise()
		}
	}
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	_, err = fmt.Fprintln(os.Stdout, string(data))
	return err
}

// WriteError writes a structured JSON error to stderr.
func WriteError(code, msg string, exitCode int) {
	v := struct {
		Error    string `json:"error"`
		Code     string `json:"code"`
		ExitCode int    `json:"exit_code"`
	}{
		Error:    msg,
		Code:     code,
		ExitCode: exitCode,
	}
	data, _ := json.Marshal(v)
	fmt.Fprintln(os.Stderr, string(data))
}
