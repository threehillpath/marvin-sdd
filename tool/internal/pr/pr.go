// Package pr provides PR discovery and base-branch resolution for plan-workflow branches.
package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"threehillpath.com/claude-plan-workflow/tool/internal/clierr"
	"threehillpath.com/claude-plan-workflow/tool/internal/config"
	"threehillpath.com/claude-plan-workflow/tool/internal/exec"
)

// FindResult is the output of Find.
type FindResult struct {
	// Found is true when a PR with the matching ident was located.
	Found  bool
	Number int
	Title  string
	URL    string
	Head   string
	Base   string
	State  string
}

// ghPR is the JSON shape from gh pr list --json.
type ghPR struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	HeadRefName string `json:"headRefName"`
	BaseRefName string `json:"baseRefName"`
	State       string `json:"state"`
}

// Find searches for an open PR whose title contains ident (e.g. "[PLAN-00002-3]").
// A missing match is not an error — the result will have Found=false.
func Find(ctx context.Context, runner exec.Runner, cfg *config.Config, ident string) (FindResult, error) {
	args := []string{
		"pr", "list",
		"--repo", cfg.Repo,
		"--json", "number,title,url,headRefName,baseRefName,state",
		"--state", "all",
		"--limit", "200",
	}
	stdout, stderr, code, err := runner.Run(ctx, "gh", args...)
	if err != nil {
		return FindResult{}, fmt.Errorf("pr find: %w", err)
	}
	if code != 0 {
		return FindResult{}, fmt.Errorf("pr find: gh exited %d: %s", code, stderr)
	}

	var prs []ghPR
	if err := json.Unmarshal(stdout, &prs); err != nil {
		return FindResult{}, fmt.Errorf("pr find: parse response: %w", err)
	}

	for _, p := range prs {
		if strings.Contains(p.Title, ident) {
			return FindResult{
				Found:  true,
				Number: p.Number,
				Title:  p.Title,
				URL:    p.URL,
				Head:   p.HeadRefName,
				Base:   p.BaseRefName,
				State:  p.State,
			}, nil
		}
	}
	return FindResult{Found: false}, nil
}

// phaseBranchRe matches feature/plan-NNNNN[-suffix]-N (a phase branch).
// Group 1: plan number digits, Group 2: optional suffix, Group 3: phase number.
var phaseBranchRe = regexp.MustCompile(`^feature/plan-(\d{5})(?:-([a-z]+))?-(\d+)$`)

// implBranchRe matches feature/plan-NNNNN[-suffix] (impl branch, no trailing -N).
var implBranchRe = regexp.MustCompile(`^feature/plan-(\d{5})(?:-([a-z]+))?$`)

// Base maps a plan branch to its PR target base branch:
//   - feature/plan-XXXXX-N (phase branch) → feature/plan-XXXXX (impl branch)
//   - feature/plan-XXXXX   (impl branch)  → main
//   - anything else         → CLIError{Code:1}
func Base(branch string) (string, error) {
	if m := phaseBranchRe.FindStringSubmatch(branch); m != nil {
		num := m[1]
		suffix := m[2]
		if suffix == "" {
			return fmt.Sprintf("feature/plan-%s", num), nil
		}
		return fmt.Sprintf("feature/plan-%s-%s", num, suffix), nil
	}
	if m := implBranchRe.FindStringSubmatch(branch); m != nil {
		return "main", nil
	}
	return "", clierr.Operational(fmt.Sprintf("cannot determine base for branch %q: not a plan branch", branch))
}
