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
	used := 4 + lipgloss.Width(titleRendered) + 1 // "╭─ " + title + " "
	remaining := width - used - 1                  // -1 for "╮"
	if remaining < 0 {
		remaining = 0
	}
	top.WriteString(borderFg.Render(strings.Repeat("─", remaining) + "╮"))

	// Content with side borders and padding.
	lines := strings.Split(content, "\n")
	var body strings.Builder
	for _, line := range lines {
		// Pad or truncate line to fill inner width.
		lineWidth := lipgloss.Width(line)
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

// renderHelpBar renders a styled help bar: "key desc  key desc  ..."
func renderHelpBar(items []helpItem) string {
	var parts []string
	for _, item := range items {
		parts = append(parts,
			helpKeyStyle.Render(item.key)+" "+helpDescStyle.Render(item.desc),
		)
	}
	sep := helpSepStyle.Render("·")
	return "  " + strings.Join(parts, " "+sep+" ")
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
