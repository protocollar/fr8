package tui

import (
	"strings"
	"testing"
)

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		name     string
		item     workspaceItem
		contains []string
		excludes []string
	}{
		{
			name:     "clean",
			item:     workspaceItem{},
			contains: []string{"clean"},
		},
		{
			name:     "dirty",
			item:     workspaceItem{Dirty: true},
			contains: []string{"dirty"},
			excludes: []string{"clean"},
		},
		{
			name:     "merged",
			item:     workspaceItem{Merged: true},
			contains: []string{"merged"},
			excludes: []string{"clean"},
		},
		{
			name:     "ahead only",
			item:     workspaceItem{Ahead: 3},
			contains: []string{"\u21913"}, // ↑3
			excludes: []string{"clean", "\u2193"},
		},
		{
			name:     "behind only",
			item:     workspaceItem{Behind: 2},
			contains: []string{"\u21932"}, // ↓2
			excludes: []string{"clean", "\u2191"},
		},
		{
			name:     "ahead and behind",
			item:     workspaceItem{Ahead: 1, Behind: 4},
			contains: []string{"\u21911", "\u21934"},
			excludes: []string{"clean"},
		},
		{
			name:     "dirty and merged",
			item:     workspaceItem{Dirty: true, Merged: true},
			contains: []string{"dirty", "merged"},
			excludes: []string{"clean"},
		},
		{
			name:     "error",
			item:     workspaceItem{StatusErr: errStub{}},
			contains: []string{"error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatStatus(tt.item)
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("formatStatus() = %q, want it to contain %q", got, want)
				}
			}
			for _, nope := range tt.excludes {
				if strings.Contains(got, nope) {
					t.Errorf("formatStatus() = %q, should not contain %q", got, nope)
				}
			}
		})
	}
}

type errStub struct{}

func (errStub) Error() string { return "stub error" }

func TestTruncate(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  string
	}{
		{"short", 10, "short"},
		{"exact-len!", 10, "exact-len!"},
		{"this-is-too-long", 10, "this-is-t\u2026"},
		{"", 5, ""},
		{"ab", 2, "ab"},
		{"abc", 2, "a\u2026"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := truncate(tt.input, tt.max)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
			}
		})
	}
}
