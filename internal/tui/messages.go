package tui

import (
	"github.com/protocollar/fr8/internal/gh"
	"github.com/protocollar/fr8/internal/git"
	"github.com/protocollar/fr8/internal/registry"
	"github.com/protocollar/fr8/internal/tmux"
	"github.com/protocollar/fr8/internal/userconfig"
)

type viewState int

const (
	viewRepoList viewState = iota
	viewWorkspaceList
	viewConfirmArchive
	viewConfirmBatchArchive
	viewOpenerPicker
	viewCreateWorkspace
	viewHelp
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
	Workspace     registry.Workspace
	Branch        string          // live branch from git (not stored in state)
	DirtyCount    git.DirtyCount  // staged/modified/untracked counts
	Merged        bool
	Ahead         int             // ahead of upstream tracking branch
	Behind        int             // behind upstream tracking branch
	DefaultAhead  int             // ahead of default branch
	DefaultBehind int             // behind default branch
	LastCommit    *git.CommitInfo // nil if unavailable
	PR            *gh.PRInfo      // nil if no PR / gh unavailable
	PortFree      bool            // true when nothing is listening on the workspace port
	Running       bool            // true when a tmux session is active for this workspace
	StatusErr     error
}

// Messages for async operations.

type reposLoadedMsg struct {
	repos []repoItem
	err   error
}

type workspacesLoadedMsg struct {
	workspaces    []workspaceItem
	repoName      string
	rootPath      string
	defaultBranch string
	err           error
}

type archiveResultMsg struct {
	name string
	err  error
}

type shellRequestMsg struct {
	workspace registry.Workspace
	rootPath  string
}

type attachRequestMsg struct {
	workspace registry.Workspace
	rootPath  string
}

type openRequestMsg struct {
	workspace  registry.Workspace
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
	openers []userconfig.Opener
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
}

// Toast notifications
type toastTickMsg struct{}

// Multi-select batch operations
type batchStartResultMsg struct {
	started int
	err     error
}

type batchStopResultMsg struct {
	stopped int
	err     error
}

// Auto-refresh
type autoRefreshTickMsg struct{}

type autoRefreshResultMsg struct {
	sessions []tmux.Session
	err      error
}
