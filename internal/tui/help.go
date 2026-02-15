package tui

import "strings"

func renderHelp(m model) string {
	var b strings.Builder
	w := m.width

	b.WriteString(renderBreadcrumb([]string{"fr8", "help"}))
	b.WriteString("\n\n")

	var sections strings.Builder

	sections.WriteString(breadcrumbActiveStyle.Render("Navigation"))
	sections.WriteString("\n")
	sections.WriteString(formatHelpLine("j/↓", "Move down"))
	sections.WriteString(formatHelpLine("k/↑", "Move up"))
	sections.WriteString(formatHelpLine("enter", "Select / drill down"))
	sections.WriteString(formatHelpLine("esc", "Back / cancel / clear selection"))
	sections.WriteString(formatHelpLine("/", "Filter list"))
	sections.WriteString(formatHelpLine("ctrl+r", "Refresh data"))
	sections.WriteString(formatHelpLine("ctrl+l", "Redraw screen"))
	sections.WriteString(formatHelpLine("?", "Toggle this help"))
	sections.WriteString(formatHelpLine("q", "Quit"))

	sections.WriteString("\n")
	sections.WriteString(breadcrumbActiveStyle.Render("Repo List"))
	sections.WriteString("\n")
	sections.WriteString(formatHelpLine("enter", "View workspaces"))
	sections.WriteString(formatHelpLine("r", "Run all workspaces in repo"))
	sections.WriteString(formatHelpLine("x", "Stop all workspaces in repo"))
	sections.WriteString(formatHelpLine("R", "Run all workspaces globally"))
	sections.WriteString(formatHelpLine("X", "Stop all workspaces globally"))

	sections.WriteString("\n")
	sections.WriteString(breadcrumbActiveStyle.Render("Workspace List"))
	sections.WriteString("\n")
	sections.WriteString(formatHelpLine("n", "Create new workspace"))
	sections.WriteString(formatHelpLine("space", "Toggle selection for bulk operations"))
	sections.WriteString(formatHelpLine("r", "Run dev server (or run all selected)"))
	sections.WriteString(formatHelpLine("x", "Stop dev server (or stop all selected)"))
	sections.WriteString(formatHelpLine("t", "Attach to running session"))
	sections.WriteString(formatHelpLine("s", "Open shell"))
	sections.WriteString(formatHelpLine("o", "Open with configured opener"))
	sections.WriteString(formatHelpLine("b", "Open in browser"))
	sections.WriteString(formatHelpLine("a", "Archive workspace"))
	sections.WriteString(formatHelpLine("A", "Archive all merged+clean"))

	b.WriteString(renderTitledPanel("Keybindings", sections.String(), w))
	b.WriteString("\n\n")
	b.WriteString(renderHelpBar([]helpItem{{"?", "close"}, {"q", "quit"}}, w))
	b.WriteString("\n")

	return b.String()
}

func formatHelpLine(key, desc string) string {
	return "  " + helpKeyStyle.Render(key) + "  " + dimStyle.Render(desc) + "\n"
}
