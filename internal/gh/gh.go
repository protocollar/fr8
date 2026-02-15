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
// no PR for the branch).
func PRStatus(dir, branch string) (*PRInfo, error) {
	if Available() != nil {
		return nil, nil
	}

	cmd := exec.Command("gh", "pr", "view", branch,
		"--json", "number,state,isDraft,reviewDecision,url",
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

	var pr PRInfo
	if err := json.Unmarshal([]byte(trimmed), &pr); err != nil {
		return nil, nil
	}
	if pr.Number == 0 {
		return nil, nil
	}
	return &pr, nil
}
