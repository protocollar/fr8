package tui

import (
	"fmt"
	"strings"
)

func renderRepoList(m model) string {
	var b strings.Builder
	w := m.width

	// Breadcrumb
	b.WriteString(renderBreadcrumb([]string{"fr8", "repos"}))
	b.WriteString("\n\n")

	if m.loading {
		content := fmt.Sprintf("%s %s", m.spinner.View(), dimStyle.Render("Loading repos..."))
		b.WriteString(renderTitledPanel("Repos", content, w))
		b.WriteString("\n")
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
		return b.String()
	}

	if len(m.repos) == 0 {
		content := dimStyle.Render("No repos registered. Add one with: fr8 repo add")
		b.WriteString(renderTitledPanel("Repos", content, w))
		b.WriteString("\n\n")
		b.WriteString(renderHelpBar([]helpItem{{"q", "quit"}}, w))
		b.WriteString("\n")
		return b.String()
	}

	// List panel — compute available lines for the list.
	// Chrome: breadcrumb(2) + detail(6) + help(2) = 10 lines of fixed chrome.
	listHeight := m.height - 10
	if listHeight < 3 {
		listHeight = 3
	}

	// Build list rows
	var rows []string
	start, end := scrollWindow(m.cursor, len(m.repos), listHeight)
	for i := start; i < end; i++ {
		item := m.repos[i]
		wsCount := fmt.Sprintf("%d", item.WorkspaceCount)
		if item.Err != nil {
			wsCount = "?"
		}

		name := item.Repo.Name
		path := shortenPath(item.Repo.Path)

		// Compute inner width minus cursor/padding: "  ▸ " = 4
		innerAvail := w - 4 - 4 // 4 for panel border/padding, 4 for cursor prefix
		if innerAvail < 20 {
			innerAvail = 20
		}

		// Layout: name (dynamic), running badge, ws count, path
		nameWidth := 20
		if nameWidth > innerAvail/2 {
			nameWidth = innerAvail / 2
		}

		var runBadge string
		if item.RunningCount > 0 {
			runBadge = statusCleanStyle.Render(fmt.Sprintf("▶ %d/%d", item.RunningCount, item.WorkspaceCount)) + "   "
		}

		countStr := fmt.Sprintf("%s workspaces", wsCount)
		pathStr := dimStyle.Render(path)

		var line string
		if i == m.cursor {
			line = fmt.Sprintf("%s %s   %s%s   %s",
				cursorStyle.Render("▸"),
				selectedRowStyle.Render(fmt.Sprintf("%-*s", nameWidth, name)),
				runBadge,
				dimStyle.Render(countStr),
				pathStr,
			)
		} else {
			line = fmt.Sprintf("  %s   %s%s   %s",
				normalRowStyle.Render(fmt.Sprintf("%-*s", nameWidth, name)),
				runBadge,
				dimStyle.Render(countStr),
				pathStr,
			)
		}
		rows = append(rows, line)
	}

	b.WriteString(renderTitledPanel("Repos", strings.Join(rows, "\n"), w))
	b.WriteString("\n")

	// Detail pane for selected repo
	if m.cursor < len(m.repos) {
		item := m.repos[m.cursor]
		var detail strings.Builder
		detail.WriteString(renderDetailRow("Name", item.Repo.Name))
		detail.WriteString("\n")
		detail.WriteString(renderDetailRow("Path", shortenPath(item.Repo.Path)))
		detail.WriteString("\n")
		wsLabel := fmt.Sprintf("%d", item.WorkspaceCount)
		if item.Err != nil {
			wsLabel = errorStyle.Render("error loading")
		}
		detail.WriteString(renderDetailRow("Workspaces", wsLabel))
		detail.WriteString("\n")
		if item.RunningCount > 0 {
			detail.WriteString(renderDetailRow("Running", statusCleanStyle.Render(fmt.Sprintf("▶ %d of %d", item.RunningCount, item.WorkspaceCount))))
		} else {
			detail.WriteString(renderDetailRow("Running", dimStyle.Render("none")))
		}

		b.WriteString(renderTitledPanel("Details", detail.String(), w))
		b.WriteString("\n")
	}

	// Help bar
	b.WriteString(renderHelpBar([]helpItem{
		{"enter", "open"},
		{"r", "run all"},
		{"x", "stop all"},
		{"R", "global run"},
		{"X", "global stop"},
		{"q", "quit"},
	}, w))
	b.WriteString("\n")

	return b.String()
}

// scrollWindow computes the visible range [start, end) for a list given the
// cursor position and viewport height.
func scrollWindow(cursor, total, height int) (int, int) {
	if total <= height {
		return 0, total
	}
	start := cursor - height/2
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > total {
		end = total
		start = end - height
	}
	return start, end
}
