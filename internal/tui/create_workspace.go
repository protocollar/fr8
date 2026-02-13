package tui

import (
	"strings"
)

func renderCreateWorkspace(m model) string {
	var b strings.Builder
	w := m.width

	b.WriteString(renderBreadcrumb([]string{"fr8", m.repoName, "new workspace"}))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render(m.err.Error()))
		b.WriteString("\n\n")
	}

	b.WriteString(renderTitledPanel("New Workspace", m.createInput.View(), w))
	b.WriteString("\n\n")

	b.WriteString(renderHelpBar([]helpItem{
		{"enter", "create"},
		{"esc", "cancel"},
	}, w))
	b.WriteString("\n")

	return b.String()
}
