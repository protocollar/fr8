package tui

import (
	"fmt"
	"strings"
)

func renderOpenerPicker(m model) string {
	var b strings.Builder
	w := m.width

	wsName := ""
	if m.openerWsIdx < len(m.workspaces) {
		wsName = m.workspaces[m.openerWsIdx].Workspace.Name
	}

	b.WriteString(renderBreadcrumb([]string{"fr8", m.repoName, wsName, "open with"}))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	listHeight := m.height - 8 // breadcrumb(2) + help(2) + margins
	if listHeight < 3 {
		listHeight = 3
	}

	var rows []string
	start, end := scrollWindow(m.openerCursor, len(m.openers), listHeight)
	for i := start; i < end; i++ {
		o := m.openers[i]
		name := o.Name
		suffix := ""
		if o.Command != o.Name {
			suffix = "  " + dimStyle.Render(o.Command)
		}
		if i == m.openerCursor {
			rows = append(rows, fmt.Sprintf("%s %s%s",
				cursorStyle.Render("▸"),
				selectedRowStyle.Render(fmt.Sprintf("%-16s", name)),
				suffix,
			))
		} else {
			rows = append(rows, fmt.Sprintf("  %s%s",
				normalRowStyle.Render(fmt.Sprintf("%-16s", name)),
				suffix,
			))
		}
	}

	b.WriteString(renderTitledPanelWithPos("Open With", strings.Join(rows, "\n"), w, m.openerCursor+1, len(m.openers), listHeight))
	b.WriteString("\n\n")

	b.WriteString(renderHelpBar([]helpItem{
		{"enter", "select"},
		{"esc", "back"},
		{"q", "quit"},
	}, w))
	b.WriteString("\n")

	return b.String()
}
