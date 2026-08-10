// Package pr provides PR discovery and base-branch resolution for plan-workflow branches.
package pr

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"threehillpath.com/claude-plan-workflow/tool/internal/clierr"
	"threehillpath.com/claude-plan-workflow/tool/internal/config"
	"threehillpath.com/claude-plan-workflow/tool/internal/exec"
	"threehillpath.com/claude-plan-workflow/tool/internal/gh"
	"threehillpath.com/claude-plan-workflow/tool/internal/parse"
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

// identMatches reports whether a candidate PR title structurally matches ident.
// Both are parsed via parse.PlanIdent; a match requires Plan, Suffix, Phase, and
// Kind to all be equal. If ident itself fails to parse, this falls back to a
// substring check against the candidate title (defensive — no current caller
// passes a non-conforming ident, but nothing guarantees against it).
func identMatches(ident, title string) bool {
	wantIdent, ok := parse.PlanIdent(ident)
	if !ok {
		return strings.Contains(title, ident)
	}
	gotIdent, ok := parse.PlanIdent(title)
	if !ok {
		return false
	}
	return gotIdent == wantIdent
}

// Find searches for a PR whose title structurally matches ident (e.g. "[PLAN-00002-3]"),
// filtered by state. A missing match is not an error — the result will have Found=false.
func Find(ctx context.Context, runner exec.Runner, cfg *config.Config, ident string, state State) (FindResult, error) {
	prs, err := gh.New(runner).PRList(ctx, cfg.Repo, state.ghArg(), 200)
	if err != nil {
		return FindResult{}, fmt.Errorf("pr find: %w", err)
	}

	for _, p := range prs {
		if identMatches(ident, p.Title) {
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
