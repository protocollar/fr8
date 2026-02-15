package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/protocollar/fr8/internal/gh"
	"github.com/protocollar/fr8/internal/git"
	"github.com/protocollar/fr8/internal/registry"
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

// --- 1.2 Scroll Position Indicator ---

func TestRenderTitledPanelWithPosShowsIndicator(t *testing.T) {
	result := renderTitledPanelWithPos("Items", "line1\nline2", 40, 1, 20, 5)
	if !strings.Contains(result, "(1/20)") {
		t.Error("expected position indicator (1/20) in title when total > visible")
	}
}

func TestRenderTitledPanelWithPosHidesWhenFits(t *testing.T) {
	result := renderTitledPanelWithPos("Items", "line1\nline2", 40, 1, 3, 5)
	if strings.Contains(result, "(1/3)") {
		t.Error("position indicator should not appear when total <= visible")
	}
}

// --- 1.3 Status Bar ---

func TestRenderStatusBar(t *testing.T) {
	repos := []repoItem{
		{Repo: registry.Repo{Name: "a"}, WorkspaceCount: 3, RunningCount: 1},
		{Repo: registry.Repo{Name: "b"}, WorkspaceCount: 5, RunningCount: 2},
	}

	got := renderStatusBar(repos, 80)
	if !strings.Contains(got, "2 repos") {
		t.Errorf("status bar should contain '2 repos', got %q", got)
	}
	if !strings.Contains(got, "8 workspaces") {
		t.Errorf("status bar should contain '8 workspaces', got %q", got)
	}
	if !strings.Contains(got, "3 running") {
		t.Errorf("status bar should contain '3 running', got %q", got)
	}
}

func TestRenderStatusBarEmpty(t *testing.T) {
	got := renderStatusBar(nil, 80)
	if got != "" {
		t.Errorf("expected empty status bar for nil repos, got %q", got)
	}
}

// --- 1.4 Toast Rendering ---

func TestRenderToast(t *testing.T) {
	got := renderToast("started ws-one", false, 80)
	if !strings.Contains(got, "started ws-one") {
		t.Errorf("toast should contain message, got %q", got)
	}

	got = renderToast("", false, 80)
	if got != "" {
		t.Errorf("empty toast should return empty string, got %q", got)
	}
}

// --- 3.1 Short Relative Time ---

func TestShortRelativeTime(t *testing.T) {
	tests := []struct {
		name string
		ago  time.Duration
		want string
	}{
		{"now", 5 * time.Second, "now"},
		{"minutes", 5 * time.Minute, "5m"},
		{"hours", 3 * time.Hour, "3h"},
		{"days", 48 * time.Hour, "2d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shortRelativeTime(time.Now().Add(-tt.ago))
			if got != tt.want {
				t.Errorf("shortRelativeTime() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- 3.2 Wide Layout ---

func TestIsWide(t *testing.T) {
	t.Setenv("TERMINAL_EMULATOR", "") // ensure JediTerm detection doesn't interfere
	if isWide(80) {
		t.Error("80 should not be wide")
	}
	if !isWide(120) {
		t.Error("120 should be wide")
	}
	if !isWide(160) {
		t.Error("160 should be wide")
	}
}

func TestIsWideJediTerm(t *testing.T) {
	t.Setenv("TERMINAL_EMULATOR", "JetBrains-JediTerm")
	if isWide(160) {
		t.Error("JediTerm should never be wide")
	}
}

func TestChromeHeight(t *testing.T) {
	t.Setenv("TERMINAL_EMULATOR", "") // ensure JediTerm detection doesn't interfere
	m := model{
		width: 80, // not wide
		repos: []repoItem{{Repo: registry.Repo{Name: "a"}}},
	}

	base := chromeHeight(m)

	// With toast
	m.toast = "hello"
	withToast := chromeHeight(m)
	if withToast != base+1 {
		t.Errorf("toast should add 1 to chrome height: got %d, want %d", withToast, base+1)
	}

	// With filtering
	m.toast = ""
	m.filtering = true
	withFilter := chromeHeight(m)
	if withFilter != base+1 {
		t.Errorf("filter should add 1 to chrome height: got %d, want %d", withFilter, base+1)
	}

	// Wide mode removes detail pane chrome
	m.filtering = false
	m.width = 160
	wideH := chromeHeight(m)
	if wideH >= base {
		t.Errorf("wide mode should reduce chrome height: got %d, base %d", wideH, base)
	}
}

func TestWideRendersJoined(t *testing.T) {
	t.Setenv("TERMINAL_EMULATOR", "") // ensure JediTerm detection doesn't interfere
	m := seedRepoModel()
	m.width = 160
	m.height = 40

	output := renderRepoList(m)
	// In wide mode, list and detail panels are joined — detail should still be present
	if !strings.Contains(output, "Details") {
		t.Error("wide mode should still render Details panel")
	}
}

func TestNarrowRendersStacked(t *testing.T) {
	m := seedRepoModel()
	m.width = 80
	m.height = 40

	output := renderRepoList(m)
	if !strings.Contains(output, "Details") {
		t.Error("narrow mode should render Details panel")
	}
}

// --- Resize Safety ---

func TestConstrainWidthTruncatesLongLines(t *testing.T) {
	// A line wider than maxWidth should be truncated
	wide := strings.Repeat("A", 50)
	got := constrainWidth(wide, 30)
	for i, line := range strings.Split(got, "\n") {
		w := lipgloss.Width(line)
		if w != 30 {
			t.Errorf("line %d: width = %d, want 30", i, w)
		}
	}
}

func TestConstrainWidthPadsShortLines(t *testing.T) {
	short := "hi"
	got := constrainWidth(short, 10)
	if lipgloss.Width(got) != 10 {
		t.Errorf("constrainWidth(%q, 10) width = %d, want 10", short, lipgloss.Width(got))
	}
}

func TestConstrainWidthMultiline(t *testing.T) {
	input := "short\n" + strings.Repeat("X", 50) + "\nexact"
	got := constrainWidth(input, 20)
	for i, line := range strings.Split(got, "\n") {
		w := lipgloss.Width(line)
		if w != 20 {
			t.Errorf("line %d: width = %d, want 20: %q", i, w, line)
		}
	}
}

func TestConstrainWidthPreservesExact(t *testing.T) {
	exact := strings.Repeat("B", 15)
	got := constrainWidth(exact, 15)
	if got != exact {
		t.Errorf("exact-width line should be unchanged")
	}
}

func TestWorkspaceRowContainsBranch(t *testing.T) {
	m := seedWorkspaceModel()
	m.width = 120 // wide enough for branch column
	m.height = 40

	output := renderWorkspaceList(m)
	if !strings.Contains(output, "feat-1") {
		t.Error("workspace row should contain branch name at sufficient width")
	}
}

// --- scrollWindow ---

func TestScrollWindow(t *testing.T) {
	tests := []struct {
		name      string
		cursor    int
		total     int
		height    int
		wantStart int
		wantEnd   int
	}{
		{"fits all", 0, 3, 5, 0, 3},
		{"cursor at top", 0, 20, 5, 0, 5},
		{"cursor middle", 10, 20, 5, 8, 13},
		{"cursor at end", 19, 20, 5, 15, 20},
		{"single item", 0, 1, 5, 0, 1},
		{"exact fit", 2, 5, 5, 0, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := scrollWindow(tt.cursor, tt.total, tt.height)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Errorf("scrollWindow(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tt.cursor, tt.total, tt.height, start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

// --- Selection markers in workspace rows ---

func TestWorkspaceRowSelectionMarker(t *testing.T) {
	item := workspaceItem{
		Workspace: registry.Workspace{Name: "ws-sel", Port: 3000},
		Branch:    "main",
	}
	selected := map[int]bool{0: true}

	// Selected row should show [*]
	row := renderWorkspaceRow(item, 0, 0, 0, selected, 120)
	if !strings.Contains(row, "[*]") {
		t.Errorf("selected row should contain [*], got: %q", row)
	}

	// Unselected row should show [ ]
	row = renderWorkspaceRow(item, 0, 0, 0, map[int]bool{1: true}, 120)
	if !strings.Contains(row, "[ ]") {
		t.Errorf("unselected row with active selection should contain [ ], got: %q", row)
	}

	// No selection at all — no markers
	row = renderWorkspaceRow(item, 0, 0, 0, nil, 120)
	if strings.Contains(row, "[*]") || strings.Contains(row, "[ ]") {
		t.Errorf("row with no selection should have no markers, got: %q", row)
	}
}

// --- Wide mode help bar visible ---

func TestWideRendersHelpBar(t *testing.T) {
	t.Setenv("TERMINAL_EMULATOR", "")
	m := seedRepoModel()
	m.width = 160
	m.height = 40

	output := renderRepoList(m)
	// Help bar should contain key hints like "enter", "filter", "quit"
	if !strings.Contains(output, "enter") {
		t.Error("wide mode should render help bar with 'enter' hint")
	}
	if !strings.Contains(output, "quit") {
		t.Error("wide mode should render help bar with 'quit' hint")
	}
}

func TestWideWorkspaceRendersHelpBar(t *testing.T) {
	t.Setenv("TERMINAL_EMULATOR", "")
	m := seedWorkspaceModel()
	m.width = 160
	m.height = 40

	output := renderWorkspaceList(m)
	if !strings.Contains(output, "filter") {
		t.Error("wide workspace view should render help bar with 'filter' hint")
	}
	if !strings.Contains(output, "quit") {
		t.Error("wide workspace view should render help bar with 'quit' hint")
	}
}

// --- View output consistency (padToHeight + constrainWidth order) ---

func TestViewOutputConsistentLineWidths(t *testing.T) {
	t.Setenv("TERMINAL_EMULATOR", "")
	m := seedRepoModel()
	m.width = 80
	m.height = 24

	output := m.View()
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w != 80 {
			t.Errorf("line %d: width = %d, want 80", i, w)
		}
	}
}

func TestViewOutputConsistentLineWidthsWide(t *testing.T) {
	t.Setenv("TERMINAL_EMULATOR", "")
	m := seedRepoModel()
	m.width = 160
	m.height = 40

	output := m.View()
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w != 160 {
			t.Errorf("line %d: width = %d, want 160", i, w)
		}
	}
}

// --- Toast rendering in error mode ---

func TestRenderToastError(t *testing.T) {
	got := renderToast("something failed", true, 80)
	if !strings.Contains(got, "something failed") {
		t.Errorf("error toast should contain message, got %q", got)
	}
}

// --- Empty selection map treated as no selection ---

func TestWorkspaceRowEmptySelectionMap(t *testing.T) {
	item := workspaceItem{
		Workspace: registry.Workspace{Name: "ws-test", Port: 3000},
	}
	// Empty map (not nil) — should still show markers since len > 0
	selected := map[int]bool{}
	row := renderWorkspaceRow(item, 0, 0, 0, selected, 120)
	// Empty map has len 0, so no markers should appear
	if strings.Contains(row, "[*]") || strings.Contains(row, "[ ]") {
		t.Errorf("empty selection map should have no markers, got: %q", row)
	}
}
