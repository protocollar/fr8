package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/thomascarr/fr8/internal/registry"
	"github.com/thomascarr/fr8/internal/state"
)

// Key helpers for constructing tea.KeyMsg values.
func keyRune(r rune) tea.KeyMsg   { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }
func keyEnter() tea.KeyMsg        { return tea.KeyMsg{Type: tea.KeyEnter} }
func keyEsc() tea.KeyMsg          { return tea.KeyMsg{Type: tea.KeyEsc} }

func seedRepoModel() model {
	return model{
		view: viewRepoList,
		repos: []repoItem{
			{Repo: registry.Repo{Name: "alpha", Path: "/a"}},
			{Repo: registry.Repo{Name: "bravo", Path: "/b"}},
			{Repo: registry.Repo{Name: "charlie", Path: "/c"}},
		},
		cursor: 0,
	}
}

func seedWorkspaceModel() model {
	return model{
		view:     viewWorkspaceList,
		repoName: "alpha",
		rootPath: "/a",
		commonDir: "/a/.git",
		workspaces: []workspaceItem{
			{Workspace: state.Workspace{Name: "ws-one", Branch: "feat-1", Port: 3000}},
			{Workspace: state.Workspace{Name: "ws-two", Branch: "feat-2", Port: 3010}},
			{Workspace: state.Workspace{Name: "ws-three", Branch: "feat-3", Port: 3020}},
		},
		cursor: 0,
	}
}

func updateModel(m model, msgs ...tea.Msg) model {
	for _, msg := range msgs {
		result, _ := m.Update(msg)
		m = result.(model)
	}
	return m
}

func TestRepoListNavigation(t *testing.T) {
	m := seedRepoModel()

	// j moves cursor down
	m = updateModel(m, keyRune('j'))
	if m.cursor != 1 {
		t.Errorf("after j: cursor = %d, want 1", m.cursor)
	}

	m = updateModel(m, keyRune('j'))
	if m.cursor != 2 {
		t.Errorf("after j j: cursor = %d, want 2", m.cursor)
	}

	// Clamp at bottom
	m = updateModel(m, keyRune('j'))
	if m.cursor != 2 {
		t.Errorf("after j at bottom: cursor = %d, want 2", m.cursor)
	}

	// k moves cursor up
	m = updateModel(m, keyRune('k'))
	if m.cursor != 1 {
		t.Errorf("after k: cursor = %d, want 1", m.cursor)
	}

	// Clamp at top
	m = updateModel(m, keyRune('k'), keyRune('k'))
	if m.cursor != 0 {
		t.Errorf("after k k at top: cursor = %d, want 0", m.cursor)
	}
}

func TestRepoListEnterTriggersLoad(t *testing.T) {
	m := seedRepoModel()

	result, cmd := m.Update(keyEnter())
	m = result.(model)

	if !m.loading {
		t.Error("expected loading=true after Enter")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd after Enter")
	}
}

func TestWorkspaceListNavigation(t *testing.T) {
	m := seedWorkspaceModel()

	m = updateModel(m, keyRune('j'))
	if m.cursor != 1 {
		t.Errorf("after j: cursor = %d, want 1", m.cursor)
	}

	m = updateModel(m, keyRune('j'))
	if m.cursor != 2 {
		t.Errorf("after j j: cursor = %d, want 2", m.cursor)
	}

	// Clamp at bottom
	m = updateModel(m, keyRune('j'))
	if m.cursor != 2 {
		t.Errorf("after j at bottom: cursor = %d, want 2", m.cursor)
	}

	m = updateModel(m, keyRune('k'))
	if m.cursor != 1 {
		t.Errorf("after k: cursor = %d, want 1", m.cursor)
	}
}

func TestWorkspaceListBack(t *testing.T) {
	m := seedWorkspaceModel()
	m.cursor = 1

	m = updateModel(m, keyEsc())

	if m.view != viewRepoList {
		t.Errorf("view = %d, want viewRepoList", m.view)
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0 after back", m.cursor)
	}
}

func TestArchiveConfirmFlow(t *testing.T) {
	m := seedWorkspaceModel()
	m.cursor = 1

	// Press 'a' to trigger archive confirmation
	m = updateModel(m, keyRune('a'))
	if m.view != viewConfirmArchive {
		t.Errorf("view = %d, want viewConfirmArchive", m.view)
	}
	if m.archiveIdx != 1 {
		t.Errorf("archiveIdx = %d, want 1", m.archiveIdx)
	}

	// Press 'y' to confirm
	result, cmd := m.Update(keyRune('y'))
	m = result.(model)
	if !m.loading {
		t.Error("expected loading=true after confirming archive")
	}
	if m.view != viewWorkspaceList {
		t.Errorf("view = %d, want viewWorkspaceList after confirm", m.view)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for archive operation")
	}

	// Simulate archive result
	m = updateModel(m, archiveResultMsg{name: "ws-two"})
	if len(m.workspaces) != 2 {
		t.Errorf("workspaces count = %d, want 2", len(m.workspaces))
	}
	// Verify ws-two was removed
	for _, ws := range m.workspaces {
		if ws.Workspace.Name == "ws-two" {
			t.Error("ws-two should have been removed")
		}
	}
}

func TestArchiveCancel(t *testing.T) {
	m := seedWorkspaceModel()
	m.cursor = 0

	// Press 'a' then 'n'
	m = updateModel(m, keyRune('a'))
	if m.view != viewConfirmArchive {
		t.Fatalf("view = %d, want viewConfirmArchive", m.view)
	}

	m = updateModel(m, keyRune('n'))
	if m.view != viewWorkspaceList {
		t.Errorf("view = %d, want viewWorkspaceList after cancel", m.view)
	}
	if len(m.workspaces) != 3 {
		t.Errorf("workspaces count = %d, want 3 (unchanged)", len(m.workspaces))
	}
}

func TestShellRequest(t *testing.T) {
	m := seedWorkspaceModel()
	m.cursor = 1

	result, cmd := m.Update(keyRune('s'))
	m = result.(model)

	if m.shellRequest == nil {
		t.Fatal("expected shellRequest to be set")
	}
	if m.shellRequest.workspace.Name != "ws-two" {
		t.Errorf("shellRequest.workspace.Name = %q, want ws-two", m.shellRequest.workspace.Name)
	}
	if m.shellRequest.rootPath != "/a" {
		t.Errorf("shellRequest.rootPath = %q, want /a", m.shellRequest.rootPath)
	}

	// Should produce a quit command
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
	quitMsg := cmd()
	if _, ok := quitMsg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", quitMsg)
	}
}

func TestRunRequest(t *testing.T) {
	m := seedWorkspaceModel()
	m.cursor = 1

	result, cmd := m.Update(keyRune('r'))
	m = result.(model)

	if m.runRequest == nil {
		t.Fatal("expected runRequest to be set")
	}
	if m.runRequest.workspace.Name != "ws-two" {
		t.Errorf("runRequest.workspace.Name = %q, want ws-two", m.runRequest.workspace.Name)
	}
	if m.runRequest.rootPath != "/a" {
		t.Errorf("runRequest.rootPath = %q, want /a", m.runRequest.rootPath)
	}

	// Should produce a quit command
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
	quitMsg := cmd()
	if _, ok := quitMsg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", quitMsg)
	}
}

func TestBrowserRequest(t *testing.T) {
	m := seedWorkspaceModel()
	m.cursor = 1

	result, cmd := m.Update(keyRune('b'))
	m = result.(model)

	if m.browserRequest == nil {
		t.Fatal("expected browserRequest to be set")
	}
	if m.browserRequest.workspace.Name != "ws-two" {
		t.Errorf("browserRequest.workspace.Name = %q, want ws-two", m.browserRequest.workspace.Name)
	}
	if m.browserRequest.rootPath != "/a" {
		t.Errorf("browserRequest.rootPath = %q, want /a", m.browserRequest.rootPath)
	}

	// Should produce a quit command
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
	quitMsg := cmd()
	if _, ok := quitMsg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", quitMsg)
	}
}

func TestQuit(t *testing.T) {
	m := seedRepoModel()

	_, cmd := m.Update(keyRune('q'))
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
	quitMsg := cmd()
	if _, ok := quitMsg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", quitMsg)
	}

	// Also works from workspace view
	m2 := seedWorkspaceModel()
	_, cmd = m2.Update(keyRune('q'))
	if cmd == nil {
		t.Fatal("expected quit cmd from workspace view")
	}
}

func TestKeysIgnoredWhileLoading(t *testing.T) {
	m := seedRepoModel()
	m.loading = true

	// Navigation keys should be ignored while loading (except quit)
	m = updateModel(m, keyRune('j'))
	if m.cursor != 0 {
		t.Errorf("cursor moved while loading: got %d, want 0", m.cursor)
	}
}

func TestReposLoadedMsg(t *testing.T) {
	m := newModel()
	repos := []repoItem{
		{Repo: registry.Repo{Name: "test", Path: "/test"}},
	}

	m = updateModel(m, reposLoadedMsg{repos: repos})

	if m.loading {
		t.Error("expected loading=false after repos loaded")
	}
	if len(m.repos) != 1 {
		t.Errorf("repos count = %d, want 1", len(m.repos))
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

func TestWorkspacesLoadedMsg(t *testing.T) {
	m := model{view: viewRepoList, loading: true}

	workspaces := []workspaceItem{
		{Workspace: state.Workspace{Name: "ws1", Branch: "feat", Port: 3000}},
	}

	m = updateModel(m, workspacesLoadedMsg{
		workspaces: workspaces,
		repoName:   "myrepo",
		rootPath:   "/myrepo",
		commonDir:  "/myrepo/.git",
	})

	if m.view != viewWorkspaceList {
		t.Errorf("view = %d, want viewWorkspaceList", m.view)
	}
	if m.loading {
		t.Error("expected loading=false")
	}
	if m.repoName != "myrepo" {
		t.Errorf("repoName = %q, want myrepo", m.repoName)
	}
	if len(m.workspaces) != 1 {
		t.Errorf("workspaces count = %d, want 1", len(m.workspaces))
	}
}

func TestArchiveResultCursorClamp(t *testing.T) {
	m := seedWorkspaceModel()
	m.cursor = 2 // Last item

	// Remove the last workspace
	m = updateModel(m, archiveResultMsg{name: "ws-three"})
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (clamped after removing last item)", m.cursor)
	}
}
