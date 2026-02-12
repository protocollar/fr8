package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Worktree represents a git worktree entry.
type Worktree struct {
	Path   string
	HEAD   string
	Branch string
	Bare   bool
}

// WorktreeList returns all worktrees for the repo at dir.
func WorktreeList(dir string) ([]Worktree, error) {
	out, err := run(dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}
	return parsePorcelain(out), nil
}

// WorktreeAdd creates a new worktree at path. If newBranch is true, creates a
// new branch with the given name. startPoint optionally specifies the commit to
// branch from (e.g. "origin/main"); if empty, uses the current HEAD.
func WorktreeAdd(dir, path, branch string, newBranch bool, startPoint string) error {
	args := []string{"worktree", "add"}
	if newBranch {
		args = append(args, "-b", branch, path)
		if startPoint != "" {
			args = append(args, startPoint)
		}
	} else {
		args = append(args, path, branch)
	}
	_, err := run(dir, args...)
	if err != nil {
		return fmt.Errorf("git worktree add: %w", err)
	}
	return nil
}

// WorktreeRemove removes the worktree at path.
func WorktreeRemove(dir, path string) error {
	_, err := run(dir, "worktree", "remove", path, "--force")
	if err != nil {
		return fmt.Errorf("git worktree remove: %w", err)
	}
	return nil
}

// CommonDir returns the path to the shared .git directory.
func CommonDir(dir string) (string, error) {
	out, err := run(dir, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", fmt.Errorf("git rev-parse --git-common-dir: %w", err)
	}
	p := strings.TrimSpace(out)
	if !filepath.IsAbs(p) {
		p = filepath.Join(dir, p)
	}
	return filepath.Clean(p), nil
}

// RootWorktreePath returns the path to the main (first) worktree.
func RootWorktreePath(dir string) (string, error) {
	wts, err := WorktreeList(dir)
	if err != nil {
		return "", err
	}
	if len(wts) == 0 {
		return "", fmt.Errorf("no worktrees found")
	}
	return wts[0].Path, nil
}

// DefaultBranch returns "main" or "master", whichever exists.
func DefaultBranch(dir string) (string, error) {
	for _, branch := range []string{"main", "master"} {
		_, err := run(dir, "rev-parse", "--verify", "refs/heads/"+branch)
		if err == nil {
			return branch, nil
		}
	}
	return "", fmt.Errorf("neither main nor master branch found")
}

// CurrentBranch returns the current branch name for the repo at dir.
func CurrentBranch(dir string) (string, error) {
	out, err := run(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("git rev-parse --abbrev-ref HEAD: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// HasUncommittedChanges returns true if the worktree at dir has uncommitted changes.
func HasUncommittedChanges(dir string) (bool, error) {
	out, err := run(dir, "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("git status: %w", err)
	}
	return strings.TrimSpace(out) != "", nil
}

// IsInsideWorkTree returns true if dir is inside a git repository.
func IsInsideWorkTree(dir string) bool {
	_, err := run(dir, "rev-parse", "--is-inside-work-tree")
	return err == nil
}

// BranchExists returns true if the given branch exists locally.
func BranchExists(dir, branch string) bool {
	_, err := run(dir, "rev-parse", "--verify", "refs/heads/"+branch)
	return err == nil
}

// RemoteRefExists returns true if the given remote ref exists (e.g. "origin/main").
func RemoteRefExists(dir, ref string) bool {
	_, err := run(dir, "rev-parse", "--verify", "refs/remotes/"+ref)
	return err == nil
}

// Fetch runs git fetch for the given remote.
func Fetch(dir, remote string) error {
	_, err := run(dir, "fetch", remote)
	if err != nil {
		return fmt.Errorf("git fetch %s: %w", remote, err)
	}
	return nil
}

// IsMerged returns true if branch has been merged into target.
// Uses git merge-base --is-ancestor (exit 0 = merged, exit 1 = not merged).
func IsMerged(dir, branch, target string) (bool, error) {
	cmd := exec.Command("git", "merge-base", "--is-ancestor", branch, target)
	cmd.Dir = dir
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 1 {
			return false, nil
		}
	}
	return false, fmt.Errorf("git merge-base --is-ancestor: %w", err)
}

// AheadBehind returns how many commits branch is ahead and behind upstream.
func AheadBehind(dir, branch, upstream string) (ahead int, behind int, err error) {
	out, runErr := run(dir, "rev-list", "--count", "--left-right", branch+"..."+upstream)
	if runErr != nil {
		return 0, 0, fmt.Errorf("git rev-list --left-right: %w", runErr)
	}
	parts := strings.Fields(strings.TrimSpace(out))
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected rev-list output: %q", out)
	}
	ahead, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parsing ahead count: %w", err)
	}
	behind, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parsing behind count: %w", err)
	}
	return ahead, behind, nil
}

// TrackingBranch returns the upstream tracking branch for the given branch.
func TrackingBranch(dir, branch string) (string, error) {
	out, err := run(dir, "rev-parse", "--abbrev-ref", branch+"@{upstream}")
	if err != nil {
		return "", fmt.Errorf("no tracking branch for %s: %w", branch, err)
	}
	return strings.TrimSpace(out), nil
}

func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func parsePorcelain(output string) []Worktree {
	var worktrees []Worktree
	var current Worktree

	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			current.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "HEAD "):
			current.HEAD = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			current.Branch = strings.TrimPrefix(ref, "refs/heads/")
		case line == "bare":
			current.Bare = true
		case line == "":
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = Worktree{}
			}
		}
	}
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees
}
