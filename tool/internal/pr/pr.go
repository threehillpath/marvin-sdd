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

// State filters PR search by lifecycle state.
type State int

const (
	StateAny    State = iota // search open, closed, and merged PRs
	StateOpen                // open PRs only
	StateMerged              // merged PRs only
)

// ghArg returns the --state value for gh pr list.
func (s State) ghArg() string {
	switch s {
	case StateOpen:
		return "open"
	case StateMerged:
		return "merged"
	default:
		return "all"
	}
}

// ParseState maps the CLI flag value ("open", "merged", "any") to a State.
func ParseState(s string) (State, error) {
	switch s {
	case "any", "":
		return StateAny, nil
	case "open":
		return StateOpen, nil
	case "merged":
		return StateMerged, nil
	default:
		return StateAny, fmt.Errorf("invalid state %q: must be open, merged, or any", s)
	}
}

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

// Find searches for a PR whose title contains ident (e.g. "[PLAN-00002-3]"),
// filtered by state. A missing match is not an error — the result will have Found=false.
func Find(ctx context.Context, runner exec.Runner, cfg *config.Config, ident string, state State) (FindResult, error) {
	args := []string{
		"pr", "list",
		"--repo", cfg.Repo,
		"--json", "number,title,url,headRefName,baseRefName,state",
		"--state", state.ghArg(),
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

// phaseBranchRe matches <type>/PLAN-NNNNN/phase-[suffix-]N (a phase branch).
// Group 1: type, Group 2: plan number digits, Group 3: optional suffix, Group 4: phase number.
var phaseBranchRe = regexp.MustCompile(`^([a-z]+)/PLAN-(\d{5})/phase-(?:([a-z]+)-)?(\d+)$`)

// mainBranchRe matches <type>/PLAN-NNNNN/main[-suffix] (trunk branch).
// Group 1: type, Group 2: plan number digits, Group 3: optional suffix.
var mainBranchRe = regexp.MustCompile(`^([a-z]+)/PLAN-(\d{5})/main(?:-([a-z]+))?$`)

// Base maps a plan branch to its PR target base branch:
//   - <type>/PLAN-XXXXX/phase-N (phase branch)   → <type>/PLAN-XXXXX/main (trunk branch)
//   - <type>/PLAN-XXXXX/main    (trunk branch)   → main
//   - anything else                              → CLIError{Code:1}
//
// <type> and any suffix are preserved from the matched branch.
func Base(branch string) (string, error) {
	if m := phaseBranchRe.FindStringSubmatch(branch); m != nil {
		typ := m[1]
		num := m[2]
		suffix := m[3]
		if suffix == "" {
			return fmt.Sprintf("%s/PLAN-%s/main", typ, num), nil
		}
		return fmt.Sprintf("%s/PLAN-%s/main-%s", typ, num, suffix), nil
	}
	if m := mainBranchRe.FindStringSubmatch(branch); m != nil {
		return "main", nil
	}
	return "", clierr.Operational(fmt.Sprintf("cannot determine base for branch %q: not a plan branch", branch))
}
