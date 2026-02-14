package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDir(t *testing.T) {
	dir, err := configDir()
	if err != nil {
		t.Fatalf("configDir: %v", err)
	}

	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "fr8")
	if dir != want {
		t.Errorf("configDir() = %q, want %q", dir, want)
	}
}
