---
name: review-impl
description: Comprehensive code-review of the impl PR (feature/plan-XXXXX → main) after it is opened by finish-impl
argument-hint: <impl-plan-issue-number>
allowed-tools: Bash, Read, Agent
model: sonnet
---

Run after `/finish-impl` has opened the impl PR to main. Spawns an opus sub-agent (fresh context, extended thinking) that reviews the cumulative impl-branch diff against `main`, treating the impl plan as the spec and the accumulated wrap-phase comments as the record of intentional drift. Posts findings as a GitHub PR review. Catches integration issues that no individual phase review could see.

**Before starting**: Read `.claude/plan-workflow-config.md` for project configuration. Read `../SHARED/GLOSSARY.md` for naming and status conventions.

## Arguments

- `$0` — Impl plan issue number (the parent `[PLAN-XXXXX]` issue)

## Steps

### 1. Locate the open impl PR and verify all phases are merged

```bash
gh issue view $0 --repo <repo> --json number,title,body,comments
```

Extract PLAN-XXXXX and the phase issue list from the "Phases created:" comment. For each phase:

```bash
gh issue view <phase-issue> --repo <repo> --json state,title
```

If any phase is not `CLOSED`, list the open phases and ask the user whether to proceed anyway.

Locate the open impl PR:

```bash
gh pr list --repo <repo> --state open --search "in:title PLAN-XXXXX" \
  --json number,title,url,headRefName,baseRefName --limit 5
```

Confirm exactly one open PR exists targeting `main`. If zero, stop: "No open impl PR found — run `/finish-impl $0` first." If multiple, ask the user which to review. Capture the PR number and head ref.

Confirm the local impl branch is current:

```bash
git fetch origin feature/plan-XXXXX
```

### 2. Spawn the review sub-agent

Spawn an **Agent** with:

- `subagent_type: "general-purpose"`
- `model: "opus"`
- No worktree isolation (review is read-only)

Prompt template:

> You are reviewing an implementation PR before it merges to main. Your output is a single JSON object per the format in `<absolute-path-to>/skills/SHARED/REVIEW_FINDING_FORMAT.md`. Read that file first, then read `<absolute-path-to>/skills/SHARED/REVIEW_RUBRIC.md` for the rubric. The `integration` category is in scope for this review.
>
> **PR to review**: #<pr-number> in repo `<repo>`. Head: `feature/plan-XXXXX`. Base: `main`.
> **Impl plan (the spec)**: #$0 in repo `<repo>`. Read the body for component sections, success criteria, and TDD entry points.
> **Phase wrap-up comments**: comments on issue #$0 posted by `/wrap-phase`. These record decisions, scope changes, deferred items, and corrections. Drift recorded here is **not** a finding — it is the legitimate channel.
>
> Fetch the inputs you need with these commands (run them yourself):
>
> ```
> gh issue view $0 --repo <repo> --json title,body,comments
> gh pr view <pr-number> --repo <repo> --json title,body,commits,files
> gh pr diff <pr-number> --repo <repo>
> git fetch origin main feature/plan-XXXXX
> git log origin/main..origin/feature/plan-XXXXX --oneline
> ```
>
> Read the diff fully. Review by component sections from the impl plan, not by phase — the goal is to catch things that span phases. Pay particular attention to:
>
> - **Integration**: cross-phase consistency, deferred items not picked up, migration ordering, two phases adding similar helpers, schema columns added by one phase and not used by another.
> - **Spec drift not in wrap-up**: divergence from the impl plan that is not documented in any phase wrap-up comment.
> - **Self-review at the impl level**: missing exports/registrations that a phase missed because the registration site lives outside the phase's scope.
> - **Standard rubric categories**: tdd, correctness, security across the cumulative change.
>
> Use extended thinking. The cumulative diff is larger than a single phase's; trace data paths end-to-end across phases, verify barrel files contain exports added by every phase, walk migration steps in commit order to confirm they apply cleanly.
>
> Return **only** the JSON object specified in `REVIEW_FINDING_FORMAT.md`. No surrounding prose, no code fence.

### 3. Parse and validate the response

Same validation as `review-phase` step 4. The `integration` category is valid in this skill's output (it is not valid in `review-phase`).

### 4. Render the findings for review

Present a human-readable summary, grouped by category since cross-cutting findings benefit from grouping:

```
Review of implementation feature/plan-XXXXX (vs main)

Verdict: <verdict>

<summary paragraph>

Blocking (<count>):
  Integration:
    [B1] <file>:<line> — <summary>
  TDD:
    [B2] <file>:<line> — <summary>
  ...

Nits (<count>):
  ...
```

Ask: **"Post this as a GitHub PR review on #<pr-number>? (yes / edit / cancel)"**

- `yes` — proceed to step 5.
- `edit` — show the full JSON; let the user remove or downgrade specific findings. Re-render after edits.
- `cancel` — stop. Do not post anything.

### 5. Post the review to GitHub

Post inline comments and a top-level review using the same pattern as `review-phase` step 6.

For each finding, post an inline comment via the API using the head SHA:

```bash
HEAD_SHA=$(gh pr view <pr-number> --repo <repo> --json headRefOid --jq '.headRefOid')

gh api -X POST repos/<repo>/pulls/<pr-number>/comments \
  -f commit_id="$HEAD_SHA" \
  -f path="<file>" \
  -F line=<line> \
  -f side="RIGHT" \
  -f body="$(cat <<'EOF'
**[<id>] <category> — <severity>**

<summary>

<details>

**Suggested fix**: <suggested_fix>

_Evidence_: <evidence>
EOF
)"
```

If a line is outside the diff, include that finding in the top-level review body under "Findings without anchorable line" instead. Do not stop for one bad anchor.

Submit the top-level review:

```bash
case "$VERDICT" in
  approve)         EVENT=APPROVE ;;
  request-changes) EVENT=REQUEST_CHANGES ;;
  comment)         EVENT=COMMENT ;;
esac

gh pr review <pr-number> --repo <repo> --$EVENT --body "$(cat <<'EOF'
<summary paragraph>

Findings recorded inline. See <count> blocking and <count> nit comments below.
EOF
)"
```

Note: if the repo owner is the same user running the review, GitHub will reject `APPROVE` — use `COMMENT` instead.

### 6. Save the findings JSON

```bash
mkdir -p .claude/reviews
echo "<findings JSON>" > .claude/reviews/impl-XXXXX.json
```

### 7. Confirm and direct next step

Report:
- Review URL (link to the GitHub review)
- Verdict
- Blocking count, nit count
- Findings JSON path

Direct the user:

- If `verdict` is `request-changes`: "Address findings on the impl branch (either as new commits to `feature/plan-XXXXX` directly, or by opening a follow-up phase if the work is large), then re-run `/review-impl $0` for a fresh pass."
- If `verdict` is `approve` or `comment`: "Ready to merge when you are. After merge, the impl plan issue will auto-close via `Closes #$0` in the PR body — move it to Done on the board manually if needed."
