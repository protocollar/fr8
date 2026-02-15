package opener

import (
	"strings"
	"testing"

	"github.com/protocollar/fr8/internal/userconfig"
)

func TestRunMissingExecutable(t *testing.T) {
	o := userconfig.Opener{Name: "fake", Command: "fr8_nonexistent_binary_xyz"}
	err := Run(o, "/tmp")
	if err == nil {
		t.Fatal("expected error for missing executable")
	}
	if !strings.Contains(err.Error(), "executable not found") {
		t.Errorf("error = %q, want it to mention 'executable not found'", err.Error())
	}
}

func TestRunMultiWordCommand(t *testing.T) {
	o := userconfig.Opener{Name: "echo-test", Command: "echo --flag extra"}
	err := Run(o, "/tmp/workspace")
	if err != nil {
		t.Fatalf("Run with multi-word command: %v", err)
	}
}

func TestRunEmptyCommand(t *testing.T) {
	o := userconfig.Opener{Name: "empty", Command: ""}
	err := Run(o, "/tmp")
	if err == nil {
		t.Fatal("expected error for empty command")
	}
	if !strings.Contains(err.Error(), "empty command") {
		t.Errorf("error = %q, want it to mention 'empty command'", err.Error())
	}
}
