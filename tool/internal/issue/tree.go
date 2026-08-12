package issue

import (
	"context"
	"fmt"
	"sort"

	"threehillpath.com/claude-plan-workflow/tool/internal/board"
	"threehillpath.com/claude-plan-workflow/tool/internal/config"
	"threehillpath.com/claude-plan-workflow/tool/internal/exec"
	"threehillpath.com/claude-plan-workflow/tool/internal/gh"
	"threehillpath.com/claude-plan-workflow/tool/internal/parse"
)

// Node is one issue in a plan hierarchy, as returned by Tree.
type Node struct {
	Kind   string `json:"kind"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Status string `json:"status"`
}

// kindOf classifies a node's title as "arch", "impl", or "phase" via
// parse.PlanIdent. Titles that fail to parse (should not occur for nodes
// Tree adds, since every addition point already validated the title) fall
// back to "impl" defensively.
func kindOf(title string) string {
	ident, ok := parse.PlanIdent(title)
	if !ok {
		return "impl"
	}
	if ident.Phase != 0 {
		return "phase"
	}
	if ident.Kind == parse.KindArch {
		return "arch"
	}
	return "impl"
}

// Tree resolves the full arch/impl/phase hierarchy for a plan-related issue
// number via real GitHub sub-issue relationships (IssueRef/ParentIssue/
// SubIssues), replacing the historical --title-prefix string-matching
// lookup. See tool/internal/cli/handlers_integrations.go's newIssueTreeCmd
// for the CLI-level rendering of this result.
//
// Behavior:
//  1. target's own title/state is resolved via IssueRef first — the walk
//     always includes target as a node, whether or not it has a parent or
//     children.
//  2. If target's title does not parse via parse.PlanIdent, Tree returns an
//     empty slice: target is not a plan issue at all.
//  3. ParentIssue is walked upward until a parent's title fails to parse
//     (or there is no parent) — this is the arch-plan root for the walk,
//     though it may just be target itself for an unlinked plan issue.
//  4. SubIssues is queried once on that root. If the root's own title
//     parses as parse.KindArch, its children are impl plans: when target IS
//     the arch root, every impl child is kept and each is walked one level
//     further for its phases; otherwise only the impl child whose Plan and
//     Suffix match target's own is kept (sibling impl tracks are excluded
//     entirely) and walked for its phases. If the root's title does not
//     parse as KindArch (an isolated impl/phase root with no arch above
//     it), the root's SubIssues are themselves the leaf phase nodes — no
//     further recursion.
//  5. Nodes are deduplicated by issue number.
//  6. status is resolved via one board.List call, mapping issue number to
//     status from that single result set (not board.Status per node, which
//     would multiply a full board scan by node count).
func Tree(ctx context.Context, runner exec.Runner, cfg *config.Config, repo string, number int) ([]Node, error) {
	client := gh.New(runner)

	targetRef, _, err := client.IssueRef(ctx, repo, number)
	if err != nil {
		return nil, fmt.Errorf("issue tree: %w", err)
	}

	targetIdent, ok := parse.PlanIdent(targetRef.Title)
	if !ok {
		return []Node{}, nil
	}

	// Walk upward to the arch-plan root, stopping at the first parent whose
	// title does not parse as a plan ident, when there is no parent, or when
	// a parent repeats one already visited (defends against a cyclic or
	// self-referential parent chain in the remote data; GitHub's sub-issue
	// API rejects cycles today, so this is a defensive bound, not a live case).
	visited := map[int]bool{targetRef.Number: true}
	rootRef := targetRef
	cur := targetRef
	for {
		parentRef, hasParent, perr := client.ParentIssue(ctx, repo, cur.Number)
		if perr != nil {
			return nil, fmt.Errorf("issue tree: %w", perr)
		}
		if !hasParent {
			break
		}
		if _, ok := parse.PlanIdent(parentRef.Title); !ok {
			break
		}
		if visited[parentRef.Number] {
			break
		}
		visited[parentRef.Number] = true
		rootRef = parentRef
		cur = parentRef
	}
	rootIdent, _ := parse.PlanIdent(rootRef.Title) // rootRef always parses: it's either targetRef or a walked parent that parsed above.

	var nodes []Node
	seen := map[int]bool{}
	add := func(ref gh.SubIssueRef) {
		if seen[ref.Number] {
			return
		}
		seen[ref.Number] = true
		nodes = append(nodes, Node{
			Kind:   kindOf(ref.Title),
			Number: ref.Number,
			Title:  ref.Title,
			State:  ref.State,
			Status: board.NotOnBoard,
		})
	}

	add(rootRef)

	// Step 3 (always happens, regardless of how many children get kept).
	children, err := client.SubIssues(ctx, repo, rootRef.Number)
	if err != nil {
		return nil, fmt.Errorf("issue tree: %w", err)
	}

	if rootIdent.Kind == parse.KindArch {
		// rootRef is a true arch plan: children are impl plans, one level
		// above phases. Scope by target's own Plan/Suffix unless target IS
		// the arch root, in which case every impl child is kept.
		targetIsRoot := rootRef.Number == targetRef.Number
		var retained []gh.SubIssueRef
		for _, child := range children {
			childIdent, ok := parse.PlanIdent(child.Title)
			if !ok {
				// Not a plan issue (e.g. a stray issue manually linked under
				// the arch plan) — the spec says "descend into every impl
				// child", which presupposes the child IS an impl plan.
				continue
			}
			if targetIsRoot {
				retained = append(retained, child)
				continue
			}
			if childIdent.Plan == targetIdent.Plan && childIdent.Suffix == targetIdent.Suffix {
				retained = append(retained, child)
			}
		}
		for _, impl := range retained {
			add(impl)
			phases, perr := client.SubIssues(ctx, repo, impl.Number)
			if perr != nil {
				return nil, fmt.Errorf("issue tree: %w", perr)
			}
			sort.Slice(phases, func(i, j int) bool { return phases[i].Number < phases[j].Number })
			for _, p := range phases {
				add(p)
			}
		}
	} else {
		// rootRef is not an arch plan (an isolated impl/phase root): its
		// SubIssues are leaf phase nodes directly, no further recursion.
		// Non-plan children (a stray issue manually linked under the impl
		// plan) are skipped, matching the arch-branch filtering above.
		var validChildren []gh.SubIssueRef
		for _, child := range children {
			if _, ok := parse.PlanIdent(child.Title); ok {
				validChildren = append(validChildren, child)
			}
		}
		sort.Slice(validChildren, func(i, j int) bool { return validChildren[i].Number < validChildren[j].Number })
		for _, p := range validChildren {
			add(p)
		}
	}

	// Defensive: guarantee target's own node is present even if the walk
	// above somehow never reached it (should not occur with consistent
	// sub-issue data, since rootRef==targetRef or the downward walk from
	// rootRef reaches back to target through its parent chain).
	add(targetRef)

	// 500 rather than the default 100: unlike a direct `board list` call,
	// Tree's caller has no way to raise this limit if a project's board grows
	// past it, so nodes beyond the cap would silently render as
	// board.NotOnBoard instead of their real status.
	statuses, err := board.List(ctx, runner, cfg, "", 500)
	if err != nil {
		return nil, fmt.Errorf("issue tree: %w", err)
	}
	statusByNumber := make(map[int]string, len(statuses))
	for _, item := range statuses {
		statusByNumber[item.Number] = item.Status
	}
	for i := range nodes {
		if s, ok := statusByNumber[nodes[i].Number]; ok {
			nodes[i].Status = s
		}
	}

	return nodes, nil
}
