package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/protocollar/fr8/internal/config"
	"github.com/protocollar/fr8/internal/env"
	"github.com/protocollar/fr8/internal/gh"
	"github.com/protocollar/fr8/internal/git"
	"github.com/protocollar/fr8/internal/registry"
	"github.com/protocollar/fr8/internal/state"
	"github.com/protocollar/fr8/internal/tmux"
	"github.com/protocollar/fr8/internal/workspace"
)

// mcpResult marshals v as JSON and returns it as MCP text content.
func mcpResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling result: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

// mcpError returns an MCP error result.
func mcpError(msg string) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(msg)},
		IsError: true,
	}, nil
}

// mcpResolveWorkspace resolves a workspace by name with optional repo filter.
// Unlike the CLI's resolveWorkspace(), this never detects from CWD — it always
// uses the global registry for lookup, since the MCP server runs as a long-lived process.
func mcpResolveWorkspace(name, repo string) (*state.Workspace, string, string, error) {
	if name == "" {
		return nil, "", "", fmt.Errorf("workspace name is required")
	}
	if repo != "" {
		return workspace.ResolveFromRepo(name, repo)
	}
	return workspace.ResolveGlobal(name)
}

// mcpResolveRepo resolves a repo's root path and git common dir from a repo name.
// Unlike the CLI, this never detects from CWD — the MCP server runs as a long-lived process.
func mcpResolveRepo(repo string) (rootPath, commonDir string, err error) {
	if repo == "" {
		return "", "", fmt.Errorf("repo parameter is required")
	}
	regPath, err := registry.DefaultPath()
	if err != nil {
		return "", "", err
	}
	reg, err := registry.Load(regPath)
	if err != nil {
		return "", "", fmt.Errorf("loading registry: %w", err)
	}
	r := reg.Find(repo)
	if r == nil {
		return "", "", fmt.Errorf("repo %q not found in registry (see: fr8 repo list)", repo)
	}
	rootPath, err = git.RootWorktreePath(r.Path)
	if err != nil {
		rootPath = r.Path
	}
	commonDir, err = git.CommonDir(r.Path)
	if err != nil {
		return "", "", fmt.Errorf("finding git common dir: %w", err)
	}
	return rootPath, commonDir, nil
}

func registerMCPTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("workspace_list",
			mcp.WithDescription("List workspaces. Without repo param, lists across all registered repos."),
			mcp.WithString("repo", mcp.Description("Filter to a specific repo name")),
			mcp.WithBoolean("running", mcp.Description("Only show running workspaces")),
			mcp.WithBoolean("dirty", mcp.Description("Only show workspaces with uncommitted changes")),
			mcp.WithBoolean("merged", mcp.Description("Only show workspaces whose branch is merged")),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
		),
		handleWorkspaceList,
	)

	s.AddTool(
		mcp.NewTool("workspace_status",
			mcp.WithDescription("Get workspace details including environment variables, process status, and dirty state."),
			mcp.WithString("name", mcp.Description("Workspace name"), mcp.Required()),
			mcp.WithString("repo", mcp.Description("Repo name (required if workspace exists in multiple repos)")),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
		),
		handleWorkspaceStatus,
	)

	s.AddTool(
		mcp.NewTool("workspace_create",
			mcp.WithDescription("Create a new workspace (git worktree, port allocation, file sync, setup script)."),
			mcp.WithString("name", mcp.Description("Workspace name (auto-generated if omitted)")),
			mcp.WithString("branch", mcp.Description("Branch name (creates new branch if it doesn't exist)")),
			mcp.WithString("remote", mcp.Description("Track an existing remote branch")),
			mcp.WithString("pr", mcp.Description("Create from a GitHub PR number (requires gh CLI)")),
			mcp.WithString("repo", mcp.Description("Target repo name from registry")),
			mcp.WithBoolean("no_setup", mcp.Description("Skip running the setup script")),
			mcp.WithBoolean("if_not_exists", mcp.Description("Succeed silently if workspace already exists")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
		),
		handleWorkspaceCreate,
	)

	s.AddTool(
		mcp.NewTool("workspace_archive",
			mcp.WithDescription("Archive (tear down) a workspace: runs archive script, removes worktree, frees port."),
			mcp.WithString("name", mcp.Description("Workspace name"), mcp.Required()),
			mcp.WithString("repo", mcp.Description("Repo name")),
			mcp.WithBoolean("force", mcp.Description("Skip uncommitted changes check")),
			mcp.WithBoolean("if_exists", mcp.Description("Succeed silently if workspace not found")),
			mcp.WithDestructiveHintAnnotation(true),
		),
		handleWorkspaceArchive,
	)

	s.AddTool(
		mcp.NewTool("workspace_run",
			mcp.WithDescription("Start the dev server in a background tmux session."),
			mcp.WithString("name", mcp.Description("Workspace name"), mcp.Required()),
			mcp.WithString("repo", mcp.Description("Repo name")),
			mcp.WithBoolean("if_not_running", mcp.Description("Succeed silently if already running")),
			mcp.WithDestructiveHintAnnotation(false),
		),
		handleWorkspaceRun,
	)

	s.AddTool(
		mcp.NewTool("workspace_stop",
			mcp.WithDescription("Stop a workspace's background tmux session."),
			mcp.WithString("name", mcp.Description("Workspace name"), mcp.Required()),
			mcp.WithString("repo", mcp.Description("Repo name")),
			mcp.WithBoolean("if_running", mcp.Description("Succeed silently if not running")),
			mcp.WithDestructiveHintAnnotation(false),
		),
		handleWorkspaceStop,
	)

	s.AddTool(
		mcp.NewTool("workspace_env",
			mcp.WithDescription("Get workspace environment variables (FR8_* vars)."),
			mcp.WithString("name", mcp.Description("Workspace name"), mcp.Required()),
			mcp.WithString("repo", mcp.Description("Repo name")),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
		),
		handleWorkspaceEnv,
	)

	s.AddTool(
		mcp.NewTool("workspace_logs",
			mcp.WithDescription("Get recent output from a workspace's background tmux session."),
			mcp.WithString("name", mcp.Description("Workspace name"), mcp.Required()),
			mcp.WithString("repo", mcp.Description("Repo name")),
			mcp.WithNumber("lines", mcp.Description("Number of lines to capture (default: 50)")),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
		),
		handleWorkspaceLogs,
	)

	s.AddTool(
		mcp.NewTool("workspace_rename",
			mcp.WithDescription("Rename a workspace."),
			mcp.WithString("old_name", mcp.Description("Current workspace name"), mcp.Required()),
			mcp.WithString("new_name", mcp.Description("New workspace name"), mcp.Required()),
			mcp.WithString("repo", mcp.Description("Repo name")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
		),
		handleWorkspaceRename,
	)

	s.AddTool(
		mcp.NewTool("repo_list",
			mcp.WithDescription("List registered repos."),
			mcp.WithBoolean("workspaces", mcp.Description("Include workspace details for each repo")),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
		),
		handleRepoList,
	)

	s.AddTool(
		mcp.NewTool("config_show",
			mcp.WithDescription("Show resolved fr8 configuration for a repo."),
			mcp.WithString("repo", mcp.Description("Repo name")),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
		),
		handleConfigShow,
	)

	s.AddTool(
		mcp.NewTool("config_doctor",
			mcp.WithDescription("Check fr8 configuration health for a repo. Reports errors, warnings, and fixable issues."),
			mcp.WithString("repo", mcp.Description("Repo name")),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
		),
		handleConfigDoctor,
	)
}

func handleWorkspaceList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repo := req.GetString("repo", "")
	filterRunning := req.GetBool("running", false)
	filterDirty := req.GetBool("dirty", false)
	filterMerged := req.GetBool("merged", false)
	hasFilters := filterRunning || filterDirty || filterMerged

	regPath, err := registry.DefaultPath()
	if err != nil {
		return mcpError(err.Error())
	}
	reg, err := registry.Load(regPath)
	if err != nil {
		return mcpError(fmt.Sprintf("loading registry: %v", err))
	}

	hasTmux := tmux.Available() == nil
	var items []workspaceListItem

	for _, r := range reg.Repos {
		if repo != "" && r.Name != repo {
			continue
		}
		commonDir, err := git.CommonDir(r.Path)
		if err != nil {
			continue
		}
		st, err := state.Load(commonDir)
		if err != nil {
			continue
		}
		rootPath, _ := git.RootWorktreePath(r.Path)
		defaultBranch, _ := git.DefaultBranch(rootPath)

		for _, ws := range st.Workspaces {
			running := false
			if hasTmux {
				sessionName := tmux.SessionName(r.Name, ws.Name)
				running = tmux.IsRunning(sessionName)
			}

			if hasFilters {
				if filterRunning && !running {
					continue
				}
				if filterDirty {
					dc, _ := git.DirtyStatus(ws.Path)
					if !dc.Dirty() {
						continue
					}
				}
				if filterMerged && defaultBranch != "" {
					merged, _ := git.IsMerged(ws.Path, ws.Branch, defaultBranch)
					if !merged {
						continue
					}
				}
			}

			items = append(items, workspaceListItem{
				Repo:      r.Name,
				Name:      ws.Name,
				Branch:    ws.Branch,
				Port:      ws.Port,
				Path:      ws.Path,
				Running:   running,
				CreatedAt: ws.CreatedAt,
			})
		}
	}

	if items == nil {
		items = []workspaceListItem{}
	}
	return mcpResult(items)
}

func handleWorkspaceStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	repo := req.GetString("repo", "")

	ws, rootPath, _, err := mcpResolveWorkspace(name, repo)
	if err != nil {
		return mcpError(err.Error())
	}

	defaultBranch, _ := git.DefaultBranch(rootPath)
	branch, _ := git.CurrentBranch(ws.Path)
	if branch == "" {
		branch = ws.Branch
	}

	dc, _ := git.DirtyStatus(ws.Path)
	lastCommit, _ := git.LastCommit(ws.Path)
	var lastCommitPtr *git.CommitInfo
	if lastCommit.Subject != "" {
		lastCommitPtr = &lastCommit
	}

	var pr *gh.PRInfo
	if gh.Available() == nil {
		pr, _ = gh.PRStatus(ws.Path, branch)
	}

	running := false
	if tmux.Available() == nil {
		sessionName := tmux.SessionName(tmux.RepoName(rootPath), ws.Name)
		running = tmux.IsRunning(sessionName)
	}

	vars := env.BuildFr8Only(ws, rootPath, defaultBranch)
	envMap := make(map[string]string)
	for _, v := range vars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 && strings.HasPrefix(parts[0], "FR8_") {
			envMap[parts[0]] = parts[1]
		}
	}

	return mcpResult(workspaceStatusJSON{
		Name:       ws.Name,
		Path:       ws.Path,
		Branch:     branch,
		Port:       ws.Port,
		PortEnd:    ws.Port + 9,
		Dirty:      dc.Dirty(),
		Staged:     dc.Staged,
		Modified:   dc.Modified,
		Untracked:  dc.Untracked,
		Running:    running,
		CreatedAt:  ws.CreatedAt,
		Env:        envMap,
		LastCommit: lastCommitPtr,
		PR:         pr,
	})
}

func handleWorkspaceCreate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	wsName := req.GetString("name", "")
	branch := req.GetString("branch", "")
	remote := req.GetString("remote", "")
	pr := req.GetString("pr", "")
	repo := req.GetString("repo", "")
	noSetup := req.GetBool("no_setup", false)
	ifNotExists := req.GetBool("if_not_exists", false)

	rootPath, commonDir, err := mcpResolveRepo(repo)
	if err != nil {
		return mcpError(err.Error())
	}

	// Resolve branch from PR or remote
	trackRemote := false
	if pr != "" {
		resolved, err := resolvePRBranch(rootPath, pr)
		if err != nil {
			return mcpError(err.Error())
		}
		branch = resolved
		trackRemote = true
	} else if remote != "" {
		branch = remote
		trackRemote = true
	}

	// Handle if_not_exists before calling createWorkspace (avoids global flag dependency)
	if ifNotExists && wsName != "" {
		st, err := state.Load(commonDir)
		if err == nil {
			if existing := st.Find(wsName); existing != nil {
				return mcpResult(struct {
					Action    string           `json:"action"`
					Workspace *state.Workspace `json:"workspace"`
				}{Action: "already_exists", Workspace: existing})
			}
		}
	}

	ws, err := createWorkspace(rootPath, commonDir, wsName, branch, trackRemote, !noSetup, false)
	if err != nil {
		return mcpError(err.Error())
	}

	return mcpResult(struct {
		Action    string           `json:"action"`
		Workspace *state.Workspace `json:"workspace"`
	}{Action: "created", Workspace: ws})
}

func handleWorkspaceArchive(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	repo := req.GetString("repo", "")
	force := req.GetBool("force", false)
	ifExists := req.GetBool("if_exists", false)

	ws, rootPath, commonDir, err := mcpResolveWorkspace(name, repo)
	if err != nil {
		if ifExists {
			return mcpResult(struct {
				Action string `json:"action"`
			}{Action: "not_found"})
		}
		return mcpError(err.Error())
	}

	cfg, err := config.Load(rootPath)
	if err != nil {
		return mcpError(fmt.Sprintf("loading config: %v", err))
	}

	st, err := state.Load(commonDir)
	if err != nil {
		return mcpError(fmt.Sprintf("loading state: %v", err))
	}

	// Safety: check for uncommitted changes
	if !force {
		dirty, _ := git.HasUncommittedChanges(ws.Path)
		if dirty {
			return mcpError(fmt.Sprintf("workspace %q has uncommitted changes (use force=true to override)", ws.Name))
		}
	}

	// Stop tmux session if running
	if tmux.Available() == nil {
		sessionName := tmux.SessionName(tmux.RepoName(rootPath), ws.Name)
		if tmux.IsRunning(sessionName) {
			_ = tmux.Stop(sessionName)
		}
	}

	// Run archive script
	defaultBranch, _ := git.DefaultBranch(rootPath)
	if cfg.Scripts.Archive != "" {
		envVars := env.Build(ws, rootPath, defaultBranch)
		_ = runScript(cfg.Scripts.Archive, ws.Path, envVars)
	}

	// Remove worktree
	_ = git.WorktreeRemove(rootPath, ws.Path)

	// Update state
	_ = st.Remove(ws.Name)
	if err := st.Save(commonDir); err != nil {
		return mcpError(fmt.Sprintf("saving state: %v", err))
	}

	return mcpResult(struct {
		Action    string `json:"action"`
		Workspace struct {
			Name   string `json:"name"`
			Branch string `json:"branch"`
			Port   int    `json:"port"`
			Path   string `json:"path"`
		} `json:"workspace"`
	}{
		Action: "archived",
		Workspace: struct {
			Name   string `json:"name"`
			Branch string `json:"branch"`
			Port   int    `json:"port"`
			Path   string `json:"path"`
		}{Name: ws.Name, Branch: ws.Branch, Port: ws.Port, Path: ws.Path},
	})
}

func handleWorkspaceRun(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	repo := req.GetString("repo", "")
	ifNotRunning := req.GetBool("if_not_running", false)

	if err := tmux.Available(); err != nil {
		return mcpError(err.Error())
	}

	ws, rootPath, _, err := mcpResolveWorkspace(name, repo)
	if err != nil {
		return mcpError(err.Error())
	}

	cfg, err := config.Load(rootPath)
	if err != nil {
		return mcpError(fmt.Sprintf("loading config: %v", err))
	}
	if cfg.Scripts.Run == "" {
		return mcpError("no run script configured (add \"scripts.run\" to fr8.json)")
	}

	defaultBranch, _ := git.DefaultBranch(rootPath)
	envVars := env.BuildFr8Only(ws, rootPath, defaultBranch)
	sessionName := tmux.SessionName(tmux.RepoName(rootPath), ws.Name)

	if tmux.IsRunning(sessionName) {
		if ifNotRunning {
			return mcpResult(struct {
				Action    string `json:"action"`
				Workspace string `json:"workspace"`
				Session   string `json:"session"`
			}{Action: "already_running", Workspace: ws.Name, Session: sessionName})
		}
		return mcpError(fmt.Sprintf("session %q is already running (use workspace_stop first or set if_not_running=true)", sessionName))
	}

	if err := tmux.Start(sessionName, ws.Path, cfg.Scripts.Run, envVars); err != nil {
		return mcpError(err.Error())
	}

	return mcpResult(struct {
		Action    string `json:"action"`
		Workspace string `json:"workspace"`
		Session   string `json:"session"`
	}{Action: "started", Workspace: ws.Name, Session: sessionName})
}

func handleWorkspaceStop(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	repo := req.GetString("repo", "")
	ifRunning := req.GetBool("if_running", false)

	if err := tmux.Available(); err != nil {
		return mcpError(err.Error())
	}

	ws, rootPath, _, err := mcpResolveWorkspace(name, repo)
	if err != nil {
		return mcpError(err.Error())
	}

	sessionName := tmux.SessionName(tmux.RepoName(rootPath), ws.Name)
	if !tmux.IsRunning(sessionName) {
		if !ifRunning {
			return mcpError(fmt.Sprintf("workspace %q is not running", ws.Name))
		}
		return mcpResult(struct {
			Action    string `json:"action"`
			Workspace string `json:"workspace"`
			Session   string `json:"session"`
		}{Action: "already_stopped", Workspace: ws.Name, Session: sessionName})
	}

	if err := tmux.Stop(sessionName); err != nil {
		return mcpError(err.Error())
	}

	return mcpResult(struct {
		Action    string `json:"action"`
		Workspace string `json:"workspace"`
		Session   string `json:"session"`
	}{Action: "stopped", Workspace: ws.Name, Session: sessionName})
}

func handleWorkspaceEnv(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	repo := req.GetString("repo", "")

	ws, rootPath, _, err := mcpResolveWorkspace(name, repo)
	if err != nil {
		return mcpError(err.Error())
	}

	defaultBranch, _ := git.DefaultBranch(rootPath)
	vars := env.BuildFr8Only(ws, rootPath, defaultBranch)

	envMap := make(map[string]string)
	for _, v := range vars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 && strings.HasPrefix(parts[0], "FR8_") {
			envMap[parts[0]] = parts[1]
		}
	}

	return mcpResult(envMap)
}

func handleWorkspaceLogs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	repo := req.GetString("repo", "")
	lines := req.GetInt("lines", 50)

	if err := tmux.Available(); err != nil {
		return mcpError(err.Error())
	}

	ws, rootPath, _, err := mcpResolveWorkspace(name, repo)
	if err != nil {
		return mcpError(err.Error())
	}

	sessionName := tmux.SessionName(tmux.RepoName(rootPath), ws.Name)
	output, err := tmux.CapturePanes(sessionName, lines)
	if err != nil {
		return mcpError(err.Error())
	}

	return mcpResult(struct {
		Workspace string `json:"workspace"`
		Session   string `json:"session"`
		Output    string `json:"output"`
	}{Workspace: ws.Name, Session: sessionName, Output: output})
}

func handleWorkspaceRename(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	oldName := req.GetString("old_name", "")
	newName := req.GetString("new_name", "")
	repo := req.GetString("repo", "")

	if oldName == "" || newName == "" {
		return mcpError("both old_name and new_name are required")
	}

	ws, rootPath, commonDir, err := mcpResolveWorkspace(oldName, repo)
	if err != nil {
		return mcpError(err.Error())
	}

	st, err := state.Load(commonDir)
	if err != nil {
		return mcpError(fmt.Sprintf("loading state: %v", err))
	}

	oldPath := ws.Path
	newPath := filepath.Join(filepath.Dir(oldPath), newName)
	if err := git.WorktreeMove(rootPath, oldPath, newPath); err != nil {
		return mcpError(fmt.Sprintf("moving worktree: %v", err))
	}

	if err := st.Rename(oldName, newName); err != nil {
		return mcpError(err.Error())
	}
	renamed := st.Find(newName)
	renamed.Path = newPath

	if err := st.Save(commonDir); err != nil {
		return mcpError(fmt.Sprintf("saving state: %v", err))
	}

	// Rename tmux session if running
	if tmux.Available() == nil {
		repoName := tmux.RepoName(rootPath)
		oldSession := tmux.SessionName(repoName, oldName)
		if tmux.IsRunning(oldSession) {
			newSession := tmux.SessionName(repoName, newName)
			_ = tmux.RenameSession(oldSession, newSession)
		}
	}

	return mcpResult(struct {
		Action  string `json:"action"`
		OldName string `json:"old_name"`
		NewName string `json:"new_name"`
		Path    string `json:"path"`
	}{Action: "renamed", OldName: oldName, NewName: newName, Path: newPath})
}

func handleRepoList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	withWorkspaces := req.GetBool("workspaces", false)

	regPath, err := registry.DefaultPath()
	if err != nil {
		return mcpError(err.Error())
	}
	reg, err := registry.Load(regPath)
	if err != nil {
		return mcpError(fmt.Sprintf("loading registry: %v", err))
	}

	type mcpRepoItem struct {
		Name       string             `json:"name"`
		Path       string             `json:"path"`
		Workspaces []workspaceListItem `json:"workspaces,omitempty"`
	}

	var items []mcpRepoItem
	for _, r := range reg.Repos {
		item := mcpRepoItem{Name: r.Name, Path: r.Path}
		if withWorkspaces {
			item.Workspaces = repoWorkspaces(r)
		}
		items = append(items, item)
	}
	if items == nil {
		items = []mcpRepoItem{}
	}
	return mcpResult(items)
}

func handleConfigShow(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repo := req.GetString("repo", "")
	rootPath, _, err := mcpResolveRepo(repo)
	if err != nil {
		return mcpError(err.Error())
	}

	cfg, err := config.Load(rootPath)
	if err != nil {
		return mcpError(fmt.Sprintf("loading config: %v", err))
	}

	resolved := map[string]interface{}{
		"scripts": map[string]string{
			"setup":   cfg.Scripts.Setup,
			"run":     cfg.Scripts.Run,
			"archive": cfg.Scripts.Archive,
		},
		"port_range":             cfg.PortRange,
		"base_port":              cfg.BasePort,
		"worktree_path":          cfg.WorktreePath,
		"resolved_worktree_path": config.ResolveWorktreePath(cfg, rootPath),
	}

	return mcpResult(resolved)
}

func handleConfigDoctor(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repo := req.GetString("repo", "")
	rootPath, _, err := mcpResolveRepo(repo)
	if err != nil {
		return mcpError(err.Error())
	}

	cfg, err := config.Load(rootPath)
	if err != nil {
		return mcpError(fmt.Sprintf("loading config: %v", err))
	}

	var warnings []string
	var configErrors []string

	for name, script := range map[string]string{
		"setup":   cfg.Scripts.Setup,
		"run":     cfg.Scripts.Run,
		"archive": cfg.Scripts.Archive,
	} {
		if script == "" {
			continue
		}
		parts := strings.Fields(script)
		if _, err := exec.LookPath(parts[0]); err != nil {
			if _, err := os.Stat(fmt.Sprintf("%s/%s", rootPath, parts[0])); err != nil {
				warnings = append(warnings, fmt.Sprintf("scripts.%s: %q not found in $PATH or repo", name, parts[0]))
			}
		}
	}

	wtPath := config.ResolveWorktreePath(cfg, rootPath)
	if info, err := os.Stat(wtPath); err == nil {
		if !info.IsDir() {
			configErrors = append(configErrors, fmt.Sprintf("worktree_path: %q exists but is not a directory", wtPath))
		}
	}

	if cfg.BasePort < 1024 {
		warnings = append(warnings, fmt.Sprintf("base_port: %d is a privileged port (< 1024)", cfg.BasePort))
	}
	if cfg.BasePort > 65535 {
		configErrors = append(configErrors, fmt.Sprintf("base_port: %d is out of range (> 65535)", cfg.BasePort))
	}
	if cfg.PortRange < 1 {
		configErrors = append(configErrors, fmt.Sprintf("port_range: %d must be at least 1", cfg.PortRange))
	}

	if configErrors == nil {
		configErrors = []string{}
	}
	if warnings == nil {
		warnings = []string{}
	}

	return mcpResult(struct {
		Valid    bool     `json:"valid"`
		Errors   []string `json:"errors"`
		Warnings []string `json:"warnings"`
	}{
		Valid:    len(configErrors) == 0,
		Errors:   configErrors,
		Warnings: warnings,
	})
}
