package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/thomascarr/fr8/internal/state"
)

// DashboardResult holds the outcome of the TUI session.
type DashboardResult struct {
	ShellWorkspace  *state.Workspace
	AttachWorkspace *state.Workspace
	OpenWorkspace   *state.Workspace
	OpenerName      string
	RootPath        string
}

// RunDashboard launches the interactive TUI and returns the result.
func RunDashboard() (*DashboardResult, error) {
	m := newModel()
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("running dashboard: %w", err)
	}

	fm := finalModel.(model)
	result := &DashboardResult{}
	if fm.shellRequest != nil {
		result.ShellWorkspace = &fm.shellRequest.workspace
		result.RootPath = fm.shellRequest.rootPath
	}
	if fm.attachRequest != nil {
		result.AttachWorkspace = &fm.attachRequest.workspace
		result.RootPath = fm.attachRequest.rootPath
	}
	if fm.openRequest != nil {
		result.OpenWorkspace = &fm.openRequest.workspace
		result.OpenerName = fm.openRequest.openerName
	}
	return result, nil
}
