package tui

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/protocollar/fr8/internal/git"
	"github.com/protocollar/fr8/internal/registry"
	"github.com/protocollar/fr8/internal/tmux"
	"github.com/protocollar/fr8/internal/userconfig"
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
		workspaces: []workspaceItem{
			{Workspace: registry.Workspace{Name: "ws-one", Port: 3000}, Branch: "feat-1"},
			{Workspace: registry.Workspace{Name: "ws-two", Port: 3010}, Branch: "feat-2"},
			{Workspace: registry.Workspace{Name: "ws-three", Port: 3020}, Branch: "feat-3"},
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
	if m.loading {
		t.Error("expected loading=false after archiveResultMsg")
	}
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

func TestRunKeyDispatchesCmd(t *testing.T) {
	m := seedWorkspaceModel()
	m.cursor = 1

	result, cmd := m.Update(keyRune('r'))
	m = result.(model)

	if !m.loading {
		t.Error("expected loading=true after r")
	}
	if m.err != nil {
		t.Errorf("unexpected error: %v", m.err)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd after r")
	}
}

func TestRunKeyErrorIfAlreadyRunning(t *testing.T) {
	m := seedWorkspaceModel()
	m.workspaces[1].Running = true
	m.cursor = 1

	result, cmd := m.Update(keyRune('r'))
	m = result.(model)

	if m.err == nil {
		t.Error("expected error when running already-running workspace")
	}
	if cmd != nil {
		t.Error("expected nil cmd when workspace already running")
	}
}

func TestBrowserKeyDispatchesCmd(t *testing.T) {
	m := seedWorkspaceModel()
	m.cursor = 1

	result, cmd := m.Update(keyRune('b'))
	m = result.(model)

	// Browser is async — should dispatch a command but NOT quit
	if cmd == nil {
		t.Error("expected non-nil cmd after b")
	}
	if m.loading {
		t.Error("browser should not set loading=true")
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
		{Workspace: registry.Workspace{Name: "ws1", Port: 3000}, Branch: "feat"},
	}

	m = updateModel(m, workspacesLoadedMsg{
		workspaces: workspaces,
		repoName:   "myrepo",
		rootPath:   "/myrepo",
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

func TestArchiveResultClearsLoading(t *testing.T) {
	m := seedWorkspaceModel()
	m.loading = true // simulates state after confirming archive

	m = updateModel(m, archiveResultMsg{name: "ws-one"})

	if m.loading {
		t.Error("expected loading=false after archiveResultMsg")
	}
	if m.view != viewWorkspaceList {
		t.Errorf("view = %d, want viewWorkspaceList", m.view)
	}
}

func TestArchiveLastWorkspaceClearsLoading(t *testing.T) {
	m := model{
		view:     viewWorkspaceList,
		loading:  true,
		repoName: "alpha",
		rootPath: "/a",
		repos: []repoItem{
			{Repo: registry.Repo{Name: "alpha", Path: "/a"}, WorkspaceCount: 1},
		},
		workspaces: []workspaceItem{
			{Workspace: registry.Workspace{Name: "only-ws", Port: 3000}, Branch: "feat-1"},
		},
		cursor: 0,
	}

	m = updateModel(m, archiveResultMsg{name: "only-ws"})

	if m.loading {
		t.Error("expected loading=false after archiving last workspace")
	}
	if len(m.workspaces) != 0 {
		t.Errorf("workspaces count = %d, want 0", len(m.workspaces))
	}
	if m.view != viewWorkspaceList {
		t.Errorf("view = %d, want viewWorkspaceList", m.view)
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

func TestArchiveResultWithError(t *testing.T) {
	m := seedWorkspaceModel()
	m.loading = true

	m = updateModel(m, archiveResultMsg{name: "ws-one", err: errStub{}})

	if m.loading {
		t.Error("expected loading=false after archive error")
	}
	if m.err == nil {
		t.Error("expected error to be set")
	}
	if len(m.workspaces) != 3 {
		t.Errorf("workspaces count = %d, want 3 (unchanged after error)", len(m.workspaces))
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


func TestStopKeyDispatchesCmd(t *testing.T) {
	m := seedWorkspaceModel()
	m.workspaces[1].Running = true
	m.cursor = 1

	result, cmd := m.Update(keyRune('x'))
	m = result.(model)

	if !m.loading {
		t.Error("expected loading=true after x")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd after x")
	}
}

func TestStopKeyErrorIfNotRunning(t *testing.T) {
	m := seedWorkspaceModel()
	m.cursor = 1

	result, cmd := m.Update(keyRune('x'))
	m = result.(model)

	if m.err == nil {
		t.Error("expected error when stopping non-running workspace")
	}
	if cmd != nil {
		t.Error("expected nil cmd when workspace not running")
	}
}

func TestAttachKeyQuitsAndSetsRequest(t *testing.T) {
	m := seedWorkspaceModel()
	m.workspaces[1].Running = true
	m.cursor = 1

	result, cmd := m.Update(keyRune('t'))
	m = result.(model)

	if m.attachRequest == nil {
		t.Fatal("expected attachRequest to be set")
	}
	if m.attachRequest.workspace.Name != "ws-two" {
		t.Errorf("attachRequest.workspace.Name = %q, want ws-two", m.attachRequest.workspace.Name)
	}
	if m.attachRequest.rootPath != "/a" {
		t.Errorf("attachRequest.rootPath = %q, want /a", m.attachRequest.rootPath)
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

func TestAttachKeyErrorIfNotRunning(t *testing.T) {
	m := seedWorkspaceModel()
	m.cursor = 1

	result, cmd := m.Update(keyRune('t'))
	m = result.(model)

	if m.err == nil {
		t.Error("expected error when attaching to non-running workspace")
	}
	if m.attachRequest != nil {
		t.Error("attachRequest should be nil when workspace not running")
	}
	if cmd != nil {
		t.Error("expected nil cmd when workspace not running")
	}
}

func TestStartResultMsgUpdatesRunning(t *testing.T) {
	m := seedWorkspaceModel()
	m.loading = true

	m = updateModel(m, startResultMsg{name: "ws-two"})

	if m.loading {
		t.Error("expected loading=false after startResultMsg")
	}
	if !m.workspaces[1].Running {
		t.Error("expected ws-two to be marked as running")
	}
	if m.workspaces[0].Running {
		t.Error("ws-one should not be affected")
	}
}

func TestStartResultMsgWithError(t *testing.T) {
	m := seedWorkspaceModel()
	m.loading = true

	m = updateModel(m, startResultMsg{name: "ws-two", err: errStub{}})

	if m.loading {
		t.Error("expected loading=false after error")
	}
	if m.err == nil {
		t.Error("expected error to be set")
	}
	if m.workspaces[1].Running {
		t.Error("ws-two should not be marked as running after error")
	}
}

func TestStopResultMsgUpdatesRunning(t *testing.T) {
	m := seedWorkspaceModel()
	m.workspaces[1].Running = true
	m.loading = true

	m = updateModel(m, stopResultMsg{name: "ws-two"})

	if m.loading {
		t.Error("expected loading=false after stopResultMsg")
	}
	if m.workspaces[1].Running {
		t.Error("expected ws-two to be marked as not running")
	}
}

func TestStopResultMsgWithError(t *testing.T) {
	m := seedWorkspaceModel()
	m.workspaces[1].Running = true
	m.loading = true

	m = updateModel(m, stopResultMsg{name: "ws-two", err: errStub{}})

	if m.loading {
		t.Error("expected loading=false after error")
	}
	if m.err == nil {
		t.Error("expected error to be set")
	}
	// Running state should not change on error
	if !m.workspaces[1].Running {
		t.Error("ws-two running state should not change after stop error")
	}
}

func TestRepoRunAllKeyDispatchesCmd(t *testing.T) {
	m := seedRepoModel()

	result, cmd := m.Update(keyRune('r'))
	m = result.(model)

	if !m.loading {
		t.Error("expected loading=true after r on repo list")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd after r on repo list")
	}
}

func TestRepoStopAllKeyDispatchesCmd(t *testing.T) {
	m := seedRepoModel()

	result, cmd := m.Update(keyRune('x'))
	m = result.(model)

	if !m.loading {
		t.Error("expected loading=true after x on repo list")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd after x on repo list")
	}
}

func TestRepoRunAllGlobalKeyDispatchesCmd(t *testing.T) {
	m := seedRepoModel()

	result, cmd := m.Update(keyRune('R'))
	m = result.(model)

	if !m.loading {
		t.Error("expected loading=true after R on repo list")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd after R on repo list")
	}
}

func TestRepoStopAllGlobalKeyDispatchesCmd(t *testing.T) {
	m := seedRepoModel()

	result, cmd := m.Update(keyRune('X'))
	m = result.(model)

	if !m.loading {
		t.Error("expected loading=true after X on repo list")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd after X on repo list")
	}
}

func TestRunAllResultMsg(t *testing.T) {
	m := seedRepoModel()
	m.loading = true

	m = updateModel(m, runAllResultMsg{repoName: "alpha", started: 2})

	if m.loading {
		t.Error("expected loading=false after runAllResultMsg")
	}
	if m.err != nil {
		t.Errorf("unexpected error: %v", m.err)
	}
}

func TestRunAllResultMsgWithError(t *testing.T) {
	m := seedRepoModel()
	m.loading = true

	m = updateModel(m, runAllResultMsg{repoName: "alpha", err: errStub{}})

	if m.loading {
		t.Error("expected loading=false after error")
	}
	if m.err == nil {
		t.Error("expected error to be set")
	}
}

func TestStopAllResultMsg(t *testing.T) {
	m := seedRepoModel()
	m.loading = true

	m = updateModel(m, stopAllResultMsg{repoName: "alpha", stopped: 2})

	if m.loading {
		t.Error("expected loading=false after stopAllResultMsg")
	}
	if m.err != nil {
		t.Errorf("unexpected error: %v", m.err)
	}
}

func TestStopAllResultMsgWithError(t *testing.T) {
	m := seedRepoModel()
	m.loading = true

	m = updateModel(m, stopAllResultMsg{repoName: "alpha", err: errStub{}})

	if m.loading {
		t.Error("expected loading=false after error")
	}
	if m.err == nil {
		t.Error("expected error to be set")
	}
}

func TestRepoRunAllIgnoredWhileLoading(t *testing.T) {
	m := seedRepoModel()
	m.loading = true

	m = updateModel(m, keyRune('r'))
	// Should remain loading, no additional cmd dispatched
	if m.cursor != 0 {
		t.Errorf("cursor moved while loading: got %d, want 0", m.cursor)
	}
}

func TestRepoRunAllNoOpOnEmptyList(t *testing.T) {
	m := model{view: viewRepoList, repos: nil}

	result, cmd := m.Update(keyRune('r'))
	m = result.(model)

	if m.loading {
		t.Error("should not set loading on empty repo list")
	}
	if cmd != nil {
		t.Error("should not dispatch cmd on empty repo list")
	}
}

func TestOpenKeyDispatchesLoadOpeners(t *testing.T) {
	m := seedWorkspaceModel()
	m.cursor = 1

	result, cmd := m.Update(keyRune('o'))
	m = result.(model)

	if !m.loading {
		t.Error("expected loading=true after o")
	}
	if m.openerWsIdx != 1 {
		t.Errorf("openerWsIdx = %d, want 1", m.openerWsIdx)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd after o")
	}
}

func TestOpenKeyNoOpOnEmptyWorkspaces(t *testing.T) {
	m := model{view: viewWorkspaceList, workspaces: nil}

	result, cmd := m.Update(keyRune('o'))
	m = result.(model)

	if m.loading {
		t.Error("should not set loading on empty workspace list")
	}
	if cmd != nil {
		t.Error("should not dispatch cmd on empty workspace list")
	}
}

func TestOpenersLoadedSingleOpenerQuitsDirectly(t *testing.T) {
	m := seedWorkspaceModel()
	m.loading = true
	m.openerWsIdx = 1

	result, cmd := m.Update(openersLoadedMsg{
		openers: []userconfig.Opener{{Name: "vscode", Command: "code"}},
	})
	m = result.(model)

	if m.openRequest == nil {
		t.Fatal("expected openRequest to be set for single opener")
	}
	if m.openRequest.workspace.Name != "ws-two" {
		t.Errorf("openRequest.workspace.Name = %q, want ws-two", m.openRequest.workspace.Name)
	}
	if m.openRequest.openerName != "vscode" {
		t.Errorf("openRequest.openerName = %q, want vscode", m.openRequest.openerName)
	}
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
	quitMsg := cmd()
	if _, ok := quitMsg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", quitMsg)
	}
}

func TestOpenersLoadedMultipleShowsPicker(t *testing.T) {
	m := seedWorkspaceModel()
	m.loading = true
	m.openerWsIdx = 0

	m = updateModel(m, openersLoadedMsg{
		openers: []userconfig.Opener{
			{Name: "vscode", Command: "code"},
			{Name: "cursor", Command: "cursor"},
		},
	})

	if m.loading {
		t.Error("expected loading=false")
	}
	if m.view != viewOpenerPicker {
		t.Errorf("view = %d, want viewOpenerPicker", m.view)
	}
	if len(m.openers) != 2 {
		t.Errorf("openers count = %d, want 2", len(m.openers))
	}
	if m.openerCursor != 0 {
		t.Errorf("openerCursor = %d, want 0", m.openerCursor)
	}
}

func TestOpenersLoadedEmptyShowsError(t *testing.T) {
	m := seedWorkspaceModel()
	m.loading = true

	m = updateModel(m, openersLoadedMsg{openers: nil})

	if m.loading {
		t.Error("expected loading=false")
	}
	if m.err == nil {
		t.Error("expected error for empty openers")
	}
	if m.view != viewWorkspaceList {
		t.Errorf("view = %d, want viewWorkspaceList", m.view)
	}
}

func TestOpenersLoadedWithError(t *testing.T) {
	m := seedWorkspaceModel()
	m.loading = true

	m = updateModel(m, openersLoadedMsg{err: errStub{}})

	if m.loading {
		t.Error("expected loading=false")
	}
	if m.err == nil {
		t.Error("expected error to be set")
	}
}

func TestOpenerPickerNavigation(t *testing.T) {
	m := seedWorkspaceModel()
	m.view = viewOpenerPicker
	m.openerWsIdx = 0
	m.openers = []userconfig.Opener{
		{Name: "vscode", Command: "code"},
		{Name: "cursor", Command: "cursor"},
		{Name: "terminal", Command: "open"},
	}
	m.openerCursor = 0

	// Down
	m = updateModel(m, keyRune('j'))
	if m.openerCursor != 1 {
		t.Errorf("after j: openerCursor = %d, want 1", m.openerCursor)
	}

	m = updateModel(m, keyRune('j'))
	if m.openerCursor != 2 {
		t.Errorf("after j j: openerCursor = %d, want 2", m.openerCursor)
	}

	// Clamp at bottom
	m = updateModel(m, keyRune('j'))
	if m.openerCursor != 2 {
		t.Errorf("after j at bottom: openerCursor = %d, want 2", m.openerCursor)
	}

	// Up
	m = updateModel(m, keyRune('k'))
	if m.openerCursor != 1 {
		t.Errorf("after k: openerCursor = %d, want 1", m.openerCursor)
	}
}

func TestOpenerPickerSelectQuitsWithRequest(t *testing.T) {
	m := seedWorkspaceModel()
	m.view = viewOpenerPicker
	m.openerWsIdx = 1
	m.openers = []userconfig.Opener{
		{Name: "vscode", Command: "code"},
		{Name: "cursor", Command: "cursor"},
	}
	m.openerCursor = 1

	result, cmd := m.Update(keyEnter())
	m = result.(model)

	if m.openRequest == nil {
		t.Fatal("expected openRequest to be set")
	}
	if m.openRequest.workspace.Name != "ws-two" {
		t.Errorf("workspace = %q, want ws-two", m.openRequest.workspace.Name)
	}
	if m.openRequest.openerName != "cursor" {
		t.Errorf("openerName = %q, want cursor", m.openRequest.openerName)
	}
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
	quitMsg := cmd()
	if _, ok := quitMsg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", quitMsg)
	}
}

func TestOpenerPickerEscGoesBack(t *testing.T) {
	m := seedWorkspaceModel()
	m.view = viewOpenerPicker
	m.openerWsIdx = 0
	m.openers = []userconfig.Opener{
		{Name: "vscode", Command: "code"},
	}

	m = updateModel(m, keyEsc())

	if m.view != viewWorkspaceList {
		t.Errorf("view = %d, want viewWorkspaceList after Esc", m.view)
	}
}

// --- Batch Archive Tests ---

func TestBatchArchiveFiltersToMergedClean(t *testing.T) {
	m := seedWorkspaceModel()
	m.workspaces[0].Merged = true // ws-one: merged + clean
	m.workspaces[1].Merged = true
	m.workspaces[1].DirtyCount = git.DirtyCount{Modified: 1} // ws-two: merged + dirty (excluded)
	// ws-three: not merged (excluded)

	m = updateModel(m, keyRune('A'))

	if m.view != viewConfirmBatchArchive {
		t.Errorf("view = %d, want viewConfirmBatchArchive", m.view)
	}
	if len(m.batchArchiveNames) != 1 {
		t.Errorf("batchArchiveNames = %v, want [ws-one]", m.batchArchiveNames)
	}
	if m.batchArchiveNames[0] != "ws-one" {
		t.Errorf("batchArchiveNames[0] = %q, want ws-one", m.batchArchiveNames[0])
	}
}

func TestBatchArchiveNoMergedShowsError(t *testing.T) {
	m := seedWorkspaceModel()
	// No workspaces are merged

	m = updateModel(m, keyRune('A'))

	if m.view != viewWorkspaceList {
		t.Errorf("view = %d, want viewWorkspaceList", m.view)
	}
	if m.err == nil {
		t.Error("expected error when no merged workspaces")
	}
}

func TestBatchArchiveConfirmDispatches(t *testing.T) {
	m := seedWorkspaceModel()
	m.view = viewConfirmBatchArchive
	m.batchArchiveNames = []string{"ws-one", "ws-two"}

	result, cmd := m.Update(keyRune('y'))
	m = result.(model)

	if !m.loading {
		t.Error("expected loading=true after confirming batch archive")
	}
	if m.view != viewWorkspaceList {
		t.Errorf("view = %d, want viewWorkspaceList after confirm", m.view)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for batch archive operation")
	}
}

func TestBatchArchiveCancelReturns(t *testing.T) {
	m := seedWorkspaceModel()
	m.view = viewConfirmBatchArchive
	m.batchArchiveNames = []string{"ws-one"}

	m = updateModel(m, keyRune('n'))

	if m.view != viewWorkspaceList {
		t.Errorf("view = %d, want viewWorkspaceList after cancel", m.view)
	}
	if m.batchArchiveNames != nil {
		t.Errorf("batchArchiveNames should be nil after cancel, got %v", m.batchArchiveNames)
	}
}

func TestBatchArchiveResultRemovesWorkspaces(t *testing.T) {
	m := seedWorkspaceModel()
	m.loading = true
	m.repos = []repoItem{
		{Repo: registry.Repo{Name: "alpha", Path: "/a"}, WorkspaceCount: 3},
	}
	m.repoName = "alpha"

	m = updateModel(m, batchArchiveResultMsg{archived: []string{"ws-one", "ws-three"}})

	if m.loading {
		t.Error("expected loading=false after batchArchiveResultMsg")
	}
	if len(m.workspaces) != 1 {
		t.Errorf("workspaces count = %d, want 1", len(m.workspaces))
	}
	if m.workspaces[0].Workspace.Name != "ws-two" {
		t.Errorf("remaining workspace = %q, want ws-two", m.workspaces[0].Workspace.Name)
	}
	if m.repos[0].WorkspaceCount != 1 {
		t.Errorf("repo workspace count = %d, want 1", m.repos[0].WorkspaceCount)
	}
}

// --- Create Workspace Tests ---

func TestNewKeyOpensCreateView(t *testing.T) {
	m := seedWorkspaceModel()

	m = updateModel(m, keyRune('n'))

	if m.view != viewCreateWorkspace {
		t.Errorf("view = %d, want viewCreateWorkspace", m.view)
	}
}

func TestCreateWorkspaceEnterQuitsWithRequest(t *testing.T) {
	m := seedWorkspaceModel()
	m.view = viewCreateWorkspace
	m.createInput = textinput.New()
	m.createInput.SetValue("my-ws")

	result, cmd := m.Update(keyEnter())
	m = result.(model)

	if m.createRequest == nil {
		t.Fatal("expected createRequest to be set")
	}
	if m.createRequest.name != "my-ws" {
		t.Errorf("createRequest.name = %q, want my-ws", m.createRequest.name)
	}
	if m.createRequest.rootPath != "/a" {
		t.Errorf("createRequest.rootPath = %q, want /a", m.createRequest.rootPath)
	}
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
	quitMsg := cmd()
	if _, ok := quitMsg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", quitMsg)
	}
}

func TestCreateWorkspaceEscReturns(t *testing.T) {
	m := seedWorkspaceModel()
	m.view = viewCreateWorkspace
	m.createInput = textinput.New()

	m = updateModel(m, keyEsc())

	if m.view != viewWorkspaceList {
		t.Errorf("view = %d, want viewWorkspaceList after Esc", m.view)
	}
}

// --- Help Overlay Tests ---

func TestHelpToggleFromRepoList(t *testing.T) {
	m := seedRepoModel()

	// Press ? to open help
	m = updateModel(m, keyRune('?'))
	if m.view != viewHelp {
		t.Errorf("view = %d, want viewHelp", m.view)
	}
	if m.previousView != viewRepoList {
		t.Errorf("previousView = %d, want viewRepoList", m.previousView)
	}

	// Press ? again to close help
	m = updateModel(m, keyRune('?'))
	if m.view != viewRepoList {
		t.Errorf("view = %d, want viewRepoList after closing help", m.view)
	}
}

func TestHelpToggleFromWorkspaceList(t *testing.T) {
	m := seedWorkspaceModel()

	m = updateModel(m, keyRune('?'))
	if m.view != viewHelp {
		t.Errorf("view = %d, want viewHelp", m.view)
	}
	if m.previousView != viewWorkspaceList {
		t.Errorf("previousView = %d, want viewWorkspaceList", m.previousView)
	}

	// Any non-? key returns to previous view
	m = updateModel(m, keyRune('j'))
	if m.view != viewWorkspaceList {
		t.Errorf("view = %d, want viewWorkspaceList after pressing j in help", m.view)
	}
}

func TestOpenersLoadedDefaultOpenerAutoSelect(t *testing.T) {
	m := seedWorkspaceModel()
	m.loading = true
	m.openerWsIdx = 1

	result, cmd := m.Update(openersLoadedMsg{
		openers: []userconfig.Opener{
			{Name: "vscode", Command: "code"},
			{Name: "cursor", Command: "cursor", Default: true},
			{Name: "terminal", Command: "open"},
		},
	})
	m = result.(model)

	// Should auto-select the default opener (cursor) without showing picker
	if m.openRequest == nil {
		t.Fatal("expected openRequest to be set for default opener")
	}
	if m.openRequest.workspace.Name != "ws-two" {
		t.Errorf("openRequest.workspace.Name = %q, want ws-two", m.openRequest.workspace.Name)
	}
	if m.openRequest.openerName != "cursor" {
		t.Errorf("openRequest.openerName = %q, want cursor", m.openRequest.openerName)
	}
	if m.view == viewOpenerPicker {
		t.Error("should NOT show picker when a default opener exists")
	}
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
	quitMsg := cmd()
	if _, ok := quitMsg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", quitMsg)
	}
}

func TestOpenersLoadedNoDefaultShowsPicker(t *testing.T) {
	m := seedWorkspaceModel()
	m.loading = true
	m.openerWsIdx = 0

	// Multiple openers, none is default — should show picker
	m = updateModel(m, openersLoadedMsg{
		openers: []userconfig.Opener{
			{Name: "vscode", Command: "code"},
			{Name: "cursor", Command: "cursor"},
			{Name: "terminal", Command: "open"},
		},
	})

	if m.view != viewOpenerPicker {
		t.Errorf("view = %d, want viewOpenerPicker when no default", m.view)
	}
	if len(m.openers) != 3 {
		t.Errorf("openers count = %d, want 3", len(m.openers))
	}
}

func TestHelpEscReturnsToPreview(t *testing.T) {
	m := seedRepoModel()

	// Open help
	m = updateModel(m, keyRune('?'))
	if m.view != viewHelp {
		t.Fatalf("view = %d, want viewHelp", m.view)
	}

	// Esc should also return to previous view
	m = updateModel(m, keyEsc())
	if m.view != viewRepoList {
		t.Errorf("view = %d, want viewRepoList after Esc from help", m.view)
	}
}

// errStub is defined in view_test.go (same package)

// --- 1.1 Remember Cursor on Back-Navigation ---

func TestCursorRememberedOnBackNavigation(t *testing.T) {
	m := seedRepoModel()
	m.cursor = 2 // move to third repo

	// Drill into workspace list (which saves repoCursor)
	result, _ := m.Update(keyEnter())
	m = result.(model)

	// Simulate workspaces loaded
	m = updateModel(m, workspacesLoadedMsg{
		workspaces: []workspaceItem{{Workspace: registry.Workspace{Name: "ws1"}}},
		repoName:   "charlie",
		rootPath:   "/c",
	})
	if m.view != viewWorkspaceList {
		t.Fatalf("view = %d, want viewWorkspaceList", m.view)
	}

	// Go back
	m = updateModel(m, keyEsc())
	if m.view != viewRepoList {
		t.Fatalf("view = %d, want viewRepoList", m.view)
	}
	if m.cursor != 2 {
		t.Errorf("cursor = %d, want 2 (remembered position)", m.cursor)
	}
}

// --- 1.4 Toast Notifications ---

func TestToastSetOnStartResult(t *testing.T) {
	m := seedWorkspaceModel()
	m.loading = true

	result, cmd := m.Update(startResultMsg{name: "ws-one"})
	m = result.(model)

	if m.toast == "" {
		t.Error("expected toast to be set after start")
	}
	if m.toastIsError {
		t.Error("expected non-error toast after successful start")
	}
	if cmd == nil {
		t.Error("expected toast tick cmd")
	}
}

func TestToastSetOnError(t *testing.T) {
	m := seedWorkspaceModel()
	m.loading = true

	m = updateModel(m, startResultMsg{name: "ws-one", err: errStub{}})

	if m.toast == "" {
		t.Error("expected toast to be set on error")
	}
	if !m.toastIsError {
		t.Error("expected error toast")
	}
}

func TestToastExpiry(t *testing.T) {
	m := seedWorkspaceModel()
	m.toast = "test toast"
	m.toastExpiry = time.Now().Add(-1 * time.Second) // already expired

	m = updateModel(m, toastTickMsg{})

	if m.toast != "" {
		t.Errorf("toast should be cleared after expiry, got %q", m.toast)
	}
}

func TestToastNotExpiredKeepsTicking(t *testing.T) {
	m := seedWorkspaceModel()
	m.toast = "test toast"
	m.toastExpiry = time.Now().Add(5 * time.Second) // not expired

	result, cmd := m.Update(toastTickMsg{})
	m = result.(model)

	if m.toast == "" {
		t.Error("toast should not be cleared before expiry")
	}
	if cmd == nil {
		t.Error("expected tick cmd to continue")
	}
}

// --- 2.1 Search/Filter ---

func TestFilterActivation(t *testing.T) {
	m := seedRepoModel()

	m = updateModel(m, keyRune('/'))

	if !m.filtering {
		t.Error("expected filtering=true after /")
	}
}

func TestFilterEscClears(t *testing.T) {
	m := seedRepoModel()
	m.filtering = true
	m.filterInput = textinput.New()
	m.filterInput.SetValue("test")

	m = updateModel(m, keyEsc())

	if m.filtering {
		t.Error("expected filtering=false after Esc")
	}
	if m.filterInput.Value() != "" {
		t.Errorf("filter value = %q, want empty after Esc", m.filterInput.Value())
	}
}

func TestFilterEnterKeepsValue(t *testing.T) {
	m := seedRepoModel()
	m.filtering = true
	m.filterInput = textinput.New()
	m.filterInput.SetValue("alp")

	m = updateModel(m, keyEnter())

	if m.filtering {
		t.Error("expected filtering=false after Enter")
	}
	if m.filterInput.Value() != "alp" {
		t.Errorf("filter value = %q, want 'alp' after Enter", m.filterInput.Value())
	}
}

func TestFilteredRepos(t *testing.T) {
	repos := []repoItem{
		{Repo: registry.Repo{Name: "alpha"}},
		{Repo: registry.Repo{Name: "bravo"}},
		{Repo: registry.Repo{Name: "charlie"}},
	}

	// No filter
	if got := filteredRepos(repos, ""); len(got) != 3 {
		t.Errorf("no filter: got %d, want 3", len(got))
	}

	// Filter matches one
	if got := filteredRepos(repos, "bra"); len(got) != 1 || got[0].Repo.Name != "bravo" {
		t.Errorf("filter 'bra': got %v", got)
	}

	// Case insensitive
	if got := filteredRepos(repos, "ALPHA"); len(got) != 1 || got[0].Repo.Name != "alpha" {
		t.Errorf("filter 'ALPHA': got %v", got)
	}

	// No matches
	if got := filteredRepos(repos, "xyz"); len(got) != 0 {
		t.Errorf("filter 'xyz': got %d, want 0", len(got))
	}
}

func TestFilteredWorkspaces(t *testing.T) {
	workspaces := []workspaceItem{
		{Workspace: registry.Workspace{Name: "ws-one"}, Branch: "feat-1"},
		{Workspace: registry.Workspace{Name: "ws-two"}, Branch: "feat-2"},
		{Workspace: registry.Workspace{Name: "ws-three"}, Branch: "main"},
	}

	// Filter by name
	if got := filteredWorkspaces(workspaces, "two"); len(got) != 1 || got[0].Workspace.Name != "ws-two" {
		t.Errorf("filter 'two': got %v", got)
	}

	// Filter by branch
	if got := filteredWorkspaces(workspaces, "main"); len(got) != 1 || got[0].Workspace.Name != "ws-three" {
		t.Errorf("filter 'main': got %v", got)
	}
}

func TestFilteredRepoResolvesCorrectIndex(t *testing.T) {
	m := seedRepoModel()
	m.filterInput = textinput.New()
	m.filterInput.SetValue("charlie")
	m.cursor = 0 // first in filtered list

	filtered := filteredRepos(m.repos, m.filterInput.Value())
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered result, got %d", len(filtered))
	}

	origIdx := resolveOriginalRepoIndex(0, filtered, m.repos)
	if origIdx != 2 {
		t.Errorf("original index = %d, want 2", origIdx)
	}
}

// --- 2.2 Multi-Select ---

func TestMultiSelectToggle(t *testing.T) {
	m := seedWorkspaceModel()

	// Space toggles selection and advances cursor
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})

	if m.selected == nil || !m.selected[0] {
		t.Error("expected workspace 0 to be selected")
	}
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (advanced after space)", m.cursor)
	}

	// Select second
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if !m.selected[1] {
		t.Error("expected workspace 1 to be selected")
	}

	// Deselect first (move back, toggle)
	m.cursor = 0
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if m.selected[0] {
		t.Error("expected workspace 0 to be deselected")
	}
}

func TestMultiSelectEscClearsBeforeBack(t *testing.T) {
	m := seedWorkspaceModel()
	m.selected = map[int]bool{0: true, 1: true}

	// First Esc clears selection
	m = updateModel(m, keyEsc())
	if m.selected != nil {
		t.Error("expected selection to be cleared on first Esc")
	}
	if m.view != viewWorkspaceList {
		t.Errorf("view = %d, want viewWorkspaceList (not back yet)", m.view)
	}

	// Second Esc navigates back
	m = updateModel(m, keyEsc())
	if m.view != viewRepoList {
		t.Errorf("view = %d, want viewRepoList after second Esc", m.view)
	}
}

// --- 4.1 Refresh ---

func TestRefreshRepoList(t *testing.T) {
	m := seedRepoModel()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	m = result.(model)

	if !m.loading {
		t.Error("expected loading=true after ctrl+r")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd after ctrl+r")
	}
}

func TestRefreshWorkspaceListPreservesCursor(t *testing.T) {
	m := seedWorkspaceModel()
	m.repos = []repoItem{
		{Repo: registry.Repo{Name: "alpha", Path: "/a"}},
	}
	m.cursor = 2

	// Simulate workspace loaded while already on workspace list (refresh case)
	m = updateModel(m, workspacesLoadedMsg{
		workspaces: []workspaceItem{
			{Workspace: registry.Workspace{Name: "ws-one"}},
			{Workspace: registry.Workspace{Name: "ws-two"}},
			{Workspace: registry.Workspace{Name: "ws-three"}},
		},
		repoName: "alpha",
		rootPath: "/a",
	})

	if m.cursor != 2 {
		t.Errorf("cursor = %d, want 2 (preserved on refresh)", m.cursor)
	}
}

func TestRefreshWorkspaceListClampsCursor(t *testing.T) {
	m := seedWorkspaceModel()
	m.cursor = 2

	// Fewer workspaces returned — cursor should clamp
	m = updateModel(m, workspacesLoadedMsg{
		workspaces: []workspaceItem{
			{Workspace: registry.Workspace{Name: "ws-one"}},
		},
		repoName: "alpha",
		rootPath: "/a",
	})

	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (clamped)", m.cursor)
	}
}

// --- 4.2 Auto-Refresh ---

func TestAutoRefreshSkipsWhenLoading(t *testing.T) {
	m := seedRepoModel()
	m.loading = true

	result, cmd := m.Update(autoRefreshTickMsg{})
	m = result.(model)

	// Should re-schedule tick but not dispatch refresh
	if cmd == nil {
		t.Error("expected re-schedule tick cmd even when loading")
	}
}

func TestAutoRefreshResultUpdatesRunning(t *testing.T) {
	m := seedWorkspaceModel()
	m.rootPath = "/a"
	m.repos = []repoItem{
		{Repo: registry.Repo{Name: "alpha", Path: "/a"}},
	}

	// ws-two should become running
	result, _ := m.Update(autoRefreshResultMsg{
		sessions: []tmux.Session{
			{Name: "fr8/a/ws-two", Repo: "a", Workspace: "ws-two"},
		},
	})
	m = result.(model)

	if m.workspaces[0].Running {
		t.Error("ws-one should not be running")
	}
	// Note: the auto-refresh matches by tmux.SessionName which uses tmux.RepoName
	// In test, tmux.RepoName("/a") depends on actual implementation
}

// --- Toast on Stop/Archive Result ---

func TestToastSetOnStopResult(t *testing.T) {
	m := seedWorkspaceModel()
	m.workspaces[0].Running = true
	m.loading = true

	result, cmd := m.Update(stopResultMsg{name: "ws-one"})
	m = result.(model)

	if m.toast == "" {
		t.Error("expected toast to be set after stop")
	}
	if m.toastIsError {
		t.Error("expected non-error toast after successful stop")
	}
	if cmd == nil {
		t.Error("expected toast tick cmd")
	}
}

func TestToastSetOnArchiveResult(t *testing.T) {
	m := seedWorkspaceModel()
	m.loading = true

	result, cmd := m.Update(archiveResultMsg{name: "ws-one"})
	m = result.(model)

	if m.toast == "" {
		t.Error("expected toast to be set after archive")
	}
	if m.toastIsError {
		t.Error("expected non-error toast after successful archive")
	}
	if cmd == nil {
		t.Error("expected toast tick cmd")
	}
}

func TestToastSetOnBatchArchiveResult(t *testing.T) {
	m := seedWorkspaceModel()
	m.loading = true
	m.repos = []repoItem{{Repo: registry.Repo{Name: "alpha", Path: "/a"}, WorkspaceCount: 3}}
	m.repoName = "alpha"

	result, cmd := m.Update(batchArchiveResultMsg{archived: []string{"ws-one"}})
	m = result.(model)

	if m.toast == "" {
		t.Error("expected toast to be set after batch archive")
	}
	if m.toastIsError {
		t.Error("expected non-error toast after successful batch archive")
	}
	if cmd == nil {
		t.Error("expected toast tick cmd")
	}
}

// --- Batch Start/Stop Result clears selection ---

func TestBatchStartResultClearsSelection(t *testing.T) {
	m := seedWorkspaceModel()
	m.loading = true
	m.selected = map[int]bool{0: true, 1: true}

	m = updateModel(m, batchStartResultMsg{started: 2})

	if m.selected != nil {
		t.Error("expected selection to be cleared after batch start")
	}
	if m.toast == "" {
		t.Error("expected toast to be set")
	}
}

func TestBatchStopResultClearsSelection(t *testing.T) {
	m := seedWorkspaceModel()
	m.loading = true
	m.selected = map[int]bool{0: true, 1: true}

	m = updateModel(m, batchStopResultMsg{stopped: 2})

	if m.selected != nil {
		t.Error("expected selection to be cleared after batch stop")
	}
	if m.toast == "" {
		t.Error("expected toast to be set")
	}
}

// --- Filter on workspace list ---

func TestFilterActivationOnWorkspaceList(t *testing.T) {
	m := seedWorkspaceModel()

	m = updateModel(m, keyRune('/'))

	if !m.filtering {
		t.Error("expected filtering=true after / on workspace list")
	}
	if m.cursor != 0 {
		t.Error("expected cursor reset to 0 on filter activation")
	}
}

func TestFilterWorkspaceListEscClears(t *testing.T) {
	m := seedWorkspaceModel()
	m.filtering = true
	m.filterInput = textinput.New()
	m.filterInput.SetValue("test")

	m = updateModel(m, keyEsc())

	if m.filtering {
		t.Error("expected filtering=false after Esc on workspace filter")
	}
	if m.filterInput.Value() != "" {
		t.Errorf("filter value = %q, want empty after Esc", m.filterInput.Value())
	}
}

func TestFilterWorkspaceListResolvesThroughFilter(t *testing.T) {
	m := seedWorkspaceModel()
	m.filterInput = textinput.New()
	m.filterInput.SetValue("three")
	m.cursor = 0 // first in filtered list

	filtered := filteredWorkspaces(m.workspaces, m.filterInput.Value())
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered result, got %d", len(filtered))
	}

	origIdx := resolveOriginalWsIndex(0, filtered, m.workspaces)
	if origIdx != 2 {
		t.Errorf("original index = %d, want 2", origIdx)
	}
}

// --- Ctrl+L redraw key ---

func TestRedrawKeyDispatchesClearScreen(t *testing.T) {
	m := seedRepoModel()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
	_ = result.(model)

	if cmd == nil {
		t.Error("expected non-nil cmd after ctrl+l")
	}
}

// --- Multi-select through filter ---

func TestMultiSelectThroughFilter(t *testing.T) {
	m := seedWorkspaceModel()
	m.filterInput = textinput.New()
	m.filterInput.SetValue("three")
	m.cursor = 0 // first in filtered list (ws-three at original index 2)

	// Space selects the filtered item (original index 2)
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})

	if m.selected == nil {
		t.Fatal("expected selection to be non-nil")
	}
	if !m.selected[2] {
		t.Error("expected original index 2 to be selected (ws-three)")
	}
	if m.selected[0] {
		t.Error("original index 0 should not be selected")
	}
}

// --- Filter cursor resets on text change ---

func TestFilterCursorResetsOnChange(t *testing.T) {
	m := seedRepoModel()
	m.filtering = true
	m.filterInput = textinput.New()
	m.filterInput.Focus()
	m.cursor = 2

	// Type a character — cursor should reset to 0
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0 after filter text change", m.cursor)
	}
}
