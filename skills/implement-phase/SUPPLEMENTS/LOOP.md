# TDD Implementation Loop — Sub-agent Instructions

This document is assembled into the sub-agent prompt by `implement-phase`. The sub-agent operates autonomously in an isolated worktree.

## 1. Setup

Fetch the remote and create the phase branch from the implementation branch:

```bash
git fetch origin
git checkout -b <phase-branch> origin/<impl-branch>
```

Confirm you are on `<phase-branch>` before making any changes:

```bash
git branch --show-current
```

Read all source files relevant to this phase before writing any code. Understand the existing structure fully before changing it.

## 2. The TDD loop

Work through the phase success criteria **one at a time, in order**.

### Backend and non-UI frontend logic (reducers, utilities, validation helpers)

**CRITICAL: Work strictly one criterion at a time. Do not write tests for future criteria or implement beyond the current criterion. Complete the full loop (test → red → implement → green) for criterion N before touching criterion N+1.**

For each criterion:

1. **Write one failing test** — and only one. Use the TDD entry point from the phase issue as the anchor for the first criterion. Each subsequent criterion gets its own targeted test asserting the outermost observable behavior — what a caller sees, not internal state. Do not write tests for any other criterion at this step.
2. **Run the test suite. Confirm red.** Use the test command from `.claude/plan-workflow-config.md` (run from repo root). The new test must fail. If it passes immediately, the behavior already exists — note it, skip implementation, move to the next criterion.
3. **Write the minimum code to make the test pass.** Do not implement beyond what the current test requires. Do not add code in anticipation of future criteria.
4. **Run the test suite. Confirm green.** All previously passing tests must still pass.
5. If still failing after implementation: diagnose, fix, re-run — up to 3 attempts total.
6. Only after green: move to the next criterion and repeat from step 1.

### Rendered UI components (DOM-dependent code)

Implement the changes. Note what manual verification steps apply. No automated test is required.

## 3. Failure handling

**Test fails after 3 fix attempts**: Stop the loop. Return a summary to the caller:
- Which criterion failed
- What the test asserts
- Actual vs. expected output
- What was attempted
Do not make further changes.

**Repeated stalling**: If you have hit the 3-attempt limit on more than one criterion, or the same category of failure keeps recurring across different criteria, include this note in your escalation report:
> "Progress has stalled on multiple criteria. Consider re-running `/implement-phase` with the model upgraded to opus for better reasoning on this phase."

**Requirement ambiguity** (phase issue and impl plan conflict, or a requirement is underspecified): Stop the loop. Describe the ambiguity specifically. Return to the caller for input.

**Unexpected existing behavior** (a change breaks unrelated tests): Stop and diagnose before proceeding. Do not suppress or delete failing tests to make the suite pass.

## 4. Finishing

When all success criteria are implemented and passing:

### Stage and commit

Stage only the files you created or modified during this phase. Do not stage untracked files you did not intentionally create.

```bash
git add <each modified or created file by path>
git commit -m "[PLAN-XXXXX-N] <phase title>"
```

### Push

```bash
git push -u origin <phase-branch>
```

### Create PR

Target the implementation branch — **not main**.

Use the following PR body structure:

```markdown
## Summary

<2-4 bullet points describing what this phase delivers. Focus on behavior, not file changes.>

## Phase

Closes #<phase-issue-number> — [PLAN-XXXXX-N] <Phase Title>

**Implementation plan**: #<impl-plan-issue>

## Test plan

<Bulleted checklist of what to verify before merging. Include the TDD entry point test if this is a backend phase. UI phases should describe manual verification steps.>

- [ ] <verification step>
- [ ] <verification step>

## Notes

<Optional: anything a reviewer needs to know — migration steps, env var changes, known limitations.>
```

```bash
gh pr create --repo <repo> \
  --title "[PLAN-XXXXX-N] <Phase Title>" \
  --base <impl-branch> \
  --body "$(cat <<'EOF'
<PR body per template above>
EOF
)"
```

The PR body must include `Closes #<phase-issue-number>`.

### Move phase issue to In Review

Use the board management commands from `.claude/plan-workflow-config.md`. Use `--jq '.id'` when capturing the item ID from `item-add` — do not pipe the response to shell `jq`.

### Return summary

Report back to the `implement-phase` caller:
- PR URL
- Test results (pass count, any skipped)
- List of success criteria completed
- Anything requiring manual verification (UI steps)

## 5. Operating mode

You are running autonomously. Do not pause for user confirmation at individual steps. Only stop and return to the caller on unresolvable failure or ambiguity.
