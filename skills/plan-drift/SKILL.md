---
name: plan-drift
description: Audit a phase branch for coverage and scope drift against its spec — runs mid-implementation or before merge
argument-hint: <phase-issue-number>
allowed-tools: Bash, Read, Agent
model: sonnet
---

Run any time during a phase to check whether the current diff matches the phase spec — both *coverage* (each success criterion satisfied) and *containment* (no out-of-scope or undocumented changes). Spawns a fresh-context sub-agent that reads the spec, walks the diff against it, and returns structured findings per `../SHARED/PLAN_DRIFT_FORMAT.md`. The orchestrator presents the report and (optionally) posts a comment on the phase issue or PR.

This skill complements `review-phase` — it does *not* replace it. `review-phase` runs once at PR time and applies the full code-review rubric. `plan-drift` runs on demand, focuses only on drift, and is most valuable before the PR is opened so the implementer can correct course early.

**Before starting**: Read `.claude/plan-workflow-config.md` for project configuration. Read `../SHARED/GLOSSARY.md` for branch, worktree, and status conventions.

## Arguments

- `$0` — Phase issue number (the `[PLAN-XXXXX-N]` issue)

## Steps

### 1. Fetch context

```bash
gh issue view $0 --repo <repo> --json number,title,body,labels
```

Verify the issue is a phase issue (title matches `[PLAN-XXXXX-N] …`). Extract the impl plan issue number (referenced as `Implementation plan: #<n>`).

Extract the plan identifier and phase number from the issue title:

```bash
marvin parse title "<issue title>"
```

Read the `plan` field (an integer, e.g. `42`) and `phase` field (e.g. `3`) from the JSON output. Format the plan number as `plan-XXXXX` — lowercase, 5-digit zero-padded (e.g. `plan-00042`) — and use that string wherever `<plan>` appears in subsequent steps. The phase identifier for cache naming is formed as `phase-XXXXX-N` (e.g. `phase-00042-3`).

### 2. Determine the source of truth for the diff

Check whether a phase PR exists:

```bash
marvin pr find "[PLAN-XXXXX-N]" --state open
```

Branch off the result (`found` field in the JSON):

- **`found: true`** — record the `number` from the JSON. The head branch is `feature/plan-XXXXX-N` (from step 1's parse output) and the base branch is the `base` field already included in the `marvin pr find` result — no extra call needed. The sub-agent will use `gh pr diff <pr-number>`.
- **`found: false`** — the phase work lives in the local worktree at `.claude/worktrees/phase-XXXXX-N`. Verify the worktree exists:
  ```bash
  test -d .claude/worktrees/phase-XXXXX-N && echo present || echo missing
  ```
  - If present, the sub-agent will use `git -C <worktree> diff feature/plan-XXXXX...HEAD` against the impl branch.
  - If missing, stop: "No open PR for phase #$0 and no worktree at `.claude/worktrees/phase-XXXXX-N`. Run `/implement-phase $0` first, or push the phase branch and re-run."

If multiple PRs match, ask the user which to audit.

### 3. Spawn the drift sub-agent

Spawn an **Agent** with:

- `subagent_type: "general-purpose"`
- `model: "sonnet"`
- No worktree isolation (drift check is read-only)

Assemble the prompt by **referencing** the inputs the sub-agent should fetch — do not paste the spec or diff into the prompt. The sub-agent has its own context window.

Prompt template (PR-mode variant):

> You are auditing a phase branch for coverage and scope drift against its spec. Your output is a single JSON object per the format in `<absolute-path-to>/skills/SHARED/PLAN_DRIFT_FORMAT.md`. Read that file first, then read `<absolute-path-to>/skills/SHARED/PLAN_DRIFT_RUBRIC.md` for the rubric and stance. Apply the rubric to the phase under audit.
>
> **Phase issue (the spec)**: #$0 in repo `<repo>`. Read the body for objective, scope, success criteria, and TDD entry point.
> **Impl plan (parent context)**: #<impl-plan-issue> in repo `<repo>`. Read for the relevant component spec sections only — do not audit unrelated phases.
> **PR under audit**: #<pr-number> in repo `<repo>`. Head: `<head-ref>`. Base: `<base-ref>`.
>
> Fetch the inputs you need with these commands (run them yourself):
>
> ```
> gh issue view $0 --repo <repo> --json title,body
> gh issue view <impl-plan-issue> --repo <repo> --json title,body
> gh pr view <pr-number> --repo <repo> --json title,body,commits,files
> gh pr diff <pr-number> --repo <repo>
> ```
>
> Pay particular attention to the PR body's Notes section — drift documented there is **not** a finding.
>
> Read the diff fully. For files where the diff is large or context-dependent, read the surrounding source via the Read tool to verify whether a criterion is actually satisfied (the test exists *and* exercises the behavior; the export exists *and* is reachable from callers). Do not audit pre-existing code outside the diff.
>
> Use extended thinking. Walk every success criterion against the diff. For each, name the evidence that would falsify a regression — if you cannot, the status is not `met`.
>
> Apply the rubric category by category. For each finding, populate every field in the Finding schema. Use `evidence` to cite a commit SHA, a quoted line, or a diff hunk so the user can verify quickly.
>
> Return **only** the JSON object specified in `PLAN_DRIFT_FORMAT.md`. No surrounding prose, no code fence.

Prompt template (worktree-mode variant — only the PR block changes):

> **Working tree under audit**: `<absolute-path-to-worktree>` (branch `feature/plan-XXXXX-N`, based on `feature/plan-XXXXX`). There is no PR yet, so there is no Notes section to consult — any divergence is a candidate finding.
>
> Fetch the diff with:
>
> ```
> git -C <absolute-path-to-worktree> diff feature/plan-XXXXX...HEAD
> git -C <absolute-path-to-worktree> diff --name-only feature/plan-XXXXX...HEAD
> git -C <absolute-path-to-worktree> log feature/plan-XXXXX..HEAD --oneline
> ```
>
> Read source files inside the worktree path when you need surrounding context.

The sub-agent reads the rubric, the spec, and the diff in its own context window. Do not pre-fetch any of these in the orchestrator beyond what step 1 already did.

### 4. Parse and validate the response

Parse the returned JSON. Validate:

- The shape matches `PLAN_DRIFT_FORMAT.md` (top-level keys: `summary`, `verdict`, `criteria`, `blocking`, `concerns`).
- `verdict` is consistent — `aligned` requires every criterion `met` and both finding arrays empty; `revise` requires `blocking` non-empty; `attention` is the remaining case.
- Every criterion has all required fields and a recognized `status`.
- Every finding has all required fields and a recognized `category`.
- The number of criteria roughly matches the number of checkboxes in the phase spec's Success Criteria section. If counts disagree, flag it but do not fail validation.

If validation fails, summarize the problem and re-spawn the sub-agent once with feedback. After two failures, stop and show the user the raw response.

### 5. Render the report

Present a human-readable summary to the user:

```
Drift audit — phase #<n> — <title>

Verdict: <verdict>

<summary paragraph>

Success criteria (<met-count>/<total>):
  [SC1] met         — <text>
  [SC2] partial     — <text>
                      → <evidence>
  [SC3] unmet       — <text>
                      → <evidence>

Blocking (<count>):
  [B1] <category> — <file>:<line>
       <summary>
       → <suggested_action>

Concerns (<count>):
  [C1] <category> — <file>:<line>
       <summary>
       → <suggested_action>
```

Then ask:

- **PR mode**: "Post this as a comment on PR #<n>? (yes / phase-issue / save-only / cancel)"
  - `yes` — post as a PR comment.
  - `phase-issue` — post as a comment on the phase issue instead (useful when the PR comment thread is reserved for review-phase output).
  - `save-only` — write the JSON to disk but do not post.
  - `cancel` — discard.
- **Worktree mode**: "Post this as a comment on phase issue #$0? (yes / save-only / cancel)"

Default to `save-only` if the user gives an ambiguous response — the JSON on disk is always the source of truth.

### 6. Post the comment (if requested)

Render the approved report as a single markdown comment. Skip empty sections.

```markdown
## Plan Drift — verdict: <verdict>

<summary paragraph>

### Success criteria

| ID | Status | Criterion |
|---|---|---|
| SC1 | met | <text> |
| SC2 | partial | <text> |
| SC3 | unmet | <text> |

<For each non-`met` criterion, add a one-line evidence note below the table.>

### Blocking

#### [B1] <category> — `<file>:<line>`
<summary>

<details>

**Suggested action**: <suggested_action>

_Evidence_: <evidence>

### Concerns

#### [C1] <category> — `<file>:<line>`
<summary>

<details>

**Suggested action**: <suggested_action>

_Evidence_: <evidence>
```

Post the comment:

- **PR mode + `yes`**: `gh pr comment <pr-number> --repo <repo> --body "$(cat <<'EOF' … EOF)"`
- **PR mode + `phase-issue`**, or **worktree mode + `yes`**: `gh issue comment $0 --repo <repo> --body "..."`

### 7. Save the findings JSON

Cache the validated findings JSON so a future auto-correct loop can read it without re-running the audit:

```bash
echo "<findings JSON>" | marvin findings cache <plan> drift <phase-ident>
```

Where `<plan>` is the plan identifier from step 1 (e.g. `plan-00042`) and `<phase-ident>` is formed as `phase-XXXXX-N` (e.g. `phase-00042-3`). The cache is stored under `.claude/cache/<plan>/drift/<phase-ident>.json` and is gitignored by convention. The user can re-run `/plan-drift` to regenerate.

### 8. Confirm

Report:
- Comment URL (if posted) or "saved only"
- Verdict
- Criterion summary (e.g. "2/3 met, 1 unmet")
- Blocking count, concerns count
- Cached findings path (`.claude/cache/<plan>/drift/<phase-ident>.json`)

Recommend the next step based on verdict:
- **`aligned`**: "Diff matches spec. Open the PR with `/implement-phase` (if not already), or proceed to `/review-phase $0`."
- **`attention`**: "Continue implementation. Re-run `/plan-drift $0` before opening the PR."
- **`revise`**: "Address blocking findings — remove out-of-scope changes, document divergences in PR Notes, or fix interface mismatches. Then re-run `/plan-drift $0`."
