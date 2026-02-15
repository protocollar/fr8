package tui

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/protocollar/fr8/internal/config"
	"github.com/protocollar/fr8/internal/env"
	"github.com/protocollar/fr8/internal/gh"
	"github.com/protocollar/fr8/internal/git"
	"github.com/protocollar/fr8/internal/port"
	"github.com/protocollar/fr8/internal/registry"
	"github.com/protocollar/fr8/internal/tmux"
	"github.com/protocollar/fr8/internal/userconfig"
)

type model struct {
	view              viewState
	previousView      viewState
	repos             []repoItem
	workspaces        []workspaceItem
	cursor            int
	repoCursor        int // remembered cursor position on repo list
	loading           bool
	err               error
	repoName          string // current repo being viewed
	rootPath          string // root worktree path for current repo
	defaultBranch     string // default branch for current repo
	shellRequest      *shellRequestMsg
	attachRequest     *attachRequestMsg
	openRequest       *openRequestMsg
	createRequest     *createRequestMsg
	archiveIdx        int // workspace index pending archive confirmation
	batchArchiveNames []string
	openers           []userconfig.Opener
	openerCursor      int
	openerWsIdx       int // workspace index for which opener picker was opened
	createInput       textinput.Model
	width             int
	height            int
	spinner           spinner.Model

	// Toast notifications
	toast        string
	toastExpiry  time.Time
	toastIsError bool

	// Search/filter
	filtering   bool
	filterInput textinput.Model

	// Multi-select
	selected map[int]bool
}

func newModel() model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = spinnerStyle
	return model{
		view:    viewRepoList,
		loading: true,
		width:   80,
		height:  24,
		spinner: s,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(loadReposCmd, m.spinner.Tick, autoRefreshTickCmd())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		debugLog("WindowSizeMsg: width=%d→%d height=%d→%d", m.width, msg.Width, m.height, msg.Height)
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case reposLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.repos = msg.repos
		m.cursor = 0
		return m, nil

	case workspacesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.view = viewRepoList
			return m, nil
		}
		m.workspaces = msg.workspaces
		m.repoName = msg.repoName
		m.rootPath = msg.rootPath
		m.defaultBranch = msg.defaultBranch
		if m.view == viewWorkspaceList {
			// Refresh: clamp cursor instead of resetting
			if m.cursor >= len(m.workspaces) && m.cursor > 0 {
				m.cursor = len(m.workspaces) - 1
			}
		} else {
			m.cursor = 0
		}
		m.view = viewWorkspaceList
		return m, nil

	case archiveResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.toast = fmt.Sprintf("error archiving %s", msg.name)
			m.toastIsError = true
			m.toastExpiry = time.Now().Add(3 * time.Second)
			m.view = viewWorkspaceList
			return m, toastTickCmd()
		}
		// Remove archived workspace from list
		for i, ws := range m.workspaces {
			if ws.Workspace.Name == msg.name {
				m.workspaces = append(m.workspaces[:i], m.workspaces[i+1:]...)
				break
			}
		}
		if m.cursor >= len(m.workspaces) && m.cursor > 0 {
			m.cursor--
		}
		// Update workspace count in repo list
		for i := range m.repos {
			if m.repos[i].Repo.Name == m.repoName {
				m.repos[i].WorkspaceCount = len(m.workspaces)
				break
			}
		}
		m.err = nil
		m.toast = fmt.Sprintf("archived %s", msg.name)
		m.toastIsError = false
		m.toastExpiry = time.Now().Add(3 * time.Second)
		m.view = viewWorkspaceList
		return m, toastTickCmd()

	case startResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.toast = fmt.Sprintf("error starting %s", msg.name)
			m.toastIsError = true
		} else {
			for i, ws := range m.workspaces {
				if ws.Workspace.Name == msg.name {
					m.workspaces[i].Running = true
					break
				}
			}
			m.err = nil
			m.toast = fmt.Sprintf("started %s", msg.name)
			m.toastIsError = false
		}
		m.toastExpiry = time.Now().Add(3 * time.Second)
		return m, toastTickCmd()

	case stopResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.toast = fmt.Sprintf("error stopping %s", msg.name)
			m.toastIsError = true
		} else {
			for i, ws := range m.workspaces {
				if ws.Workspace.Name == msg.name {
					m.workspaces[i].Running = false
					break
				}
			}
			m.err = nil
			m.toast = fmt.Sprintf("stopped %s", msg.name)
			m.toastIsError = false
		}
		m.toastExpiry = time.Now().Add(3 * time.Second)
		return m, toastTickCmd()

	case browserResultMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		return m, nil

	case runAllResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.toast = fmt.Sprintf("error running workspaces: %v", msg.err)
			m.toastIsError = true
		} else {
			m.err = nil
			m.toast = fmt.Sprintf("started %d workspaces", msg.started)
			m.toastIsError = false
		}
		m.toastExpiry = time.Now().Add(3 * time.Second)
		refreshRunningCounts(m.repos)
		return m, toastTickCmd()

	case stopAllResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.toast = fmt.Sprintf("error stopping workspaces: %v", msg.err)
			m.toastIsError = true
		} else {
			m.err = nil
			m.toast = fmt.Sprintf("stopped %d workspaces", msg.stopped)
			m.toastIsError = false
		}
		m.toastExpiry = time.Now().Add(3 * time.Second)
		refreshRunningCounts(m.repos)
		return m, toastTickCmd()

	case batchArchiveResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.toast = "batch archive failed"
			m.toastIsError = true
			m.toastExpiry = time.Now().Add(3 * time.Second)
			m.view = viewWorkspaceList
			return m, toastTickCmd()
		}
		// Remove archived workspaces from list
		archived := make(map[string]bool, len(msg.archived))
		for _, name := range msg.archived {
			archived[name] = true
		}
		var remaining []workspaceItem
		for _, ws := range m.workspaces {
			if !archived[ws.Workspace.Name] {
				remaining = append(remaining, ws)
			}
		}
		m.workspaces = remaining
		if m.cursor >= len(m.workspaces) && m.cursor > 0 {
			m.cursor = len(m.workspaces) - 1
		}
		// Update workspace count in repo list
		for i := range m.repos {
			if m.repos[i].Repo.Name == m.repoName {
				m.repos[i].WorkspaceCount = len(m.workspaces)
				break
			}
		}
		m.batchArchiveNames = nil
		m.err = nil
		if len(msg.failed) > 0 {
			m.err = fmt.Errorf("archiving: %s", strings.Join(msg.failed, ", "))
			m.toast = fmt.Sprintf("archived %d, %d failed", len(msg.archived), len(msg.failed))
			m.toastIsError = true
		} else {
			m.toast = fmt.Sprintf("archived %d workspaces", len(msg.archived))
			m.toastIsError = false
		}
		m.toastExpiry = time.Now().Add(3 * time.Second)
		m.view = viewWorkspaceList
		return m, toastTickCmd()

	case openersLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		if len(msg.openers) == 0 {
			m.err = fmt.Errorf("no openers configured — add one with: fr8 opener add <name> <command>")
			return m, nil
		}
		m.openers = msg.openers
		if len(msg.openers) == 1 {
			// Single opener — use it directly
			ws := m.workspaces[m.openerWsIdx]
			m.openRequest = &openRequestMsg{
				workspace:  ws.Workspace,
				openerName: msg.openers[0].Name,
			}
			return m, tea.Quit
		}
		// Check for default opener
		if d := findDefaultOpener(msg.openers); d != nil {
			ws := m.workspaces[m.openerWsIdx]
			m.openRequest = &openRequestMsg{
				workspace:  ws.Workspace,
				openerName: d.Name,
			}
			return m, tea.Quit
		}
		// Multiple openers — show picker
		m.openerCursor = 0
		m.view = viewOpenerPicker
		return m, nil

	case toastTickMsg:
		if m.toast != "" && time.Now().After(m.toastExpiry) {
			m.toast = ""
			m.toastIsError = false
		}
		if m.toast != "" {
			return m, toastTickCmd()
		}
		return m, nil

	case batchStartResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.toast = fmt.Sprintf("error starting workspaces: %v", msg.err)
			m.toastIsError = true
		} else {
			m.err = nil
			m.toast = fmt.Sprintf("started %d workspaces", msg.started)
			m.toastIsError = false
		}
		m.toastExpiry = time.Now().Add(3 * time.Second)
		m.selected = nil
		refreshRunningCounts(m.repos)
		return m, toastTickCmd()

	case batchStopResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.toast = fmt.Sprintf("error stopping workspaces: %v", msg.err)
			m.toastIsError = true
		} else {
			m.err = nil
			m.toast = fmt.Sprintf("stopped %d workspaces", msg.stopped)
			m.toastIsError = false
		}
		m.toastExpiry = time.Now().Add(3 * time.Second)
		m.selected = nil
		refreshRunningCounts(m.repos)
		return m, toastTickCmd()

	case autoRefreshTickMsg:
		if m.loading {
			return m, tea.Batch(autoRefreshTickCmd(), tea.WindowSize())
		}
		return m, tea.Batch(autoRefreshCmd(), autoRefreshTickCmd(), tea.WindowSize())

	case autoRefreshResultMsg:
		if msg.err != nil {
			return m, nil
		}
		// Build running session lookup
		runningSessions := make(map[string]bool, len(msg.sessions))
		for _, s := range msg.sessions {
			runningSessions[s.Name] = true
		}
		// Update workspace running states
		if m.rootPath != "" {
			repoName := tmux.RepoName(m.rootPath)
			for i, ws := range m.workspaces {
				sessionName := tmux.SessionName(repoName, ws.Workspace.Name)
				m.workspaces[i].Running = runningSessions[sessionName]
			}
		}
		// Update repo running counts
		repoCounts := make(map[string]int)
		for _, s := range msg.sessions {
			repoCounts[s.Repo]++
		}
		for i := range m.repos {
			m.repos[i].RunningCount = repoCounts[tmux.RepoName(m.repos[i].Repo.Path)]
		}
		return m, nil
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Quit always works
	if key.Matches(msg, keys.Quit) {
		return m, tea.Quit
	}

	if m.loading {
		return m, nil
	}

	// Redraw (ctrl+l) — clear screen and re-query terminal size from any view
	if key.Matches(msg, keys.Redraw) {
		return m, tea.Batch(tea.ClearScreen, tea.WindowSize())
	}

	// While filtering, forward keys to textinput except Esc and Enter
	if m.filtering {
		return m.handleFilterKey(msg)
	}

	// Toggle help overlay from any view (except text input views)
	if key.Matches(msg, keys.Help) && m.view != viewCreateWorkspace {
		if m.view == viewHelp {
			m.view = m.previousView
		} else {
			m.previousView = m.view
			m.view = viewHelp
		}
		return m, nil
	}

	switch m.view {
	case viewHelp:
		// Any key (besides ? and q handled above) goes back
		m.view = m.previousView
		return m, nil
	case viewConfirmArchive:
		return m.handleConfirmKey(msg)
	case viewConfirmBatchArchive:
		return m.handleConfirmBatchArchiveKey(msg)
	case viewRepoList:
		return m.handleRepoKey(msg)
	case viewWorkspaceList:
		return m.handleWorkspaceKey(msg)
	case viewOpenerPicker:
		return m.handleOpenerPickerKey(msg)
	case viewCreateWorkspace:
		return m.handleCreateWorkspaceKey(msg)
	}
	return m, nil
}

func (m model) handleRepoKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtered := filteredRepos(m.repos, m.filterInput.Value())

	switch {
	case key.Matches(msg, keys.Filter):
		ti := textinput.New()
		ti.Placeholder = "filter..."
		ti.Focus()
		ti.CharLimit = 64
		m.filterInput = ti
		m.filtering = true
		m.cursor = 0
		return m, ti.Cursor.BlinkCmd()
	case key.Matches(msg, keys.Refresh):
		m.loading = true
		m.err = nil
		return m, tea.Batch(loadReposCmd, m.spinner.Tick, tea.ClearScreen)
	case key.Matches(msg, keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, keys.Down):
		if m.cursor < len(filtered)-1 {
			m.cursor++
		}
	case key.Matches(msg, keys.Enter):
		if len(filtered) > 0 {
			m.repoCursor = m.cursor // remember position for back-navigation
			m.loading = true
			m.err = nil
			origIdx := resolveOriginalRepoIndex(m.cursor, filtered, m.repos)
			repo := m.repos[origIdx].Repo
			m.filterInput.SetValue("") // clear filter on drill-down
			return m, tea.Batch(loadWorkspacesCmd(repo), m.spinner.Tick)
		}
	case key.Matches(msg, keys.Run):
		if len(filtered) > 0 {
			m.loading = true
			m.err = nil
			origIdx := resolveOriginalRepoIndex(m.cursor, filtered, m.repos)
			return m, tea.Batch(runAllCmd(m.repos[origIdx]), m.spinner.Tick)
		}
	case key.Matches(msg, keys.Stop):
		if len(filtered) > 0 {
			m.loading = true
			m.err = nil
			origIdx := resolveOriginalRepoIndex(m.cursor, filtered, m.repos)
			return m, tea.Batch(stopAllCmd(m.repos[origIdx]), m.spinner.Tick)
		}
	case key.Matches(msg, keys.RunAllGlobal):
		if len(m.repos) > 0 {
			m.loading = true
			m.err = nil
			return m, tea.Batch(runAllGlobalCmd(m.repos), m.spinner.Tick)
		}
	case key.Matches(msg, keys.StopAllGlobal):
		if len(m.repos) > 0 {
			m.loading = true
			m.err = nil
			return m, tea.Batch(stopAllGlobalCmd(), m.spinner.Tick)
		}
	}
	return m, nil
}

func (m model) handleWorkspaceKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtered := filteredWorkspaces(m.workspaces, m.filterInput.Value())

	// Helper to resolve the original workspace from the filtered cursor position
	resolveWs := func() workspaceItem {
		origIdx := resolveOriginalWsIndex(m.cursor, filtered, m.workspaces)
		return m.workspaces[origIdx]
	}

	switch {
	case key.Matches(msg, keys.Filter):
		ti := textinput.New()
		ti.Placeholder = "filter..."
		ti.Focus()
		ti.CharLimit = 64
		m.filterInput = ti
		m.filtering = true
		m.cursor = 0
		return m, ti.Cursor.BlinkCmd()
	case key.Matches(msg, keys.Refresh):
		if m.rootPath != "" {
			m.loading = true
			m.err = nil
			// Find the repo in the list
			for _, r := range m.repos {
				if r.Repo.Name == m.repoName {
					return m, tea.Batch(loadWorkspacesCmd(r.Repo), m.spinner.Tick, tea.ClearScreen)
				}
			}
		}
	case key.Matches(msg, keys.Select):
		if len(filtered) > 0 {
			if m.selected == nil {
				m.selected = make(map[int]bool)
			}
			origIdx := resolveOriginalWsIndex(m.cursor, filtered, m.workspaces)
			if m.selected[origIdx] {
				delete(m.selected, origIdx)
			} else {
				m.selected[origIdx] = true
			}
			// Advance cursor
			if m.cursor < len(filtered)-1 {
				m.cursor++
			}
		}
	case key.Matches(msg, keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, keys.Down):
		if m.cursor < len(filtered)-1 {
			m.cursor++
		}
	case key.Matches(msg, keys.Back):
		// First Esc clears selection, second navigates back
		if len(m.selected) > 0 {
			m.selected = nil
			return m, nil
		}
		m.view = viewRepoList
		m.cursor = m.repoCursor // restore remembered cursor position
		m.err = nil
		m.filterInput.SetValue("") // clear filter on back
	case key.Matches(msg, keys.Archive):
		if len(filtered) > 0 {
			origIdx := resolveOriginalWsIndex(m.cursor, filtered, m.workspaces)
			m.archiveIdx = origIdx
			m.view = viewConfirmArchive
		}
	case key.Matches(msg, keys.BatchArchive):
		if len(m.workspaces) > 0 {
			var names []string
			for _, ws := range m.workspaces {
				if ws.Merged && !ws.DirtyCount.Dirty() {
					names = append(names, ws.Workspace.Name)
				}
			}
			if len(names) == 0 {
				m.err = fmt.Errorf("no merged+clean workspaces to archive")
				return m, nil
			}
			m.batchArchiveNames = names
			m.view = viewConfirmBatchArchive
		}
	case key.Matches(msg, keys.Shell):
		if len(filtered) > 0 {
			ws := resolveWs()
			m.shellRequest = &shellRequestMsg{
				workspace: ws.Workspace,
				rootPath:  m.rootPath,
			}
			return m, tea.Quit
		}
	case key.Matches(msg, keys.Run):
		if len(filtered) > 0 {
			// Multi-select batch run
			if len(m.selected) > 0 {
				m.loading = true
				m.err = nil
				return m, tea.Batch(startSelectedCmd(m.workspaces, m.selected, m.rootPath), m.spinner.Tick)
			}
			ws := resolveWs()
			if ws.Running {
				m.err = fmt.Errorf("%q is already running", ws.Workspace.Name)
				return m, nil
			}
			m.loading = true
			m.err = nil
			return m, tea.Batch(startWorkspaceCmd(ws.Workspace, m.rootPath), m.spinner.Tick)
		}
	case key.Matches(msg, keys.Browser):
		if len(filtered) > 0 {
			ws := resolveWs()
			return m, openBrowserCmd(ws.Workspace)
		}
	case key.Matches(msg, keys.Stop):
		if len(filtered) > 0 {
			// Multi-select batch stop
			if len(m.selected) > 0 {
				m.loading = true
				m.err = nil
				return m, tea.Batch(stopSelectedCmd(m.workspaces, m.selected, m.rootPath), m.spinner.Tick)
			}
			ws := resolveWs()
			if !ws.Running {
				m.err = fmt.Errorf("%q is not running", ws.Workspace.Name)
				return m, nil
			}
			m.loading = true
			m.err = nil
			return m, tea.Batch(stopWorkspaceCmd(ws.Workspace, m.rootPath), m.spinner.Tick)
		}
	case key.Matches(msg, keys.Attach):
		if len(filtered) > 0 {
			ws := resolveWs()
			if !ws.Running {
				m.err = fmt.Errorf("%q is not running (run with r)", ws.Workspace.Name)
				return m, nil
			}
			m.attachRequest = &attachRequestMsg{
				workspace: ws.Workspace,
				rootPath:  m.rootPath,
			}
			return m, tea.Quit
		}
	case key.Matches(msg, keys.Open):
		if len(filtered) > 0 {
			origIdx := resolveOriginalWsIndex(m.cursor, filtered, m.workspaces)
			m.openerWsIdx = origIdx
			m.loading = true
			m.err = nil
			return m, tea.Batch(loadOpenersCmd(), m.spinner.Tick)
		}
	case key.Matches(msg, keys.New):
		if m.rootPath != "" {
			ti := textinput.New()
			ti.Placeholder = "workspace name (enter for auto)"
			ti.Focus()
			ti.CharLimit = 64
			m.createInput = ti
			m.view = viewCreateWorkspace
			return m, ti.Cursor.BlinkCmd()
		}
	case key.Matches(msg, keys.Enter):
		// Enter does nothing on workspace list (no further drill-down)
	}
	return m, nil
}

// handleFilterKey handles keypresses while filter mode is active.
func (m model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.filtering = false
		m.filterInput.SetValue("")
		m.cursor = 0
		return m, nil
	case tea.KeyEnter:
		m.filtering = false
		m.filterInput.Blur()
		// Keep filter value, exit filter input mode
		return m, nil
	}

	oldVal := m.filterInput.Value()
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	// Reset cursor when filter text changes
	if m.filterInput.Value() != oldVal {
		m.cursor = 0
	}
	return m, cmd
}

func (m model) handleOpenerPickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Up):
		if m.openerCursor > 0 {
			m.openerCursor--
		}
	case key.Matches(msg, keys.Down):
		if m.openerCursor < len(m.openers)-1 {
			m.openerCursor++
		}
	case key.Matches(msg, keys.Enter):
		if len(m.openers) > 0 {
			ws := m.workspaces[m.openerWsIdx]
			m.openRequest = &openRequestMsg{
				workspace:  ws.Workspace,
				openerName: m.openers[m.openerCursor].Name,
			}
			return m, tea.Quit
		}
	case key.Matches(msg, keys.Back):
		m.view = viewWorkspaceList
		m.err = nil
	}
	return m, nil
}

func (m model) handleConfirmBatchArchiveKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Yes):
		m.loading = true
		m.view = viewWorkspaceList
		return m, tea.Batch(batchArchiveCmd(m.batchArchiveNames, m.rootPath), m.spinner.Tick)
	case key.Matches(msg, keys.No):
		m.batchArchiveNames = nil
		m.view = viewWorkspaceList
	}
	return m, nil
}

func (m model) handleCreateWorkspaceKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.view = viewWorkspaceList
		m.err = nil
		return m, nil
	case tea.KeyEnter:
		name := strings.TrimSpace(m.createInput.Value())
		m.createRequest = &createRequestMsg{
			name:     name,
			rootPath: m.rootPath,
		}
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.createInput, cmd = m.createInput.Update(msg)
	return m, cmd
}

func (m model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Yes):
		ws := m.workspaces[m.archiveIdx]
		m.loading = true
		m.view = viewWorkspaceList
		return m, tea.Batch(archiveWorkspaceCmd(ws.Workspace, m.rootPath), m.spinner.Tick)
	case key.Matches(msg, keys.No):
		m.view = viewWorkspaceList
	}
	return m, nil
}

func (m model) View() string {
	var s string
	switch m.view {
	case viewRepoList:
		s = renderRepoList(m)
	case viewWorkspaceList:
		s = renderWorkspaceList(m)
	case viewConfirmArchive:
		s = renderWorkspaceList(m)
	case viewConfirmBatchArchive:
		s = renderWorkspaceList(m)
	case viewOpenerPicker:
		s = renderOpenerPicker(m)
	case viewCreateWorkspace:
		s = renderCreateWorkspace(m)
	case viewHelp:
		s = renderHelp(m)
	}
	out := constrainWidth(padToHeight(s, m.height), m.width)
	debugLogView(out, m.width, m.height)
	return out
}

// Async commands

func loadReposCmd() tea.Msg {
	regPath, err := registry.DefaultPath()
	if err != nil {
		return reposLoadedMsg{err: err}
	}

	reg, err := registry.Load(regPath)
	if err != nil {
		return reposLoadedMsg{err: err}
	}

	if len(reg.Repos) == 0 {
		return reposLoadedMsg{repos: nil}
	}

	items := make([]repoItem, len(reg.Repos))
	for i, repo := range reg.Repos {
		items[i] = repoItem{Repo: repo, WorkspaceCount: len(repo.Workspaces)}
	}

	// Enrich with running counts from tmux sessions.
	if tmux.Available() == nil {
		sessions, _ := tmux.ListFr8Sessions()
		runCounts := make(map[string]int)
		for _, s := range sessions {
			runCounts[s.Repo]++
		}
		for i := range items {
			items[i].RunningCount = runCounts[tmux.RepoName(items[i].Repo.Path)]
		}
	}

	return reposLoadedMsg{repos: items}
}

func loadWorkspacesCmd(repo registry.Repo) tea.Cmd {
	return func() tea.Msg {
		rootPath, err := git.RootWorktreePath(repo.Path)
		if err != nil {
			return workspacesLoadedMsg{err: fmt.Errorf("finding root worktree: %w", err)}
		}

		defaultBranch, _ := git.DefaultBranch(rootPath)

		hasTmux := tmux.Available() == nil
		repoName := tmux.RepoName(rootPath)

		// Build running session lookup map (one subprocess instead of N)
		runningSessions := make(map[string]bool)
		if hasTmux {
			sessions, _ := tmux.ListFr8Sessions()
			for _, s := range sessions {
				runningSessions[s.Name] = true
			}
		}

		items := make([]workspaceItem, len(repo.Workspaces))

		// Fan out git enrichment per workspace in parallel
		type enrichResult struct {
			idx  int
			item workspaceItem
		}
		gitCh := make(chan enrichResult, len(repo.Workspaces))
		for i, ws := range repo.Workspaces {
			go func(idx int, ws registry.Workspace) {
				branch, _ := git.CurrentBranch(ws.Path)
				item := workspaceItem{Workspace: ws, Branch: branch}
				item.PortFree = port.IsFree(ws.Port)

				if hasTmux {
					sessionName := tmux.SessionName(repoName, ws.Name)
					item.Running = runningSessions[sessionName]
				}

				dc, err := git.DirtyStatus(ws.Path)
				if err != nil {
					item.StatusErr = err
					gitCh <- enrichResult{idx: idx, item: item}
					return
				}
				item.DirtyCount = dc

				ci, err := git.LastCommit(ws.Path)
				if err == nil {
					item.LastCommit = &ci
				}

				if defaultBranch != "" {
					merged, err := git.IsMerged(ws.Path, branch, defaultBranch)
					if err == nil {
						item.Merged = merged
					}

					da, db, err := git.AheadBehind(ws.Path, branch, defaultBranch)
					if err == nil {
						item.DefaultAhead = da
						item.DefaultBehind = db
					}
				}

				tracking, err := git.TrackingBranch(ws.Path, branch)
				if err == nil {
					ahead, behind, err := git.AheadBehind(ws.Path, branch, tracking)
					if err == nil {
						item.Ahead = ahead
						item.Behind = behind
					}
				}

				gitCh <- enrichResult{idx: idx, item: item}
			}(i, ws)
		}
		for range repo.Workspaces {
			res := <-gitCh
			items[res.idx] = res.item
		}

		// Fan out PR queries in parallel if gh is available.
		if gh.Available() == nil {
			type prResult struct {
				idx int
				pr  *gh.PRInfo
			}
			ch := make(chan prResult, len(items))
			for i, item := range items {
				go func(idx int, branch string, ws registry.Workspace) {
					pr, _ := gh.PRStatus(ws.Path, branch)
					ch <- prResult{idx: idx, pr: pr}
				}(i, item.Branch, item.Workspace)
			}
			for range items {
				res := <-ch
				items[res.idx].PR = res.pr
			}
		}

		return workspacesLoadedMsg{
			workspaces:    items,
			repoName:      repo.Name,
			rootPath:      rootPath,
			defaultBranch: defaultBranch,
		}
	}
}

func startWorkspaceCmd(ws registry.Workspace, rootPath string) tea.Cmd {
	return func() tea.Msg {
		if err := tmux.Available(); err != nil {
			return startResultMsg{name: ws.Name, err: err}
		}

		cfg, err := config.Load(rootPath)
		if err != nil {
			return startResultMsg{name: ws.Name, err: fmt.Errorf("loading config: %w", err)}
		}

		if cfg.Scripts.Run == "" {
			return startResultMsg{name: ws.Name, err: fmt.Errorf("no run script configured")}
		}

		defaultBranch, _ := git.DefaultBranch(rootPath)
		envVars := env.BuildFr8Only(&ws, rootPath, defaultBranch)

		repoName := tmux.RepoName(rootPath)
		sessionName := tmux.SessionName(repoName, ws.Name)
		if err := tmux.Start(sessionName, ws.Path, cfg.Scripts.Run, envVars); err != nil {
			return startResultMsg{name: ws.Name, err: err}
		}

		return startResultMsg{name: ws.Name}
	}
}

func stopWorkspaceCmd(ws registry.Workspace, rootPath string) tea.Cmd {
	return func() tea.Msg {
		if err := tmux.Available(); err != nil {
			return stopResultMsg{name: ws.Name, err: err}
		}

		repoName := tmux.RepoName(rootPath)
		sessionName := tmux.SessionName(repoName, ws.Name)
		if err := tmux.Stop(sessionName); err != nil {
			return stopResultMsg{name: ws.Name, err: err}
		}

		return stopResultMsg{name: ws.Name}
	}
}

func openBrowserCmd(ws registry.Workspace) tea.Cmd {
	return func() tea.Msg {
		url := fmt.Sprintf("http://localhost:%d", ws.Port)
		err := openURL(url)
		return browserResultMsg{name: ws.Name, err: err}
	}
}

func openURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform %s", runtime.GOOS)
	}
	return cmd.Start()
}

func runAllCmd(item repoItem) tea.Cmd {
	return func() tea.Msg {
		repo := item.Repo
		if err := tmux.Available(); err != nil {
			return runAllResultMsg{repoName: repo.Name, err: err}
		}

		rootPath, err := git.RootWorktreePath(repo.Path)
		if err != nil {
			return runAllResultMsg{repoName: repo.Name, err: err}
		}

		cfg, err := config.Load(rootPath)
		if err != nil {
			return runAllResultMsg{repoName: repo.Name, err: err}
		}
		if cfg.Scripts.Run == "" {
			return runAllResultMsg{repoName: repo.Name, err: fmt.Errorf("no run script configured for %s", repo.Name)}
		}

		defaultBranch, _ := git.DefaultBranch(rootPath)
		repoName := tmux.RepoName(rootPath)

		// Build running session lookup map (one subprocess instead of N)
		runningSessions := make(map[string]bool)
		sessions, _ := tmux.ListFr8Sessions()
		for _, s := range sessions {
			runningSessions[s.Name] = true
		}

		var started int
		for _, ws := range repo.Workspaces {
			sessionName := tmux.SessionName(repoName, ws.Name)
			if runningSessions[sessionName] {
				continue
			}
			envVars := env.BuildFr8Only(&ws, rootPath, defaultBranch)
			if err := tmux.Start(sessionName, ws.Path, cfg.Scripts.Run, envVars); err != nil {
				return runAllResultMsg{repoName: repo.Name, started: started, err: err}
			}
			started++
		}

		return runAllResultMsg{repoName: repo.Name, started: started}
	}
}

func stopAllCmd(item repoItem) tea.Cmd {
	return func() tea.Msg {
		repo := item.Repo
		if err := tmux.Available(); err != nil {
			return stopAllResultMsg{repoName: repo.Name, err: err}
		}

		sessions, err := tmux.ListFr8Sessions()
		if err != nil {
			return stopAllResultMsg{repoName: repo.Name, err: err}
		}

		repoName := tmux.RepoName(repo.Path)
		var stopped int
		for _, s := range sessions {
			if s.Repo != repoName {
				continue
			}
			if err := tmux.Stop(s.Name); err != nil {
				return stopAllResultMsg{repoName: repo.Name, stopped: stopped, err: err}
			}
			stopped++
		}

		return stopAllResultMsg{repoName: repo.Name, stopped: stopped}
	}
}

func runAllGlobalCmd(items []repoItem) tea.Cmd {
	return func() tea.Msg {
		if err := tmux.Available(); err != nil {
			return runAllResultMsg{err: err}
		}

		// Build running session lookup map (one subprocess instead of N)
		runningSessions := make(map[string]bool)
		sessions, _ := tmux.ListFr8Sessions()
		for _, s := range sessions {
			runningSessions[s.Name] = true
		}

		// Collect all start jobs
		type startJob struct {
			sessionName string
			dir         string
			runScript   string
			envVars     []string
		}
		var jobs []startJob

		for _, item := range items {
			if item.Err != nil {
				continue
			}
			repo := item.Repo

			rootPath, err := git.RootWorktreePath(repo.Path)
			if err != nil {
				continue
			}

			cfg, err := config.Load(rootPath)
			if err != nil || cfg.Scripts.Run == "" {
				continue
			}

			defaultBranch, _ := git.DefaultBranch(rootPath)
			repoName := tmux.RepoName(rootPath)

			for _, ws := range repo.Workspaces {
				sessionName := tmux.SessionName(repoName, ws.Name)
				if runningSessions[sessionName] {
					continue
				}
				envVars := env.BuildFr8Only(&ws, rootPath, defaultBranch)
				jobs = append(jobs, startJob{
					sessionName: sessionName,
					dir:         ws.Path,
					runScript:   cfg.Scripts.Run,
					envVars:     envVars,
				})
			}
		}

		// Fan out tmux starts with bounded concurrency
		const maxConcurrent = 5
		sem := make(chan struct{}, maxConcurrent)
		results := make(chan bool, len(jobs))
		for _, job := range jobs {
			sem <- struct{}{}
			go func(j startJob) {
				defer func() { <-sem }()
				err := tmux.Start(j.sessionName, j.dir, j.runScript, j.envVars)
				results <- (err == nil)
			}(job)
		}
		var totalStarted int
		for range jobs {
			if <-results {
				totalStarted++
			}
		}

		return runAllResultMsg{started: totalStarted}
	}
}

func stopAllGlobalCmd() tea.Cmd {
	return func() tea.Msg {
		if err := tmux.Available(); err != nil {
			return stopAllResultMsg{err: err}
		}

		sessions, err := tmux.ListFr8Sessions()
		if err != nil {
			return stopAllResultMsg{err: err}
		}

		var stopped int
		for _, s := range sessions {
			if err := tmux.Stop(s.Name); err != nil {
				continue
			}
			stopped++
		}

		return stopAllResultMsg{stopped: stopped}
	}
}

func loadOpenersCmd() tea.Cmd {
	return func() tea.Msg {
		path, err := userconfig.DefaultPath()
		if err != nil {
			return openersLoadedMsg{err: err}
		}
		cfg, err := userconfig.Load(path)
		if err != nil {
			return openersLoadedMsg{err: err}
		}
		return openersLoadedMsg{openers: cfg.Openers}
	}
}

// findDefaultOpener returns the opener marked as default, or nil if none.
func findDefaultOpener(openers []userconfig.Opener) *userconfig.Opener {
	for i := range openers {
		if openers[i].Default {
			return &openers[i]
		}
	}
	return nil
}

// --- Filter helpers ---

// filteredRepos returns repos matching the query (case-insensitive substring).
func filteredRepos(repos []repoItem, query string) []repoItem {
	if query == "" {
		return repos
	}
	q := strings.ToLower(query)
	var result []repoItem
	for _, r := range repos {
		if strings.Contains(strings.ToLower(r.Repo.Name), q) {
			result = append(result, r)
		}
	}
	return result
}

// filteredWorkspaces returns workspaces matching the query (case-insensitive
// substring match on name or branch).
func filteredWorkspaces(workspaces []workspaceItem, query string) []workspaceItem {
	if query == "" {
		return workspaces
	}
	q := strings.ToLower(query)
	var result []workspaceItem
	for _, ws := range workspaces {
		if strings.Contains(strings.ToLower(ws.Workspace.Name), q) ||
			strings.Contains(strings.ToLower(ws.Branch), q) {
			result = append(result, ws)
		}
	}
	return result
}

// resolveOriginalRepoIndex maps a cursor index in the filtered list back to
// the original repos slice index.
func resolveOriginalRepoIndex(cursor int, filtered, original []repoItem) int {
	if cursor >= len(filtered) {
		return 0
	}
	target := filtered[cursor]
	for i, r := range original {
		if r.Repo.Name == target.Repo.Name && r.Repo.Path == target.Repo.Path {
			return i
		}
	}
	return 0
}

// resolveOriginalWsIndex maps a cursor index in the filtered list back to
// the original workspaces slice index.
func resolveOriginalWsIndex(cursor int, filtered, original []workspaceItem) int {
	if cursor >= len(filtered) {
		return 0
	}
	target := filtered[cursor]
	for i, ws := range original {
		if ws.Workspace.Name == target.Workspace.Name && ws.Workspace.Path == target.Workspace.Path {
			return i
		}
	}
	return 0
}

// --- Toast / timer commands ---

func toastTickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return toastTickMsg{}
	})
}

func autoRefreshTickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg {
		return autoRefreshTickMsg{}
	})
}

func autoRefreshCmd() tea.Cmd {
	return func() tea.Msg {
		if tmux.Available() != nil {
			return autoRefreshResultMsg{}
		}
		sessions, err := tmux.ListFr8Sessions()
		return autoRefreshResultMsg{sessions: sessions, err: err}
	}
}

// --- Multi-select batch commands ---

func startSelectedCmd(workspaces []workspaceItem, selected map[int]bool, rootPath string) tea.Cmd {
	return func() tea.Msg {
		if err := tmux.Available(); err != nil {
			return batchStartResultMsg{err: err}
		}

		cfg, err := config.Load(rootPath)
		if err != nil {
			return batchStartResultMsg{err: fmt.Errorf("loading config: %w", err)}
		}
		if cfg.Scripts.Run == "" {
			return batchStartResultMsg{err: fmt.Errorf("no run script configured")}
		}

		defaultBranch, _ := git.DefaultBranch(rootPath)
		repoName := tmux.RepoName(rootPath)

		var started int
		for idx := range selected {
			if idx >= len(workspaces) {
				continue
			}
			ws := workspaces[idx]
			if ws.Running {
				continue
			}
			sessionName := tmux.SessionName(repoName, ws.Workspace.Name)
			envVars := env.BuildFr8Only(&ws.Workspace, rootPath, defaultBranch)
			if err := tmux.Start(sessionName, ws.Workspace.Path, cfg.Scripts.Run, envVars); err != nil {
				return batchStartResultMsg{started: started, err: err}
			}
			started++
		}

		return batchStartResultMsg{started: started}
	}
}

func stopSelectedCmd(workspaces []workspaceItem, selected map[int]bool, rootPath string) tea.Cmd {
	return func() tea.Msg {
		if err := tmux.Available(); err != nil {
			return batchStopResultMsg{err: err}
		}

		repoName := tmux.RepoName(rootPath)

		var stopped int
		for idx := range selected {
			if idx >= len(workspaces) {
				continue
			}
			ws := workspaces[idx]
			if !ws.Running {
				continue
			}
			sessionName := tmux.SessionName(repoName, ws.Workspace.Name)
			if err := tmux.Stop(sessionName); err != nil {
				return batchStopResultMsg{stopped: stopped, err: err}
			}
			stopped++
		}

		return batchStopResultMsg{stopped: stopped}
	}
}

// refreshRunningCounts re-derives RunningCount on all repos from tmux sessions.
func refreshRunningCounts(repos []repoItem) {
	if tmux.Available() != nil {
		for i := range repos {
			repos[i].RunningCount = 0
		}
		return
	}
	sessions, _ := tmux.ListFr8Sessions()
	counts := make(map[string]int)
	for _, s := range sessions {
		counts[s.Repo]++
	}
	for i := range repos {
		repos[i].RunningCount = counts[tmux.RepoName(repos[i].Repo.Path)]
	}
}

func batchArchiveCmd(names []string, rootPath string) tea.Cmd {
	return func() tea.Msg {
		regPath, err := registry.DefaultPath()
		if err != nil {
			return batchArchiveResultMsg{err: fmt.Errorf("finding state path: %w", err)}
		}
		reg, err := registry.Load(regPath)
		if err != nil {
			return batchArchiveResultMsg{err: fmt.Errorf("loading registry: %w", err)}
		}
		repo := reg.FindByPath(rootPath)
		if repo == nil {
			return batchArchiveResultMsg{err: fmt.Errorf("repo not found for path %s", rootPath)}
		}

		cfg, err := config.Load(rootPath)
		if err != nil {
			return batchArchiveResultMsg{err: fmt.Errorf("loading config: %w", err)}
		}

		defaultBranch, _ := git.DefaultBranch(rootPath)
		repoName := tmux.RepoName(rootPath)

		var archived, failed []string
		for _, name := range names {
			ws := repo.FindWorkspace(name)
			if ws == nil {
				failed = append(failed, name)
				continue
			}

			// Stop tmux session
			if tmux.Available() == nil {
				sessionName := tmux.SessionName(repoName, ws.Name)
				_ = tmux.Stop(sessionName)
			}

			// Run archive script
			if cfg.Scripts.Archive != "" {
				envVars := env.Build(ws, rootPath, defaultBranch)
				cmd := exec.Command("sh", "-c", cfg.Scripts.Archive)
				cmd.Dir = ws.Path
				cmd.Env = envVars
				var buf bytes.Buffer
				cmd.Stdout = &buf
				cmd.Stderr = &buf
				if err := cmd.Run(); err != nil {
					failed = append(failed, name)
					continue
				}
			}

			// Remove worktree
			if err := git.WorktreeRemove(rootPath, ws.Path); err != nil {
				failed = append(failed, name)
				continue
			}

			archived = append(archived, name)
		}

		// Batch registry update
		for _, name := range archived {
			_ = repo.RemoveWorkspace(name)
		}
		if err := reg.Save(regPath); err != nil {
			return batchArchiveResultMsg{err: fmt.Errorf("saving state: %w", err)}
		}

		return batchArchiveResultMsg{archived: archived, failed: failed}
	}
}

func archiveWorkspaceCmd(ws registry.Workspace, rootPath string) tea.Cmd {
	return func() tea.Msg {
		// Auto-stop tmux session before archiving
		if tmux.Available() == nil {
			repoName := tmux.RepoName(rootPath)
			sessionName := tmux.SessionName(repoName, ws.Name)
			_ = tmux.Stop(sessionName) // best-effort, ignore errors
		}

		cfg, err := config.Load(rootPath)
		if err != nil {
			return archiveResultMsg{name: ws.Name, err: fmt.Errorf("loading config: %w", err)}
		}

		// Run archive script with captured output
		defaultBranch, _ := git.DefaultBranch(rootPath)
		if cfg.Scripts.Archive != "" {
			envVars := env.Build(&ws, rootPath, defaultBranch)
			cmd := exec.Command("sh", "-c", cfg.Scripts.Archive)
			cmd.Dir = ws.Path
			cmd.Env = envVars
			var buf bytes.Buffer
			cmd.Stdout = &buf
			cmd.Stderr = &buf
			if err := cmd.Run(); err != nil {
				return archiveResultMsg{
					name: ws.Name,
					err:  fmt.Errorf("archive script failed: %w\n%s", err, buf.String()),
				}
			}
		}

		// Remove worktree
		if err := git.WorktreeRemove(rootPath, ws.Path); err != nil {
			return archiveResultMsg{name: ws.Name, err: fmt.Errorf("removing worktree: %w", err)}
		}

		// Update registry
		regPath, err := registry.DefaultPath()
		if err != nil {
			return archiveResultMsg{name: ws.Name, err: fmt.Errorf("finding state path: %w", err)}
		}
		reg, err := registry.Load(regPath)
		if err != nil {
			return archiveResultMsg{name: ws.Name, err: fmt.Errorf("loading registry: %w", err)}
		}
		repo := reg.FindByPath(rootPath)
		if repo != nil {
			_ = repo.RemoveWorkspace(ws.Name)
			if err := reg.Save(regPath); err != nil {
				return archiveResultMsg{name: ws.Name, err: fmt.Errorf("saving state: %w", err)}
			}
		}

		return archiveResultMsg{name: ws.Name}
	}
}
