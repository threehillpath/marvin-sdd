package issue_test

import (
	"context"
	"fmt"
	"testing"

	"threehillpath.com/claude-plan-workflow/tool/internal/exectest"
	"threehillpath.com/claude-plan-workflow/tool/internal/issue"
)

// issueRefResponse builds a canned `gh issue view --json id,number,title,state` response.
func issueRefResponse(id string, number int, title, state string) string {
	return fmt.Sprintf(`{"id":%q,"number":%d,"title":%q,"state":%q}`, id, number, title, state)
}

// TestTreePlan00036Regression is TDD Entry Point 1: given a SubIssues response
// with two phase issues whose titles do NOT share a true string-prefix with
// the impl plan's own title (the historical --title-prefix bug), issue.Tree
// still returns both phases, since it resolves the hierarchy via real
// GitHub sub-issue links (IssueRef/ParentIssue/SubIssues), not string matching.
func TestTreePlan00036Regression(t *testing.T) {
	fake := &exectest.FakeRunner{}

	// implTitle and the phase titles below deliberately do NOT share a true
	// string-prefix: "[PLAN-00036-1]" is not a prefix of "[PLAN-00036]"
	// (the bracket closes differently), which is exactly the historical
	// --title-prefix bug case from arch plan #56's Problem Statement.
	// issue.Tree resolves the hierarchy via real sub-issue links instead of
	// string matching, so both phases must still be found.
	implTitle := "[PLAN-00036] Fix the marvin hierarchy bug"
	phase1Title := "[PLAN-00036-1] Totally different wording"
	phase2Title := "[PLAN-00036-2] Yet another unrelated title"

	// 1) IssueRef(target=36)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(issueRefResponse("ID_36", 36, implTitle, "OPEN"))})
	// 2) ParentIssue(36): internal IssueRef(36)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(issueRefResponse("ID_36", 36, implTitle, "OPEN"))})
	// 2) ParentIssue(36): graphql parent query -> no parent
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"data":{"node":{"parent":null}}}`)})
	// 3) SubIssues(archRoot=36): internal IssueRef(36)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(issueRefResponse("ID_36", 36, implTitle, "OPEN"))})
	// 3) SubIssues(36): graphql subIssues query -> two phases, titles that do NOT
	//    share a string-prefix with implTitle (the PLAN-00036 regression case).
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(
		`{"data":{"node":{"subIssues":{"nodes":[` +
			`{"number":39,"title":"` + phase1Title + `","state":"OPEN"},` +
			`{"number":40,"title":"` + phase2Title + `","state":"OPEN"}` +
			`]}}}}`,
	)})
	// 5) board.List
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"items":[]}`)})

	cfg := buildConfig()

	nodes, err := issue.Tree(context.Background(), fake, cfg, cfg.Repo, 36)
	if err != nil {
		t.Fatalf("Tree returned error: %v", err)
	}

	var phaseNumbers []int
	for _, n := range nodes {
		if n.Kind == "phase" {
			phaseNumbers = append(phaseNumbers, n.Number)
		}
	}

	if len(phaseNumbers) != 2 {
		t.Fatalf("expected 2 phase nodes, got %d: %v", len(phaseNumbers), nodes)
	}
	found39, found40 := false, false
	for _, n := range phaseNumbers {
		if n == 39 {
			found39 = true
		}
		if n == 40 {
			found40 = true
		}
	}
	if !found39 || !found40 {
		t.Errorf("expected phases #39 and #40 in result, got: %v", phaseNumbers)
	}
}

// TestTreeMultiImplScoping is TDD Entry Point 2: given a target impl plan
// with Suffix "B", whose arch-plan parent's SubIssues response contains BOTH
// impl children (Suffix "A" and Suffix "B"), issue.Tree's result contains
// only track B's phases and excludes every track-A node. Track A's data is
// present in the fake's response queue so its absence from the result is a
// real assertion, not an artifact of never having been queried. The fake is
// also enqueued with exactly the responses a correctly-scoped walk needs —
// an unscoped implementation that also queries track A's SubIssues would
// exhaust the queue and fail with "no response queued", not just produce
// extra nodes.
func TestTreeMultiImplScoping(t *testing.T) {
	fake := &exectest.FakeRunner{}

	archTitle := "[PLAN-00057-ARCH] Multi-impl arch plan"
	implATitle := "[PLAN-00057-A] Track A impl plan"
	implBTitle := "[PLAN-00057-B] Track B impl plan"
	phaseB1Title := "[PLAN-00057-B-1] Track B phase 1"
	phaseB2Title := "[PLAN-00057-B-2] Track B phase 2"

	// 1) IssueRef(target=58, track B impl plan)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(issueRefResponse("ID_58", 58, implBTitle, "OPEN"))})
	// 2) ParentIssue(58): internal IssueRef(58)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(issueRefResponse("ID_58", 58, implBTitle, "OPEN"))})
	// 2) ParentIssue(58): graphql parent query -> arch #57
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"data":{"node":{"parent":{"number":57,"title":"` + archTitle + `","state":"OPEN"}}}}`)})
	// 2) ParentIssue(57): internal IssueRef(57)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(issueRefResponse("ID_57", 57, archTitle, "OPEN"))})
	// 2) ParentIssue(57): graphql parent query -> no parent (arch is the root)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"data":{"node":{"parent":null}}}`)})
	// 3) SubIssues(archRoot=57): internal IssueRef(57)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(issueRefResponse("ID_57", 57, archTitle, "OPEN"))})
	// 3) SubIssues(57): graphql subIssues query -> BOTH impl children, A and B.
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(
		`{"data":{"node":{"subIssues":{"nodes":[` +
			`{"number":60,"title":"` + implATitle + `","state":"OPEN"},` +
			`{"number":58,"title":"` + implBTitle + `","state":"OPEN"}` +
			`]}}}}`,
	)})
	// 4) SubIssues(implB=58) only — track A's SubIssues must NEVER be queried:
	//    internal IssueRef(58)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(issueRefResponse("ID_58", 58, implBTitle, "OPEN"))})
	// 4) SubIssues(58): graphql subIssues query -> track B's phases
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(
		`{"data":{"node":{"subIssues":{"nodes":[` +
			`{"number":61,"title":"` + phaseB1Title + `","state":"OPEN"},` +
			`{"number":62,"title":"` + phaseB2Title + `","state":"OPEN"}` +
			`]}}}}`,
	)})
	// 5) board.List
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"items":[]}`)})

	cfg := buildConfig()

	nodes, err := issue.Tree(context.Background(), fake, cfg, cfg.Repo, 58)
	if err != nil {
		t.Fatalf("Tree returned error: %v", err)
	}

	numbers := map[int]bool{}
	for _, n := range nodes {
		numbers[n.Number] = true
	}

	// Track A (impl #60) must be entirely excluded.
	if numbers[60] {
		t.Errorf("expected track A's impl node (#60) to be excluded, got nodes: %v", nodes)
	}
	// Track B's own nodes must be present: the impl plan itself and both phases.
	for _, want := range []int{58, 61, 62} {
		if !numbers[want] {
			t.Errorf("expected node #%d in result, got nodes: %v", want, nodes)
		}
	}
	if len(nodes) != 4 { // arch(57) + implB(58) + phaseB1(61) + phaseB2(62)
		t.Errorf("expected 4 nodes (arch, implB, 2 phases), got %d: %v", len(nodes), nodes)
	}
}
