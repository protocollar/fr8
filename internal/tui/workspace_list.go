package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/protocollar/fr8/internal/gh"
)

func renderWorkspaceList(m model) string {
	var b strings.Builder
	w := m.width

	// Breadcrumb
	b.WriteString(renderBreadcrumb([]string{"fr8", m.repoName, "workspaces"}))
	b.WriteString("\n\n")

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

	// List panel
	listHeight := m.height - 10
	if listHeight < 3 {
		listHeight = 3
	}

	var rows []string
	start, end := scrollWindow(m.cursor, len(m.workspaces), listHeight)
	for i := start; i < end; i++ {
		item := m.workspaces[i]
		status := formatStatus(item)
		port := portStyle.Render(fmt.Sprintf(":%d", item.Workspace.Port))
		name := item.Workspace.Name

		runBadge := "  "
		if item.Running {
			runBadge = statusCleanStyle.Render("▶ ")
		}

		nameWidth := 24
		var line string
		if i == m.cursor {
			line = fmt.Sprintf("%s %s%s  %s  %s",
				cursorStyle.Render("▸"),
				runBadge,
				selectedRowStyle.Render(fmt.Sprintf("%-*s", nameWidth, name)),
				port,
				status,
			)
		} else {
			line = fmt.Sprintf("  %s%s  %s  %s",
				runBadge,
				normalRowStyle.Render(fmt.Sprintf("%-*s", nameWidth, name)),
				port,
				status,
			)
		}
		rows = append(rows, line)
	}

	b.WriteString(renderTitledPanel("Workspaces", strings.Join(rows, "\n"), w))
	b.WriteString("\n")

	// Detail pane or archive confirmation
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
		b.WriteString(renderTitledPanel("Confirm", detail.String(), w))
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
		b.WriteString(renderTitledPanel("Confirm Batch Archive", detail.String(), w))
	case m.cursor < len(m.workspaces):
		item := m.workspaces[m.cursor]
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
		b.WriteString(renderTitledPanel("Details", detail.String(), w))
	}
	b.WriteString("\n")

	// Help bar
	b.WriteString(renderHelpBar([]helpItem{
		{"n", "new"},
		{"r", "run"},
		{"x", "stop"},
		{"t", "attach"},
		{"s", "shell"},
		{"o", "open"},
		{"b", "browser"},
		{"a", "archive"},
		{"A", "archive merged"},
		{"?", "help"},
		{"esc", "back"},
		{"q", "quit"},
	}, w))
	b.WriteString("\n")

	return b.String()
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
