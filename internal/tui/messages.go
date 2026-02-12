package tui

import (
	"github.com/thomascarr/fr8/internal/registry"
	"github.com/thomascarr/fr8/internal/state"
)

type viewState int

const (
	viewRepoList viewState = iota
	viewWorkspaceList
	viewConfirmArchive
)

// repoItem is a repo with preloaded workspace count.
type repoItem struct {
	Repo           registry.Repo
	WorkspaceCount int
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

type runRequestMsg struct {
	workspace state.Workspace
	rootPath  string
}

type browserRequestMsg struct {
	workspace state.Workspace
	rootPath  string
}
