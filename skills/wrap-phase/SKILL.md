---
name: wrap-phase
description: Capture decisions from a merged phase PR onto the impl plan, close the phase issue, move it to Done, and clean up the worktree
argument-hint: <phase-issue-number> <impl-plan-issue-number>
allowed-tools: Bash, Read, Agent
model: sonnet
---

Run after a phase PR has been merged. Reads the merged PR's history, classifies it into decisions / scope changes / deferred items / corrections, posts a structured wrap-up comment on the impl plan issue, closes the phase issue, moves it to Done on the board, and removes the phase worktree.

**Before starting**: Read `.claude/plan-workflow-config.md` for project configuration and board management commands. Read `../SHARED/GLOSSARY.md` for naming, worktree, and status conventions.

## Arguments

- `$0` — Phase issue number (the `[PLAN-XXXXX-N]` issue)
- `$1` — Impl plan issue number (the parent `[PLAN-XXXXX]` issue)

## Steps

### 1. Locate the merged PR for this phase

```bash
gh pr list --repo <repo> --state merged --search "in:title PLAN-XXXXX-N" \
  --json number,title,url,mergedAt --limit 5
```

Confirm exactly one merged PR matches the phase. If zero, stop: "No merged PR found for phase #$0 — has it been merged yet?". If multiple, ask the user which to use.

Capture the PR number for the next steps.

### 2. Verify the phase issue is still open

```bash
gh issue view $0 --repo <repo> --json state,title
```

If the issue state is already `CLOSED`, ask the user whether to proceed (the PR's `Closes #` may have already auto-closed it) or stop. The wrap-up comment is still useful even if the issue is closed.

### 3. Delegate PR-history analysis to a sub-agent

Read `SUPPLEMENTS/CLASSIFY.md` for the classification rubric and output format.

Spawn a **general-purpose** agent (model **sonnet**, no worktree isolation needed) with this task:

> Classify the history of merged PR #<pr-number> in repo `<repo>` into the four categories defined in `<absolute path to skills/wrap-phase/SUPPLEMENTS/CLASSIFY.md>`. Read that file first for the rubric, then fetch the PR using these commands and analyze its body, review comments, inline comments, and commits:
>
> ```
> gh pr view <pr-number> --repo <repo> --json number,title,body,commits,reviews,comments
> gh api repos/<repo>/pulls/<pr-number>/comments  # inline review comments
> ```
>
> Return a single JSON object with keys `decisions`, `scope_changes`, `deferred`, `corrections`, each mapping to an array of objects per the rubric. Return only the JSON, no commentary.

The sub-agent does its own reading inside its own context window — do not fetch the PR contents in the orchestrator first.

### 4. Render the wrap-up draft

From the sub-agent's JSON, render a markdown comment using the structure in `SUPPLEMENTS/COMMENT_TEMPLATE.md`. Skip any section whose array is empty.

### 5. Present for confirmation

Show the rendered draft to the user. Ask:

> "This will be posted as a comment on impl plan #<impl-plan-issue> and the phase issue will be closed and moved to Done. Edit anything, or proceed?"

Iterate on the draft until the user approves. Do not proceed until explicitly confirmed.

### 6. Post the wrap-up comment on the impl plan issue

```bash
gh issue comment $1 --repo <repo> --body "$(cat <<'EOF'
<approved comment body>
EOF
)"
```

### 7. Close the phase issue (if still open) and move to Done

If the phase issue was open in step 2:

```bash
gh issue close $0 --repo <repo> --reason completed
```

Use the board management commands in `.claude/plan-workflow-config.md` to move issue #$0 to **Done**.

### 8. Remove the phase worktree

The phase worktree was created by `/implement-phase` at `.claude/worktrees/phase-XXXXX-N` and left in place for review. Remove it now:

```bash
git worktree remove --force .claude/worktrees/phase-XXXXX-N
git worktree prune
```

If the path does not exist (manually removed earlier), skip silently.

Do **not** delete the phase branch (`feature/plan-XXXXX-N`) — it remains on the remote as the merge source and is useful for archaeology.

### 9. Confirm

Report:
- Comment URL on impl plan issue
- Phase issue closed and moved to Done
- Worktree removed

If more phases remain: "Next: `/implement-phase <next-phase-issue-number>`"
If this was the last phase: "Next: `/finish-impl <impl-plan-issue-number>`"
