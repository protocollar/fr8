package tui

import (
	"github.com/thomascarr/fr8/internal/opener"
	"github.com/thomascarr/fr8/internal/registry"
	"github.com/thomascarr/fr8/internal/state"
)

type viewState int

const (
	viewRepoList viewState = iota
	viewWorkspaceList
	viewConfirmArchive
	viewConfirmBatchArchive
	viewOpenerPicker
	viewCreateWorkspace
)

// repoItem is a repo with preloaded workspace count.
type repoItem struct {
	Repo           registry.Repo
	WorkspaceCount int
	RunningCount   int
	Err            error
}

// workspaceItem is a workspace with live git status.
type workspaceItem struct {
	Workspace state.Workspace
	Dirty     bool
	Merged    bool
	Ahead     int
	Behind    int
	PortFree  bool // true when nothing is listening on the workspace port
	Running   bool // true when a tmux session is active for this workspace
	StatusErr error
}

// Messages for async operations.

type reposLoadedMsg struct {
	repos []repoItem
	err   error
}

type workspacesLoadedMsg struct {
	workspaces []workspaceItem
	repoName   string
	rootPath   string
	commonDir  string
	err        error
}

type archiveResultMsg struct {
	name string
	err  error
}

type shellRequestMsg struct {
	workspace state.Workspace
	rootPath  string
}

type attachRequestMsg struct {
	workspace state.Workspace
	rootPath  string
}

type openRequestMsg struct {
	workspace  state.Workspace
	openerName string
}

type startResultMsg struct {
	name string
	err  error
}

type stopResultMsg struct {
	name string
	err  error
}

type browserResultMsg struct {
	name string
	err  error
}

type runAllResultMsg struct {
	repoName string
	started  int
	err      error
}

type stopAllResultMsg struct {
	repoName string
	stopped  int
	err      error
}

type openersLoadedMsg struct {
	openers []opener.Opener
	err     error
}

type batchArchiveResultMsg struct {
	archived []string
	failed   []string
	err      error
}

type createRequestMsg struct {
	name     string
	rootPath string
	commonDir string
}
