---
name: review-phase
description: Code-review a phase PR with a fresh-context opus sub-agent and post the review to GitHub
argument-hint: <phase-issue-number>
allowed-tools: Bash, Read, Agent
model: sonnet
---

Run after `/implement-phase` opens a phase PR, before merging. Spawns an opus sub-agent (fresh context, extended thinking) that reads the phase spec and the PR diff, applies the rubric in `../SHARED/REVIEW_RUBRIC.md`, and returns structured findings per `../SHARED/REVIEW_FINDING_FORMAT.md`. Orchestrator presents findings, then on approval posts a single GitHub PR review.

**Before starting**: Read `.claude/plan-workflow-config.yml` for project configuration. Read `../SHARED/GLOSSARY.md` for naming and status conventions.

## Arguments

- `$0` — Phase issue number (the `[PLAN-XXXXX-N]` issue)

## Steps

### 1. Locate the open PR for this phase

```bash
gh issue view $0 --repo <repo> --json number,title
marvin pr find "[PLAN-XXXXX-N]" --state open
```

First fetch the issue title to extract the `[PLAN-XXXXX-N]` ident, then call `marvin pr find` with that ident. The output includes `found:`, `number:`, `url:`, `head:`, `base:`, and `state:` lines.

If `found` is `false`, stop: "No open PR found for phase #$0 — has `/implement-phase` opened the PR yet?". If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

Capture the PR number, head branch, and base branch directly from the `marvin pr find` output's `head:` and `base:` lines — no need to reconstruct either name, since the PR already exists and carries its real branch names (this also sidesteps having to know `<type>`, which nothing in this skill resolves).

### 2. Fetch the phase spec for the reviewer

The sub-agent will read the spec itself. The orchestrator only confirms the issue exists and captures the impl plan issue number for cross-reference:

```bash
gh issue view $0 --repo <repo> --json number,title,body
```

Extract the impl plan issue number from the phase issue body (referenced as `Implementation plan: #<n>`). The reviewer will use this to look up cross-phase context if needed.

### 2b. Deterministic structural pre-check (skill-prose-only phases)

If the phase issue's **TDD Entry Point** section says "None" (skill prose, no executable surface — no Go/test diff to build or run), semantic review from a sub-agent is the *only* verification this PR would otherwise get. That's a correctness gap: an opus review is judgment, not a guarantee, and skill-prose mistakes are often purely mechanical (transposed arguments, a forgotten trailing character in a string literal, an old invocation left behind at one of several call sites during a migration) — exactly the kind of thing a deterministic grep catches with certainty and a semantic read can miss.

Before spawning the sub-agent, run this pre-check:

1. Fetch the diff: `gh pr diff <pr-number> --repo <repo>`.
2. From the phase issue body and the impl plan's referenced component sections, extract the **literal, load-bearing patterns the spec itself calls out explicitly** — exact command invocations, exact flag/string literals, and any prose the spec marks as precise or load-bearing (e.g. "no closing bracket", "trailing hyphen", a specific argument order). Do not invent patterns beyond what the spec states in concrete terms — this step verifies literal text, not intent.
3. For each pattern, grep the diff (or the changed files directly) and record a pass/fail:
   - **Must appear** — the new invocation/string, expected once per call site the spec names.
   - **Must NOT appear** — old/stale text the spec explicitly says is being replaced (e.g. the previous command form, a superseded flag value).
4. Render a short table, e.g.:
   ```
   Structural pre-check:
     [PASS] `marvin issue link-parent` present in skills/impl-plan/SKILL.md
     [PASS] `marvin issue link-parent` present in skills/phase-split/SKILL.md
     [PASS] `--title-prefix "[PLAN-XXXXX-"` (trailing hyphen, no closing bracket) present in skills/start-impl/SKILL.md
     [FAIL] stale `--title-prefix "[PLAN-XXXXX]"` (closing bracket) still present in skills/review-impl/SKILL.md
   ```
5. This pre-check is deterministic *input* to the sub-agent's review, not a replacement for it — sequencing, placement, and whether the change actually matches the spec's intent still need judgment. Pass the table to the sub-agent (step 3) as pre-verified ground truth so it doesn't re-derive what grep already settled, and can spend its judgment on what grep can't check. Any `FAIL` is reported to the user in step 5/7 regardless of what the sub-agent's own findings say — a structural fail is never silently absorbed into the sub-agent's verdict.

Skip this step entirely for phases with a real TDD entry point (Go/test diff) — the build and test suite already provide deterministic verification there.

### 3. Spawn the review sub-agent

Spawn an **Agent** with:

- `subagent_type: "general-purpose"`
- `model: "opus"`
- No worktree isolation (review is read-only)

Assemble the prompt by **referencing** the inputs the sub-agent should fetch — do not paste the diff into the prompt. The sub-agent has its own context window.

Prompt template:

> You are reviewing a phase PR for the plan-workflow plugin. Your output is a single JSON object per the format in `<absolute-path-to>/skills/SHARED/REVIEW_FINDING_FORMAT.md`. Read that file first, then read `<absolute-path-to>/skills/SHARED/REVIEW_RUBRIC.md` for the rubric. Apply the rubric to the diff under review.
>
> **PR to review**: #<pr-number> in repo `<repo>`. Head: `<head-ref>`. Base: `<base-ref>`.
> **Phase issue (the spec)**: #$0 in repo `<repo>`. Read the body for objective, success criteria, TDD entry point, and any constraints.
> **Impl plan (parent context)**: #<impl-plan-issue> in repo `<repo>`. Read the relevant component sections only if the phase issue references them.
> **Pre-verified structural facts** (skill-prose-only phases; omit this block if step 2b was skipped): the following literal patterns were already grep-confirmed present/absent in the diff — treat these as settled, do not re-derive them, and spend your judgment on sequencing/placement/semantic correctness instead:
> ```
> <step 2b's pass/fail table, verbatim>
> ```
> Any `FAIL` above is a confirmed defect — include it in your findings at `blocking` severity regardless of how the surrounding prose reads; do not downgrade a structural fail based on context.
>
> Fetch the inputs you need with these commands (run them yourself):
>
> ```
> gh issue view $0 --repo <repo> --json title,body
> gh issue view <impl-plan-issue> --repo <repo> --json title,body
> gh pr view <pr-number> --repo <repo> --json title,body,commits,files
> gh pr diff <pr-number> --repo <repo>
> gh api repos/<repo>/pulls/<pr-number>/comments    # any prior inline comments
> ```
>
> Read the diff fully. For files where the diff is large or context-dependent, read the surrounding source via the Read tool to understand the change. Do not review pre-existing code outside the diff.
>
> Apply the rubric category by category. For each finding, populate every field in the Finding schema. Use `evidence` to cite a commit SHA, a quoted line, or a diff hunk so the user can verify quickly.
>
> Use extended thinking on this task. Take the time to trace data paths through handlers, verify exports against barrel files, check declaration order, and walk the diff against the success criteria. The cost of a missed blocking issue here is much higher than the cost of a longer review.
>
> Return **only** the JSON object specified in `REVIEW_FINDING_FORMAT.md`. No surrounding prose, no code fence.

The sub-agent reads the rubric, the spec, and the diff in its own context window. Do not pre-fetch any of these in the orchestrator.

### 4. Parse and validate the response

Parse the returned JSON. Validate:

- The shape matches `REVIEW_FINDING_FORMAT.md` (top-level keys: `summary`, `verdict`, `blocking`, `nits`).
- `verdict` is consistent with the arrays — `approve` requires both empty, `request-changes` requires `blocking` non-empty, `comment` requires `blocking` empty and `nits` non-empty.
- Every finding has all required fields and a recognized `category`.
- If step 2b produced any `FAIL` row, at least one `blocking` finding corresponds to it. If the sub-agent's response omits a finding for a confirmed structural `FAIL`, this is a validation failure — the deterministic check is ground truth and cannot be silently dropped by the semantic pass.

If validation fails, summarize the problem and re-spawn the sub-agent once with feedback. After two failures, stop and show the user the raw response — but still report step 2b's table as-is regardless (it does not depend on the sub-agent succeeding).

### 5. Render the findings for review

Present a human-readable summary to the user. If step 2b ran, show its table first, unconditionally — even if every row passed, even if the sub-agent's own findings already cover the same ground:

```
Review of PR #<n> — <title>

<step 2b's structural pre-check table, if it ran>

Verdict: <verdict>

<summary paragraph from response>

Blocking (<count>):
  [B1] <category> — <file>:<line>
       <summary>
  [B2] ...

Nits (<count>):
  [N1] <category> — <file>:<line>
       <summary>
  ...
```

Then ask: **"Post this as a GitHub PR review on #<pr-number>? (yes / edit / cancel)"**

- `yes` — proceed to step 6.
- `edit` — show the full JSON; let the user remove or downgrade specific findings (by id). Re-render the summary after edits.
- `cancel` — stop. Do not post anything. The findings JSON is discarded.

### 6. Post the review to GitHub

Build a `gh pr review` call. Inline comments are posted via the GitHub API since `gh pr review` itself does not support per-line comments.

For the review body (top-level summary):

```bash
REVIEW_BODY="$(cat <<'EOF'
<summary paragraph>

Findings recorded inline. See <count> blocking and <count> nit comments below.
EOF
)"
```

For each finding, post an inline comment via the API. Use the head SHA from the PR:

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

After all inline comments are posted, submit the top-level review:

```bash
case "$VERDICT" in
  approve)         EVENT=APPROVE ;;
  request-changes) EVENT=REQUEST_CHANGES ;;
  comment)         EVENT=COMMENT ;;
esac

gh pr review <pr-number> --repo <repo> --$EVENT --body "$REVIEW_BODY"
```

If any inline comment POST fails (e.g. line is outside the diff), fall back to including that finding in the top-level review body under a "Findings without anchorable line" section. Do not stop the rest of the run for one bad anchor.

### 7. Confirm

Report:
- Structural pre-check result, if step 2b ran (pass/fail count)
- Review URL (link to the GitHub review)
- Verdict
- Blocking count, nit count

If verdict is `request-changes`: "Address findings, push corrections, then re-run `/review-phase $0` for a fresh pass — or proceed to merge if you disagree with a finding."
If verdict is `approve` or `comment`: "Ready to merge when you are. After merge: `/wrap-phase $0 <impl-plan-issue>`."
