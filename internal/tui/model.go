package tui

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/thomascarr/fr8/internal/config"
	"github.com/thomascarr/fr8/internal/env"
	"github.com/thomascarr/fr8/internal/git"
	"github.com/thomascarr/fr8/internal/port"
	"github.com/thomascarr/fr8/internal/registry"
	"github.com/thomascarr/fr8/internal/state"
)

type model struct {
	view         viewState
	repos        []repoItem
	workspaces   []workspaceItem
	cursor       int
	loading      bool
	err          error
	repoName     string // current repo being viewed
	rootPath     string // root worktree path for current repo
	commonDir    string // git common dir for current repo
	shellRequest   *shellRequestMsg
	runRequest     *runRequestMsg
	browserRequest *browserRequestMsg
	archiveIdx   int // workspace index pending archive confirmation
	width        int
	height       int
	spinner      spinner.Model
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
		m.cursor = 0
		m.view = viewWorkspaceList
		return m, nil

	case archiveResultMsg:
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

	switch m.view {
	case viewConfirmArchive:
		return m.handleConfirmKey(msg)
	case viewRepoList:
		return m.handleRepoKey(msg)
	case viewWorkspaceList:
		return m.handleWorkspaceKey(msg)
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
			m.runRequest = &runRequestMsg{
				workspace: ws.Workspace,
				rootPath:  m.rootPath,
			}
			return m, tea.Quit
		}
	case key.Matches(msg, keys.Browser):
		if len(m.workspaces) > 0 {
			ws := m.workspaces[m.cursor]
			m.browserRequest = &browserRequestMsg{
				workspace: ws.Workspace,
				rootPath:  m.rootPath,
			}
			return m, tea.Quit
		}
	case key.Matches(msg, keys.Enter):
		// Enter does nothing on workspace list (no further drill-down)
	}
	return m, nil
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
	switch m.view {
	case viewRepoList:
		return renderRepoList(m)
	case viewWorkspaceList:
		return renderWorkspaceList(m)
	case viewConfirmArchive:
		return renderWorkspaceList(m)
	}
	return ""
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

		items := make([]workspaceItem, len(st.Workspaces))
		for i, ws := range st.Workspaces {
			items[i] = workspaceItem{Workspace: ws}
			items[i].PortFree = port.IsFree(ws.Port)

			dirty, err := git.HasUncommittedChanges(ws.Path)
			if err != nil {
				items[i].StatusErr = err
				continue
			}
			items[i].Dirty = dirty

			if defaultBranch != "" {
				merged, err := git.IsMerged(ws.Path, ws.Branch, defaultBranch)
				if err == nil {
					items[i].Merged = merged
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

		return workspacesLoadedMsg{
			workspaces: items,
			repoName:   repo.Name,
			rootPath:   rootPath,
			commonDir:  commonDir,
		}
	}
}

func archiveWorkspaceCmd(ws state.Workspace, rootPath, commonDir string) tea.Cmd {
	return func() tea.Msg {
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
		st.Remove(ws.Name)
		if err := st.Save(commonDir); err != nil {
			return archiveResultMsg{name: ws.Name, err: fmt.Errorf("saving state: %w", err)}
		}

		return archiveResultMsg{name: ws.Name}
	}
}
