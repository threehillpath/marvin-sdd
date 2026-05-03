---
name: review-impl
description: Comprehensive code-review of a fully-merged implementation before opening the impl PR to main
argument-hint: <impl-plan-issue-number>
allowed-tools: Bash, Read, Agent
model: sonnet
---

Run after all phases of an implementation have been merged into the impl branch, before `/finish-impl` opens the PR to main. Spawns an opus sub-agent (fresh context, extended thinking) that reviews the cumulative impl-branch diff against `main`, treating the impl plan as the spec and the accumulated wrap-phase comments as the record of intentional drift. Catches integration issues that no individual phase review could see.

**Before starting**: Read `.claude/plan-workflow-config.md` for project configuration. Read `../SHARED/GLOSSARY.md` for naming and status conventions.

## Arguments

- `$0` — Impl plan issue number (the parent `[PLAN-XXXXX]` issue)

## Steps

### 1. Verify all phases are merged and the impl branch is up to date

```bash
gh issue view $0 --repo <repo> --json number,title,body,comments
```

Extract PLAN-XXXXX and the phase issue list from the "Phases created:" comment. For each phase:

```bash
gh issue view <phase-issue> --repo <repo> --json state,title
```

If any phase is not `CLOSED`, list the open phases and ask the user whether to proceed anyway (a partial review still has value if a phase was intentionally dropped, but full review-impl assumes completeness).

Confirm the local impl branch is current:

```bash
git branch --show-current   # must be feature/plan-XXXXX
git pull origin feature/plan-XXXXX
git status                  # must be clean
```

### 2. Spawn the review sub-agent

Spawn an **Agent** with:

- `subagent_type: "general-purpose"`
- `model: "opus"`
- No worktree isolation (review is read-only)

Prompt template:

> You are reviewing a complete implementation prior to its PR-to-main. Your output is a single JSON object per the format in `<absolute-path-to>/skills/SHARED/REVIEW_FINDING_FORMAT.md`. Read that file first, then read `<absolute-path-to>/skills/SHARED/REVIEW_RUBRIC.md` for the rubric. The `integration` category is in scope for this review.
>
> **Implementation branch**: `feature/plan-XXXXX` in repo `<repo>`. Compare against `main`.
> **Impl plan (the spec)**: #$0 in repo `<repo>`. Read the body for component sections, success criteria, and TDD entry points.
> **Phase wrap-up comments**: comments on issue #$0 posted by `/wrap-phase`. These record decisions, scope changes, deferred items, and corrections. Drift recorded here is **not** a finding — it is the legitimate channel.
>
> Fetch the inputs you need with these commands (run them yourself):
>
> ```
> gh issue view $0 --repo <repo> --json title,body,comments
> gh api repos/<repo>/compare/main...feature/plan-XXXXX  # commit list and aggregate diff
> git fetch origin main feature/plan-XXXXX
> git diff origin/main...origin/feature/plan-XXXXX        # full cumulative diff
> git log origin/main..origin/feature/plan-XXXXX --oneline
> ```
>
> Read the cumulative diff. Review by component sections from the impl plan, not by phase — the goal is to catch things that span phases. Pay particular attention to:
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

Ask: **"Post these findings on impl plan issue #$0, or save locally only? (post / save / cancel)"**

- `post` — proceed to step 5 to comment on the issue.
- `save` — skip the comment; just write the findings JSON. Useful when the user wants to review privately before deciding what to act on.
- `cancel` — stop. Discard findings.

### 5. Post the findings on the impl plan issue

There is no PR yet (the impl PR is opened by `/finish-impl`), so findings are posted as a comment on the impl plan issue rather than as a PR review.

```bash
gh issue comment $0 --repo <repo> --body "$(cat <<'EOF'
## Pre-PR Review (`/review-impl`)

**Verdict**: <verdict>

<summary>

### Blocking (<count>)

- **[B1]** <category> — `<file>:<line>` — <summary>
  <details>
  *Suggested fix*: <suggested_fix>

- **[B2]** ...

### Nits (<count>)

- **[N1]** <category> — `<file>:<line>` — <summary>
  *Suggested fix*: <suggested_fix>

- ...
EOF
)"
```

If `blocking` is non-empty, the comment includes a clear marker so a future automation pass can detect "this impl is not yet ready for the impl PR."

### 6. Save the findings JSON

```bash
mkdir -p .claude/reviews
echo "<findings JSON>" > .claude/reviews/impl-XXXXX.json
```

### 7. Confirm and direct next step

Report:
- Comment URL (if posted)
- Verdict
- Blocking count, nit count
- Findings JSON path

Direct the user:

- If `verdict` is `request-changes`: "Address findings on the impl branch (either as new commits to `feature/plan-XXXXX` directly, or by opening a follow-up phase if the work is large), then re-run `/review-impl $0` for a fresh pass. Do not run `/finish-impl` until blocking findings are resolved or explicitly waived."
- If `verdict` is `approve` or `comment`: "Ready to open the impl PR. Next: `/finish-impl $0`."
