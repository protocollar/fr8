package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderRepoList(m model) string {
	var b strings.Builder
	w := m.width

	// Breadcrumb
	b.WriteString(renderBreadcrumb([]string{"fr8", "repos"}))
	b.WriteString("\n\n")

	// Status bar
	if sb := renderStatusBar(m.repos, w); sb != "" {
		b.WriteString(sb)
		b.WriteString("\n")
	}

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

	// Apply filter
	filtered := filteredRepos(m.repos, m.filterInput.Value())

	// List panel — compute available lines for the list.
	listHeight := m.height - chromeHeight(m)
	if listHeight < 3 {
		listHeight = 3
	}

	// Build list rows
	listW := w
	if isWide(w) {
		listW = w * 3 / 5 // 60% for list in wide mode
	}

	var rows []string
	start, end := scrollWindow(m.cursor, len(filtered), listHeight)
	for i := start; i < end; i++ {
		item := filtered[i]
		wsCount := fmt.Sprintf("%d", item.WorkspaceCount)
		if item.Err != nil {
			wsCount = "?"
		}

		name := item.Repo.Name
		path := shortenPath(item.Repo.Path)

		// Compute inner width minus cursor/padding: "  ▸ " = 4
		innerAvail := listW - 4 - 4 // 4 for panel border/padding, 4 for cursor prefix
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

	// Filter indicator
	filterQuery := m.filterInput.Value()
	if m.filtering {
		rows = append([]string{m.filterInput.View()}, rows...)
	} else if filterQuery != "" {
		rows = append([]string{filterActiveStyle.Render("filter: " + filterQuery)}, rows...)
	}

	listPanel := renderTitledPanelWithPos("Repos", strings.Join(rows, "\n"), listW, m.cursor+1, len(filtered), listHeight)

	// Detail pane for selected repo
	var detailPanel string
	if m.cursor < len(filtered) {
		origIdx := resolveOriginalRepoIndex(m.cursor, filtered, m.repos)
		item := m.repos[origIdx]
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

		detailW := w
		if isWide(w) {
			detailW = w - listW
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
			b.WriteString("\n")
		}
	}

	// Toast
	if t := renderToast(m.toast, m.toastIsError, w); t != "" {
		b.WriteString(t)
		b.WriteString("\n")
	}

	// Help bar
	b.WriteString(renderHelpBar([]helpItem{
		{"enter", "open"},
		{"/", "filter"},
		{"r", "run all"},
		{"x", "stop all"},
		{"R", "global run"},
		{"X", "global stop"},
		{"ctrl+r", "refresh"},
		{"?", "help"},
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
