package tui

import "github.com/charmbracelet/lipgloss"

// Adaptive color palette — works on both dark and light terminals.
// Format: AdaptiveColor{Light, Dark}
var (
	colorAccent  = lipgloss.AdaptiveColor{Light: "63", Dark: "63"}   // muted indigo
	colorSubtle  = lipgloss.AdaptiveColor{Light: "243", Dark: "241"} // gray
	colorText    = lipgloss.AdaptiveColor{Light: "235", Dark: "252"} // near-white on dark
	colorGreen   = lipgloss.AdaptiveColor{Light: "34", Dark: "78"}   // clean / merged
	colorOrange  = lipgloss.AdaptiveColor{Light: "208", Dark: "208"} // dirty
	colorRed     = lipgloss.AdaptiveColor{Light: "160", Dark: "203"} // error
	colorCyan    = lipgloss.AdaptiveColor{Light: "37", Dark: "75"}   // ports
	colorYellow  = lipgloss.AdaptiveColor{Light: "136", Dark: "220"} // confirm
	colorBorder  = lipgloss.AdaptiveColor{Light: "250", Dark: "238"} // panel borders
)

// Breadcrumb / title bar
var (
	breadcrumbSepStyle = lipgloss.NewStyle().
				Foreground(colorSubtle).
				Padding(0, 1)

	breadcrumbActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorAccent)

	breadcrumbDimStyle = lipgloss.NewStyle().
				Foreground(colorSubtle)
)

// List rows
var (
	cursorStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	selectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorText)

	normalRowStyle = lipgloss.NewStyle().
			Foreground(colorText)

	dimStyle = lipgloss.NewStyle().
			Foreground(colorSubtle)
)

// Status indicators
var (
	statusCleanStyle = lipgloss.NewStyle().
				Foreground(colorGreen)

	statusDirtyStyle = lipgloss.NewStyle().
				Foreground(colorOrange)

	statusMergedStyle = lipgloss.NewStyle().
				Foreground(colorGreen)

	statusErrorStyle = lipgloss.NewStyle().
				Foreground(colorRed)
)

// Detail pane
var (
	detailLabelStyle = lipgloss.NewStyle().
				Foreground(colorSubtle).
				Width(12)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(colorText)
)

// Help bar
var (
	helpKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(colorSubtle)

	helpSepStyle = lipgloss.NewStyle().
			Foreground(colorSubtle).
			Padding(0, 1)
)

// Status bar
var statusBarStyle = lipgloss.NewStyle().
	Foreground(colorSubtle)

// Toast notifications
var (
	toastStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	toastErrorStyle = lipgloss.NewStyle().
			Foreground(colorRed)
)

// Misc
var (
	errorStyle = lipgloss.NewStyle().
			Foreground(colorRed)

	confirmStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorYellow)

	portStyle = lipgloss.NewStyle().
			Foreground(colorCyan)

	spinnerStyle = lipgloss.NewStyle().
			Foreground(colorAccent)

	filterActiveStyle = lipgloss.NewStyle().
				Foreground(colorSubtle).
				Italic(true)
)
