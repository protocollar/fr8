package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderTitledPanel renders content inside a rounded-border box with an
// optional inline title embedded in the top border.
//
//	╭─ Title ──────────────────────╮
//	│  content here                │
//	╰──────────────────────────────╯
func renderTitledPanel(title, content string, width int) string {
	// Account for left/right border (1 char each) and inner padding (1 char each).
	innerWidth := width - 4
	if innerWidth < 10 {
		innerWidth = 10
	}

	borderFg := lipgloss.NewStyle().Foreground(colorBorder)
	titleRendered := breadcrumbActiveStyle.Render(title)

	// Build custom top border: ╭─ Title ─...─╮
	var top strings.Builder
	top.WriteString(borderFg.Render("╭─ "))
	top.WriteString(titleRendered)
	top.WriteString(borderFg.Render(" "))

	// Calculate remaining dashes. lipgloss.Width accounts for ANSI.
	used := 3 + lipgloss.Width(titleRendered) + 1 // "╭─ " (3) + title + " " (1)
	remaining := width - used - 1                  // -1 for "╮"
	if remaining < 0 {
		remaining = 0
	}
	top.WriteString(borderFg.Render(strings.Repeat("─", remaining) + "╮"))

	// Content with side borders and padding.
	lines := strings.Split(content, "\n")
	var body strings.Builder
	for _, line := range lines {
		lineWidth := lipgloss.Width(line)
		if lineWidth > innerWidth {
			line = lipgloss.NewStyle().MaxWidth(innerWidth).Render(line)
			lineWidth = lipgloss.Width(line)
		}
		pad := innerWidth - lineWidth
		if pad < 0 {
			pad = 0
		}
		body.WriteString(borderFg.Render("│"))
		body.WriteString(" ")
		body.WriteString(line)
		body.WriteString(strings.Repeat(" ", pad))
		body.WriteString(" ")
		body.WriteString(borderFg.Render("│"))
		body.WriteString("\n")
	}

	// Bottom border.
	bottom := borderFg.Render("╰" + strings.Repeat("─", width-2) + "╯")

	return top.String() + "\n" + body.String() + bottom
}

type helpItem struct {
	key  string
	desc string
}

// renderHelpBar renders a styled help bar that wraps onto multiple lines when
// items would exceed width.
func renderHelpBar(items []helpItem, width int) string {
	sep := helpSepStyle.Render("·")
	sepWidth := lipgloss.Width(sep) + 2 // " · "
	indent := "  "
	indentWidth := 2

	var lines []string
	var cur strings.Builder
	cur.WriteString(indent)
	curWidth := indentWidth

	for i, item := range items {
		rendered := helpKeyStyle.Render(item.key) + " " + helpDescStyle.Render(item.desc)
		itemWidth := lipgloss.Width(rendered)

		extra := 0
		if i > 0 && curWidth > indentWidth {
			extra = sepWidth
		}

		if curWidth+extra+itemWidth > width && curWidth > indentWidth {
			lines = append(lines, cur.String())
			cur.Reset()
			cur.WriteString(indent)
			curWidth = indentWidth
			extra = 0
		}

		if curWidth > indentWidth {
			cur.WriteString(" " + sep + " ")
			curWidth += sepWidth
		}
		cur.WriteString(rendered)
		curWidth += itemWidth
	}
	if curWidth > indentWidth {
		lines = append(lines, cur.String())
	}

	return strings.Join(lines, "\n")
}

// padToHeight appends empty lines so the output fills exactly targetHeight lines.
func padToHeight(s string, targetHeight int) string {
	lines := strings.Count(s, "\n")
	// Account for content after the last newline.
	if len(s) > 0 && s[len(s)-1] != '\n' {
		lines++
	}
	if lines >= targetHeight {
		return s
	}
	return s + strings.Repeat("\n", targetHeight-lines)
}

// renderBreadcrumb renders "seg > seg > active" with the last segment bold.
func renderBreadcrumb(segments []string) string {
	if len(segments) == 0 {
		return ""
	}
	sep := breadcrumbSepStyle.Render(">")
	var parts []string
	for i, seg := range segments {
		if i == len(segments)-1 {
			parts = append(parts, breadcrumbActiveStyle.Render(seg))
		} else {
			parts = append(parts, breadcrumbDimStyle.Render(seg))
		}
	}
	return strings.Join(parts, sep)
}

// shortenPath replaces the home directory prefix with ~.
func shortenPath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

// renderDetailRow renders a single label: value row for the detail pane.
func renderDetailRow(label, value string) string {
	return fmt.Sprintf("%s%s", detailLabelStyle.Render(label), detailValueStyle.Render(value))
}
