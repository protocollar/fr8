package gh

import (
	"encoding/json"
	"os/exec"
	"strings"
)

// PRInfo holds GitHub pull request status.
type PRInfo struct {
	Number         int    `json:"number"`
	State          string `json:"state"`
	IsDraft        bool   `json:"is_draft"`
	ReviewDecision string `json:"review_decision"`
	URL            string `json:"url"`
}

// Available returns nil if the gh CLI is installed.
func Available() error {
	_, err := exec.LookPath("gh")
	return err
}

// PRStatus returns PR info for the given branch, or nil if no PR exists.
// Returns nil, nil for all non-critical failures (gh missing, not a GitHub repo,
// no PR for the branch, or branch name reuse after merge).
func PRStatus(dir, branch string) (*PRInfo, error) {
	if Available() != nil {
		return nil, nil
	}

	cmd := exec.Command("gh", "pr", "view", branch,
		"--json", "number,state,isDraft,reviewDecision,url,headRefOid",
		"-q", ".")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		// gh returns non-zero for "no PR found", not a GitHub repo, etc.
		return nil, nil
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil, nil
	}

	var raw struct {
		Number         int    `json:"number"`
		State          string `json:"state"`
		IsDraft        bool   `json:"isDraft"`
		ReviewDecision string `json:"reviewDecision"`
		URL            string `json:"url"`
		HeadRefOid     string `json:"headRefOid"`
	}
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return nil, nil
	}
	if raw.Number == 0 {
		return nil, nil
	}

	// Verify the PR's head commit is in the branch's history. This prevents
	// showing stale PRs when a branch name is reused after a squash merge.
	if raw.HeadRefOid != "" {
		check := exec.Command("git", "merge-base", "--is-ancestor", raw.HeadRefOid, "HEAD")
		check.Dir = dir
		if err := check.Run(); err != nil {
			return nil, nil
		}
	}

	return &PRInfo{
		Number:         raw.Number,
		State:          raw.State,
		IsDraft:        raw.IsDraft,
		ReviewDecision: raw.ReviewDecision,
		URL:            raw.URL,
	}, nil
}
