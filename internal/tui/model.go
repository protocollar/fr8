package tui

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/protocollar/fr8/internal/config"
	"github.com/protocollar/fr8/internal/env"
	"github.com/protocollar/fr8/internal/gh"
	"github.com/protocollar/fr8/internal/git"
	"github.com/protocollar/fr8/internal/opener"
	"github.com/protocollar/fr8/internal/port"
	"github.com/protocollar/fr8/internal/registry"
	"github.com/protocollar/fr8/internal/state"
	"github.com/protocollar/fr8/internal/tmux"
)

type model struct {
	view              viewState
	previousView      viewState
	repos             []repoItem
	workspaces        []workspaceItem
	cursor            int
	loading           bool
	err               error
	repoName          string // current repo being viewed
	rootPath          string // root worktree path for current repo
	commonDir         string // git common dir for current repo
	defaultBranch     string // default branch for current repo
	shellRequest      *shellRequestMsg
	attachRequest     *attachRequestMsg
	openRequest       *openRequestMsg
	createRequest     *createRequestMsg
	archiveIdx        int // workspace index pending archive confirmation
	batchArchiveNames []string
	openers           []opener.Opener
	openerCursor      int
	openerWsIdx       int // workspace index for which opener picker was opened
	createInput       textinput.Model
	width             int
	height            int
	spinner           spinner.Model
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
	return tea.Batch(loadReposCmd, m.spinner.Tick)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
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
		m.commonDir = msg.commonDir
		m.defaultBranch = msg.defaultBranch
		m.cursor = 0
		m.view = viewWorkspaceList
		return m, nil

	case archiveResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.view = viewWorkspaceList
			return m, nil
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
		m.view = viewWorkspaceList
		return m, nil

	case startResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		for i, ws := range m.workspaces {
			if ws.Workspace.Name == msg.name {
				m.workspaces[i].Running = true
				break
			}
		}
		m.err = nil
		return m, nil

	case stopResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		for i, ws := range m.workspaces {
			if ws.Workspace.Name == msg.name {
				m.workspaces[i].Running = false
				break
			}
		}
		m.err = nil
		return m, nil

	case browserResultMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		return m, nil

	case runAllResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
		}
		refreshRunningCounts(m.repos)
		return m, nil

	case stopAllResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
		}
		refreshRunningCounts(m.repos)
		return m, nil

	case batchArchiveResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.view = viewWorkspaceList
			return m, nil
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
		}
		m.view = viewWorkspaceList
		return m, nil

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
		if d := opener.FindDefault(msg.openers); d != nil {
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
	switch {
	case key.Matches(msg, keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, keys.Down):
		if m.cursor < len(m.repos)-1 {
			m.cursor++
		}
	case key.Matches(msg, keys.Enter):
		if len(m.repos) > 0 {
			m.loading = true
			m.err = nil
			repo := m.repos[m.cursor].Repo
			return m, tea.Batch(loadWorkspacesCmd(repo), m.spinner.Tick)
		}
	case key.Matches(msg, keys.Run):
		if len(m.repos) > 0 {
			m.loading = true
			m.err = nil
			return m, tea.Batch(runAllCmd(m.repos[m.cursor]), m.spinner.Tick)
		}
	case key.Matches(msg, keys.Stop):
		if len(m.repos) > 0 {
			m.loading = true
			m.err = nil
			return m, tea.Batch(stopAllCmd(m.repos[m.cursor]), m.spinner.Tick)
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
	switch {
	case key.Matches(msg, keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, keys.Down):
		if m.cursor < len(m.workspaces)-1 {
			m.cursor++
		}
	case key.Matches(msg, keys.Back):
		m.view = viewRepoList
		m.cursor = 0
		m.err = nil
	case key.Matches(msg, keys.Archive):
		if len(m.workspaces) > 0 {
			m.archiveIdx = m.cursor
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
		if len(m.workspaces) > 0 {
			ws := m.workspaces[m.cursor]
			m.shellRequest = &shellRequestMsg{
				workspace: ws.Workspace,
				rootPath:  m.rootPath,
			}
			return m, tea.Quit
		}
	case key.Matches(msg, keys.Run):
		if len(m.workspaces) > 0 {
			ws := m.workspaces[m.cursor]
			if ws.Running {
				m.err = fmt.Errorf("%q is already running", ws.Workspace.Name)
				return m, nil
			}
			m.loading = true
			m.err = nil
			return m, tea.Batch(startWorkspaceCmd(ws.Workspace, m.rootPath, m.commonDir), m.spinner.Tick)
		}
	case key.Matches(msg, keys.Browser):
		if len(m.workspaces) > 0 {
			ws := m.workspaces[m.cursor]
			return m, openBrowserCmd(ws.Workspace)
		}
	case key.Matches(msg, keys.Stop):
		if len(m.workspaces) > 0 {
			ws := m.workspaces[m.cursor]
			if !ws.Running {
				m.err = fmt.Errorf("%q is not running", ws.Workspace.Name)
				return m, nil
			}
			m.loading = true
			m.err = nil
			return m, tea.Batch(stopWorkspaceCmd(ws.Workspace, m.rootPath), m.spinner.Tick)
		}
	case key.Matches(msg, keys.Attach):
		if len(m.workspaces) > 0 {
			ws := m.workspaces[m.cursor]
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
		if len(m.workspaces) > 0 {
			m.openerWsIdx = m.cursor
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
		return m, tea.Batch(batchArchiveCmd(m.batchArchiveNames, m.rootPath, m.commonDir), m.spinner.Tick)
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
			name:      name,
			rootPath:  m.rootPath,
			commonDir: m.commonDir,
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
		return m, tea.Batch(archiveWorkspaceCmd(ws.Workspace, m.rootPath, m.commonDir), m.spinner.Tick)
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
	return padToHeight(s, m.height)
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
		items[i] = repoItem{Repo: repo}
		commonDir, err := git.CommonDir(repo.Path)
		if err != nil {
			items[i].Err = err
			continue
		}
		st, err := state.Load(commonDir)
		if err != nil {
			items[i].Err = err
			continue
		}
		items[i].WorkspaceCount = len(st.Workspaces)
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
		commonDir, err := git.CommonDir(repo.Path)
		if err != nil {
			return workspacesLoadedMsg{err: fmt.Errorf("reading git data for %s: %w", repo.Name, err)}
		}

		rootPath, err := git.RootWorktreePath(repo.Path)
		if err != nil {
			return workspacesLoadedMsg{err: fmt.Errorf("finding root worktree: %w", err)}
		}

		st, err := state.Load(commonDir)
		if err != nil {
			return workspacesLoadedMsg{err: fmt.Errorf("loading state for %s: %w", repo.Name, err)}
		}

		defaultBranch, _ := git.DefaultBranch(rootPath)

		hasTmux := tmux.Available() == nil
		repoName := tmux.RepoName(rootPath)

		items := make([]workspaceItem, len(st.Workspaces))
		for i, ws := range st.Workspaces {
			items[i] = workspaceItem{Workspace: ws}
			items[i].PortFree = port.IsFree(ws.Port)

			if hasTmux {
				sessionName := tmux.SessionName(repoName, ws.Name)
				items[i].Running = tmux.IsRunning(sessionName)
			}

			dc, err := git.DirtyStatus(ws.Path)
			if err != nil {
				items[i].StatusErr = err
				continue
			}
			items[i].DirtyCount = dc

			ci, err := git.LastCommit(ws.Path)
			if err == nil {
				items[i].LastCommit = &ci
			}

			if defaultBranch != "" {
				merged, err := git.IsMerged(ws.Path, ws.Branch, defaultBranch)
				if err == nil {
					items[i].Merged = merged
				}

				da, db, err := git.AheadBehind(ws.Path, ws.Branch, defaultBranch)
				if err == nil {
					items[i].DefaultAhead = da
					items[i].DefaultBehind = db
				}
			}

			tracking, err := git.TrackingBranch(ws.Path, ws.Branch)
			if err == nil {
				ahead, behind, err := git.AheadBehind(ws.Path, ws.Branch, tracking)
				if err == nil {
					items[i].Ahead = ahead
					items[i].Behind = behind
				}
			}
		}

		// Fan out PR queries in parallel if gh is available.
		if gh.Available() == nil {
			type prResult struct {
				idx int
				pr  *gh.PRInfo
			}
			ch := make(chan prResult, len(items))
			for i, item := range items {
				go func(idx int, ws state.Workspace) {
					pr, _ := gh.PRStatus(ws.Path, ws.Branch)
					ch <- prResult{idx: idx, pr: pr}
				}(i, item.Workspace)
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
			commonDir:     commonDir,
			defaultBranch: defaultBranch,
		}
	}
}

func startWorkspaceCmd(ws state.Workspace, rootPath, commonDir string) tea.Cmd {
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

func stopWorkspaceCmd(ws state.Workspace, rootPath string) tea.Cmd {
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

func openBrowserCmd(ws state.Workspace) tea.Cmd {
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

		commonDir, err := git.CommonDir(repo.Path)
		if err != nil {
			return runAllResultMsg{repoName: repo.Name, err: err}
		}

		rootPath, err := git.RootWorktreePath(repo.Path)
		if err != nil {
			return runAllResultMsg{repoName: repo.Name, err: err}
		}

		st, err := state.Load(commonDir)
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

		var started int
		for _, ws := range st.Workspaces {
			sessionName := tmux.SessionName(repoName, ws.Name)
			if tmux.IsRunning(sessionName) {
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

		var totalStarted int
		for _, item := range items {
			if item.Err != nil {
				continue
			}
			repo := item.Repo

			commonDir, err := git.CommonDir(repo.Path)
			if err != nil {
				continue
			}

			rootPath, err := git.RootWorktreePath(repo.Path)
			if err != nil {
				continue
			}

			st, err := state.Load(commonDir)
			if err != nil {
				continue
			}

			cfg, err := config.Load(rootPath)
			if err != nil || cfg.Scripts.Run == "" {
				continue
			}

			defaultBranch, _ := git.DefaultBranch(rootPath)
			repoName := tmux.RepoName(rootPath)

			for _, ws := range st.Workspaces {
				sessionName := tmux.SessionName(repoName, ws.Name)
				if tmux.IsRunning(sessionName) {
					continue
				}
				envVars := env.BuildFr8Only(&ws, rootPath, defaultBranch)
				if err := tmux.Start(sessionName, ws.Path, cfg.Scripts.Run, envVars); err != nil {
					continue
				}
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
		path, err := opener.DefaultPath()
		if err != nil {
			return openersLoadedMsg{err: err}
		}
		openers, err := opener.Load(path)
		if err != nil {
			return openersLoadedMsg{err: err}
		}
		return openersLoadedMsg{openers: openers}
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

func batchArchiveCmd(names []string, rootPath, commonDir string) tea.Cmd {
	return func() tea.Msg {
		st, err := state.Load(commonDir)
		if err != nil {
			return batchArchiveResultMsg{err: fmt.Errorf("loading state: %w", err)}
		}

		cfg, err := config.Load(rootPath)
		if err != nil {
			return batchArchiveResultMsg{err: fmt.Errorf("loading config: %w", err)}
		}

		defaultBranch, _ := git.DefaultBranch(rootPath)
		repoName := tmux.RepoName(rootPath)

		var archived, failed []string
		for _, name := range names {
			ws := st.Find(name)
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

		// Batch state update
		for _, name := range archived {
			_ = st.Remove(name)
		}
		if err := st.Save(commonDir); err != nil {
			return batchArchiveResultMsg{err: fmt.Errorf("saving state: %w", err)}
		}

		return batchArchiveResultMsg{archived: archived, failed: failed}
	}
}

func archiveWorkspaceCmd(ws state.Workspace, rootPath, commonDir string) tea.Cmd {
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

		// Update state
		st, err := state.Load(commonDir)
		if err != nil {
			return archiveResultMsg{name: ws.Name, err: fmt.Errorf("loading state: %w", err)}
		}
		_ = st.Remove(ws.Name)
		if err := st.Save(commonDir); err != nil {
			return archiveResultMsg{name: ws.Name, err: fmt.Errorf("saving state: %w", err)}
		}

		return archiveResultMsg{name: ws.Name}
	}
}
