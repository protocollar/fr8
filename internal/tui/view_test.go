package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/protocollar/fr8/internal/gh"
	"github.com/protocollar/fr8/internal/git"
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
			name:     "dirty modified",
			item:     workspaceItem{DirtyCount: git.DirtyCount{Modified: 1}},
			contains: []string{"1~"},
			excludes: []string{"clean"},
		},
		{
			name:     "dirty staged",
			item:     workspaceItem{DirtyCount: git.DirtyCount{Staged: 2}},
			contains: []string{"2\u2191"},
			excludes: []string{"clean"},
		},
		{
			name:     "dirty untracked",
			item:     workspaceItem{DirtyCount: git.DirtyCount{Untracked: 3}},
			contains: []string{"3?"},
			excludes: []string{"clean"},
		},
		{
			name:     "dirty mixed",
			item:     workspaceItem{DirtyCount: git.DirtyCount{Staged: 2, Modified: 3, Untracked: 1}},
			contains: []string{"2\u2191", "3~", "1?"},
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
			item:     workspaceItem{DirtyCount: git.DirtyCount{Modified: 1}, Merged: true},
			contains: []string{"1~", "merged"},
			excludes: []string{"clean"},
		},
		{
			name:     "with PR",
			item:     workspaceItem{PR: &gh.PRInfo{Number: 42, State: "OPEN"}},
			contains: []string{"PR #42"},
			excludes: []string{"clean"},
		},
		{
			name:     "with draft PR",
			item:     workspaceItem{PR: &gh.PRInfo{Number: 10, State: "OPEN", IsDraft: true}},
			contains: []string{"PR #10", "draft"},
			excludes: []string{"clean"},
		},
		{
			name:     "with approved PR",
			item:     workspaceItem{PR: &gh.PRInfo{Number: 5, State: "OPEN", ReviewDecision: "APPROVED"}},
			contains: []string{"PR #5", "\u2713"},
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

func TestPadToHeight(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		height int
		want   int // expected newline count
	}{
		{
			name:   "pads short output",
			input:  "line1\nline2\n",
			height: 5,
			want:   5, // 2 existing + 3 padding
		},
		{
			name:   "no pad when already full",
			input:  "a\nb\nc\nd\ne\n",
			height: 5,
			want:   5,
		},
		{
			name:   "no pad when over height",
			input:  "a\nb\nc\nd\ne\nf\n",
			height: 5,
			want:   6, // unchanged
		},
		{
			name:   "empty string",
			input:  "",
			height: 3,
			want:   3,
		},
		{
			name:   "content without trailing newline",
			input:  "line1\nline2",
			height: 5,
			want:   4, // 2 lines (1 newline + trailing content) → pad 3 newlines = 4 total
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := padToHeight(tt.input, tt.height)
			newlines := strings.Count(got, "\n")
			if newlines != tt.want {
				t.Errorf("padToHeight() produced %d newlines, want %d\nresult: %q", newlines, tt.want, got)
			}
		})
	}
}

func TestRenderHelpBarWraps(t *testing.T) {
	items := []helpItem{
		{"r", "run"},
		{"x", "stop"},
		{"t", "attach"},
		{"s", "shell"},
		{"b", "browser"},
		{"a", "archive"},
		{"esc", "back"},
		{"q", "quit"},
	}

	// Wide terminal — should be single line
	wide := renderHelpBar(items, 200)
	if strings.Contains(wide, "\n") {
		t.Errorf("expected single line at width=200, got:\n%s", wide)
	}

	// Narrow terminal — should wrap
	narrow := renderHelpBar(items, 40)
	lines := strings.Split(narrow, "\n")
	if len(lines) < 2 {
		t.Errorf("expected multiple lines at width=40, got %d line(s):\n%s", len(lines), narrow)
	}

	// Each wrapped line should start with indent
	for i, line := range lines {
		if !strings.HasPrefix(line, "  ") {
			t.Errorf("line %d missing 2-space indent: %q", i, line)
		}
	}
}

func TestRenderTitledPanelTruncatesLongLines(t *testing.T) {
	// Create a line longer than inner width (width=30 → innerWidth=26)
	longLine := strings.Repeat("A", 50)
	result := renderTitledPanel("Test", longLine, 30)

	// Every non-empty line (top border, body rows, bottom border) must be
	// exactly the target width so corners and edges align.
	for i, line := range strings.Split(result, "\n") {
		if line == "" {
			continue
		}
		w := lipgloss.Width(line)
		if w != 30 {
			t.Errorf("line %d: width = %d, want 30: %q", i, w, line)
		}
	}
}

func TestRenderTitledPanelAlignment(t *testing.T) {
	// Normal-length content — verify all lines are the same width.
	result := renderTitledPanel("Info", "hello", 40)
	for i, line := range strings.Split(result, "\n") {
		if line == "" {
			continue
		}
		w := lipgloss.Width(line)
		if w != 40 {
			t.Errorf("line %d: width = %d, want 40: %q", i, w, line)
		}
	}
}

func TestRelativeTime(t *testing.T) {
	tests := []struct {
		name string
		ago  time.Duration
		want string
	}{
		{"just now", 5 * time.Second, "just now"},
		{"minutes", 5 * time.Minute, "5m ago"},
		{"hours", 3 * time.Hour, "3h ago"},
		{"days", 48 * time.Hour, "2d ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relativeTime(time.Now().Add(-tt.ago))
			if got != tt.want {
				t.Errorf("relativeTime() = %q, want %q", got, tt.want)
			}
		})
	}
}

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
