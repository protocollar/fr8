package jsonout

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"
)

func TestMsgOutDefault(t *testing.T) {
	if MsgOut() != os.Stdout {
		t.Error("default MsgOut should be os.Stdout")
	}
}

func TestSetMsgOut(t *testing.T) {
	orig := msgOut
	defer func() { msgOut = orig }()

	var buf bytes.Buffer
	SetMsgOut(&buf)
	if MsgOut() != &buf {
		t.Error("MsgOut should return the writer set by SetMsgOut")
	}
}

// conciseItem is a test type implementing the Conciser interface.
type conciseItem struct {
	Name  string `json:"name"`
	Extra string `json:"extra"`
}

func (c conciseItem) Concise() any {
	return struct {
		Name string `json:"name"`
	}{Name: c.Name}
}

// captureStdout runs fn while capturing os.Stdout and returns the output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	fn()
	_ = w.Close()

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// captureStderr runs fn while capturing os.Stderr and returns the output.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = origStderr }()

	fn()
	_ = w.Close()

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestWrite(t *testing.T) {
	origEnabled := Enabled
	origConcise := Concise
	defer func() { Enabled = origEnabled; Concise = origConcise }()
	Enabled = true
	Concise = false

	v := map[string]string{"hello": "world"}
	out := captureStdout(t, func() {
		if err := Write(v); err != nil {
			t.Fatal(err)
		}
	})

	var got map[string]string
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, out)
	}
	if got["hello"] != "world" {
		t.Errorf("got %v, want hello=world", got)
	}
}

func TestWriteConcise(t *testing.T) {
	origEnabled := Enabled
	origConcise := Concise
	defer func() { Enabled = origEnabled; Concise = origConcise }()
	Enabled = true
	Concise = true

	v := conciseItem{Name: "test", Extra: "ignored"}
	out := captureStdout(t, func() {
		if err := Write(v); err != nil {
			t.Fatal(err)
		}
	})

	var got map[string]string
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}
	if got["name"] != "test" {
		t.Errorf("name = %q, want %q", got["name"], "test")
	}
	if _, has := got["extra"]; has {
		t.Error("concise output should not contain 'extra' field")
	}
}

func TestWriteConciseNonConciser(t *testing.T) {
	origConcise := Concise
	defer func() { Concise = origConcise }()
	Concise = true

	v := map[string]string{"full": "output"}
	out := captureStdout(t, func() {
		if err := Write(v); err != nil {
			t.Fatal(err)
		}
	})

	var got map[string]string
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}
	if got["full"] != "output" {
		t.Errorf("non-Conciser value should pass through unchanged")
	}
}

func TestWriteConciseDisabled(t *testing.T) {
	origConcise := Concise
	defer func() { Concise = origConcise }()
	Concise = false

	v := conciseItem{Name: "test", Extra: "kept"}
	out := captureStdout(t, func() {
		if err := Write(v); err != nil {
			t.Fatal(err)
		}
	})

	var got map[string]string
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}
	if got["extra"] != "kept" {
		t.Error("with Concise=false, full output should include 'extra'")
	}
}

func TestWriteError(t *testing.T) {
	out := captureStderr(t, func() {
		WriteError("test_code", "something broke", 42)
	})

	var got struct {
		Error    string `json:"error"`
		Code     string `json:"code"`
		ExitCode int    `json:"exit_code"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}
	if got.Error != "something broke" {
		t.Errorf("error = %q, want %q", got.Error, "something broke")
	}
	if got.Code != "test_code" {
		t.Errorf("code = %q, want %q", got.Code, "test_code")
	}
	if got.ExitCode != 42 {
		t.Errorf("exit_code = %d, want %d", got.ExitCode, 42)
	}
}

func TestWriteMarshalError(t *testing.T) {
	// Functions cannot be marshaled to JSON
	err := Write(func() {})
	if err == nil {
		t.Error("expected error when marshaling a function")
	}
}
