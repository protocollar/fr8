package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/protocollar/fr8/internal/gh"
)

func renderWorkspaceList(m model) string {
	var b strings.Builder
	w := m.width

	// Breadcrumb
	b.WriteString(renderBreadcrumb([]string{"fr8", m.repoName, "workspaces"}))
	b.WriteString("\n\n")

	// Status bar
	if sb := renderStatusBar(m.repos, w); sb != "" {
		b.WriteString(sb)
		b.WriteString("\n")
	}

	if m.loading {
		content := fmt.Sprintf("%s %s", m.spinner.View(), dimStyle.Render("Loading workspaces..."))
		b.WriteString(renderTitledPanel("Workspaces", content, w))
		b.WriteString("\n")
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	if len(m.workspaces) == 0 {
		content := dimStyle.Render("No workspaces. Create one with: fr8 workspace new")
		b.WriteString(renderTitledPanel("Workspaces", content, w))
		b.WriteString("\n\n")
		b.WriteString(renderHelpBar([]helpItem{{"esc", "back"}, {"q", "quit"}}, w))
		b.WriteString("\n")
		return b.String()
	}

	// Apply filter
	filtered := filteredWorkspaces(m.workspaces, m.filterInput.Value())

	// List panel
	listHeight := m.height - chromeHeight(m)
	if listHeight < 3 {
		listHeight = 3
	}

	listW := w
	if isWide(w) {
		listW = w * 3 / 5
	}

	var rows []string
	start, end := scrollWindow(m.cursor, len(filtered), listHeight)
	for i := start; i < end; i++ {
		item := filtered[i]
		origIdx := resolveOriginalWsIndex(i, filtered, m.workspaces)
		rows = append(rows, renderWorkspaceRow(item, i, m.cursor, origIdx, m.selected, listW))
	}

	// Filter indicator
	filterQuery := m.filterInput.Value()
	if m.filtering {
		rows = append([]string{m.filterInput.View()}, rows...)
	} else if filterQuery != "" {
		rows = append([]string{filterActiveStyle.Render("filter: " + filterQuery)}, rows...)
	}

	listPanel := renderTitledPanelWithPos("Workspaces", strings.Join(rows, "\n"), listW, m.cursor+1, len(filtered), listHeight)

	// Detail pane or confirmation
	var detailPanel string
	detailW := w
	if isWide(w) {
		detailW = w - listW
	}

	switch {
	case m.view == viewConfirmArchive && m.archiveIdx < len(m.workspaces):
		ws := m.workspaces[m.archiveIdx]
		msg := fmt.Sprintf("Archive %q?", ws.Workspace.Name)
		if ws.DirtyCount.Dirty() {
			msg += " (has uncommitted changes!)"
		}
		var detail strings.Builder
		detail.WriteString(confirmStyle.Render(msg))
		detail.WriteString("\n\n")
		detail.WriteString(
			helpKeyStyle.Render("y") + " " + helpDescStyle.Render("yes") +
				"  " +
				helpKeyStyle.Render("n") + " " + helpDescStyle.Render("no"),
		)
		detailPanel = renderTitledPanel("Confirm", detail.String(), detailW)
	case m.view == viewConfirmBatchArchive && len(m.batchArchiveNames) > 0:
		var detail strings.Builder
		detail.WriteString(confirmStyle.Render(fmt.Sprintf("Archive %d merged+clean workspaces?", len(m.batchArchiveNames))))
		detail.WriteString("\n\n")
		for _, name := range m.batchArchiveNames {
			detail.WriteString("  " + dimStyle.Render("- "+name) + "\n")
		}
		detail.WriteString("\n")
		detail.WriteString(
			helpKeyStyle.Render("y") + " " + helpDescStyle.Render("yes") +
				"  " +
				helpKeyStyle.Render("n") + " " + helpDescStyle.Render("no"),
		)
		detailPanel = renderTitledPanel("Confirm Batch Archive", detail.String(), detailW)
	case m.cursor < len(filtered):
		origIdx := resolveOriginalWsIndex(m.cursor, filtered, m.workspaces)
		item := m.workspaces[origIdx]
		var detail strings.Builder
		detail.WriteString(renderDetailRow("Branch", item.Branch))
		detail.WriteString("\n")
		detail.WriteString(renderDetailRow("Port", fmt.Sprintf(":%d", item.Workspace.Port)))
		detail.WriteString("\n")
		detail.WriteString(renderDetailRow("Path", shortenPath(item.Workspace.Path)))
		detail.WriteString("\n")
		if item.Running {
			detail.WriteString(renderDetailRow("Process", statusCleanStyle.Render("● running")))
		} else {
			detail.WriteString(renderDetailRow("Process", dimStyle.Render("not running")))
		}
		detail.WriteString("\n")
		detail.WriteString(renderDetailRow("Status", formatStatus(item)))
		if item.LastCommit != nil {
			commitStr := truncate(item.LastCommit.Subject, 40) + " " + dimStyle.Render("("+relativeTime(item.LastCommit.Time)+")")
			detail.WriteString("\n")
			detail.WriteString(renderDetailRow("Last Commit", commitStr))
		}
		if item.DefaultAhead > 0 || item.DefaultBehind > 0 {
			divStr := fmt.Sprintf("+%d / -%d from %s", item.DefaultAhead, item.DefaultBehind, m.defaultBranch)
			detail.WriteString("\n")
			detail.WriteString(renderDetailRow("Divergence", dimStyle.Render(divStr)))
		}
		if item.PR != nil {
			detail.WriteString("\n")
			detail.WriteString(renderDetailRow("PR", formatPR(item.PR)))
		}
		detailPanel = renderTitledPanel("Details", detail.String(), detailW)
	}

	if isWide(w) && detailPanel != "" {
		lp := constrainWidth(listPanel, listW)
		dp := constrainWidth(detailPanel, w-listW)
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, lp, dp))
		b.WriteString("\n")
	} else {
		b.WriteString(listPanel)
		b.WriteString("\n")
		if detailPanel != "" {
			b.WriteString(detailPanel)
		}
		b.WriteString("\n")
	}

	// Toast
	if t := renderToast(m.toast, m.toastIsError, w); t != "" {
		b.WriteString(t)
		b.WriteString("\n")
	}

	// Help bar
	helpItems := []helpItem{
		{"n", "new"},
		{"/", "filter"},
		{"space", "select"},
		{"r", "run"},
		{"x", "stop"},
		{"t", "attach"},
		{"s", "shell"},
		{"o", "open"},
		{"b", "browser"},
		{"a", "archive"},
		{"A", "archive merged"},
		{"ctrl+r", "refresh"},
		{"?", "help"},
		{"esc", "back"},
		{"q", "quit"},
	}
	b.WriteString(renderHelpBar(helpItems, w))
	b.WriteString("\n")

	return b.String()
}

// renderWorkspaceRow renders a single workspace row with optional selection marker,
// branch name, and compact time.
func renderWorkspaceRow(item workspaceItem, displayIdx, cursor, origIdx int, selected map[int]bool, width int) string {
	status := formatStatus(item)
	port := portStyle.Render(fmt.Sprintf(":%d", item.Workspace.Port))
	name := item.Workspace.Name

	runBadge := "  "
	if item.Running {
		runBadge = statusCleanStyle.Render("▶ ")
	}

	// Selection marker
	selPrefix := ""
	if len(selected) > 0 {
		if selected[origIdx] {
			selPrefix = cursorStyle.Render("[*]") + " "
		} else {
			selPrefix = dimStyle.Render("[ ]") + " "
		}
	}

	// Dynamic column widths based on available width
	nameWidth := 16
	branchWidth := 0
	timeWidth := 0
	innerAvail := width - 4 - 4 // panel borders + cursor prefix
	if innerAvail > 60 {
		branchWidth = 16
		if innerAvail > 80 {
			branchWidth = 20
			nameWidth = 20
		}
		timeWidth = 4
	}

	// Branch (truncated, dim)
	branchStr := ""
	if branchWidth > 0 && item.Branch != "" {
		br := truncate(item.Branch, branchWidth)
		branchStr = "  " + dimStyle.Render(fmt.Sprintf("%-*s", branchWidth, br))
	}

	// Compact relative time
	timeStr := ""
	if timeWidth > 0 && item.LastCommit != nil {
		timeStr = "  " + dimStyle.Render(shortRelativeTime(item.LastCommit.Time))
	}

	var line string
	if displayIdx == cursor {
		line = fmt.Sprintf("%s %s%s%s%s  %s  %s  %s",
			cursorStyle.Render("▸"),
			selPrefix,
			runBadge,
			selectedRowStyle.Render(fmt.Sprintf("%-*s", nameWidth, name)),
			branchStr,
			port,
			timeStr,
			status,
		)
	} else {
		line = fmt.Sprintf("  %s%s%s%s  %s  %s  %s",
			selPrefix,
			runBadge,
			normalRowStyle.Render(fmt.Sprintf("%-*s", nameWidth, name)),
			branchStr,
			port,
			timeStr,
			status,
		)
	}
	return line
}

func formatStatus(item workspaceItem) string {
	if item.StatusErr != nil {
		return statusErrorStyle.Render("? error")
	}

	var parts []string

	if item.DirtyCount.Dirty() {
		var counts []string
		if item.DirtyCount.Staged > 0 {
			counts = append(counts, fmt.Sprintf("%d↑", item.DirtyCount.Staged))
		}
		if item.DirtyCount.Modified > 0 {
			counts = append(counts, fmt.Sprintf("%d~", item.DirtyCount.Modified))
		}
		if item.DirtyCount.Untracked > 0 {
			counts = append(counts, fmt.Sprintf("%d?", item.DirtyCount.Untracked))
		}
		parts = append(parts, statusDirtyStyle.Render("● "+strings.Join(counts, " ")))
	}
	if item.Merged {
		parts = append(parts, statusMergedStyle.Render("✓ merged"))
	}
	if item.Ahead > 0 || item.Behind > 0 {
		ab := ""
		if item.Ahead > 0 {
			ab += fmt.Sprintf("↑%d", item.Ahead)
		}
		if item.Behind > 0 {
			if ab != "" {
				ab += " "
			}
			ab += fmt.Sprintf("↓%d", item.Behind)
		}
		parts = append(parts, dimStyle.Render(ab))
	}
	if item.PR != nil {
		parts = append(parts, formatPR(item.PR))
	}

	if len(parts) == 0 {
		return statusCleanStyle.Render("● clean")
	}
	return strings.Join(parts, " ")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

// relativeTime returns a human-readable relative time string.
func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// shortRelativeTime returns a compact relative time string (e.g. "3h", "2d").
func shortRelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// formatPR renders a PR badge with appropriate styling.
func formatPR(pr *gh.PRInfo) string {
	badge := fmt.Sprintf("PR #%d", pr.Number)
	if pr.IsDraft {
		badge += " draft"
	}
	switch pr.ReviewDecision {
	case "APPROVED":
		badge += " ✓"
	case "CHANGES_REQUESTED":
		badge += " ✗"
	}
	return statusMergedStyle.Render(badge)
}
