package cmd

import (
	"strings"
	"testing"
)

func TestSetVersionInfo(t *testing.T) {
	SetVersionInfo("1.2.3", "abc1234", "2026-02-13")

	if Version != "1.2.3" {
		t.Errorf("Version = %q, want %q", Version, "1.2.3")
	}
	if Commit != "abc1234" {
		t.Errorf("Commit = %q, want %q", Commit, "abc1234")
	}
	if Date != "2026-02-13" {
		t.Errorf("Date = %q, want %q", Date, "2026-02-13")
	}

	v := rootCmd.Version
	if !strings.Contains(v, "1.2.3") {
		t.Errorf("rootCmd.Version = %q, want it to contain version", v)
	}
	if !strings.Contains(v, "abc1234") {
		t.Errorf("rootCmd.Version = %q, want it to contain commit", v)
	}
	if !strings.Contains(v, "2026-02-13") {
		t.Errorf("rootCmd.Version = %q, want it to contain date", v)
	}
}
