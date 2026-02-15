package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/protocollar/fr8/internal/git"
	"github.com/protocollar/fr8/internal/opener"
	"github.com/protocollar/fr8/internal/registry"
	"github.com/protocollar/fr8/internal/state"
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
			{Workspace: state.Workspace{Name: "ws-one", Port: 3000}, Branch: "feat-1"},
			{Workspace: state.Workspace{Name: "ws-two", Port: 3010}, Branch: "feat-2"},
			{Workspace: state.Workspace{Name: "ws-three", Port: 3020}, Branch: "feat-3"},
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
		{Workspace: state.Workspace{Name: "ws1", Port: 3000}, Branch: "feat"},
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
		view:      viewWorkspaceList,
		loading:   true,
		repoName:  "alpha",
		rootPath:  "/a",
		commonDir: "/a/.git",
		repos: []repoItem{
			{Repo: registry.Repo{Name: "alpha", Path: "/a"}, WorkspaceCount: 1},
		},
		workspaces: []workspaceItem{
			{Workspace: state.Workspace{Name: "only-ws", Port: 3000}, Branch: "feat-1"},
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
		openers: []opener.Opener{{Name: "vscode", Command: "code"}},
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
		openers: []opener.Opener{
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
	m.openers = []opener.Opener{
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
	m.openers = []opener.Opener{
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
	m.openers = []opener.Opener{
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
		openers: []opener.Opener{
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
		openers: []opener.Opener{
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
