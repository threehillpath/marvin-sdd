---
name: red-team-plan
description: Critique an implementation plan with a fresh-context opus sub-agent before phase-split, and post the critique to the impl plan issue
argument-hint: <impl-plan-issue-number>
allowed-tools: Bash, Read, Agent
model: sonnet
---

Run after `/impl-plan` produces an implementation plan, before `/phase-split`. Spawns a fresh-context opus sub-agent (extended thinking) that reads the impl plan, the parent arch plan, the source issue, and a code digest of the relevant files, then applies the rubric in `../SHARED/PLAN_RED_TEAM_RUBRIC.md` and returns structured findings per `../SHARED/PLAN_RED_TEAM_FORMAT.md`. Orchestrator presents findings, then on approval posts a single critique comment on the impl plan issue.

The red-team is a check on the plan, not the code. It surfaces hidden assumptions, missing dependencies, phase-ordering risks, weak TDD entry points, and unfalsifiable success criteria — the failure modes whose cost compounds across every phase if not caught here.

**Before starting**: Read `.claude/plan-workflow-config.md` for project configuration. Read `../SHARED/GLOSSARY.md` for naming and status conventions.

## Arguments

- `$0` — Impl plan issue number (the `[PLAN-XXXXX]` issue)

## Steps

### 1. Fetch context

```bash
gh issue view $0 --repo <repo> --json number,title,body,labels
```

Verify the issue is an impl plan (title matches `[PLAN-XXXXX] …`, label includes `plan:impl`). If it is the arch plan or a phase issue, stop with a clear message — red-team-plan only runs against impl plans.

Extract the arch plan issue number and source issue number from the impl plan body (referenced as `Architecture Plan: #<n>` and `Source Issue: #<n>`).

Extract the plan identifier from the issue title:

```bash
marvin parse title "<issue title>"
```

Read the `plan` field from the JSON output (e.g. `PLAN-00042`). Use this value wherever `<plan>` appears in subsequent steps.

### 2. Identify relevant source files

From the impl plan's component sections, identify the **paths** of source files that the plan touches or assumes. List them; do not read them yourself. The sub-agent will read them in its own context window.

If the plan names a path the repo does not contain, that itself is a finding the reviewer should surface — pass the list as the plan describes it, and let the reviewer verify.

### 3. Spawn the red-team sub-agent

Spawn an **Agent** with:

- `subagent_type: "general-purpose"`
- `model: "opus"`
- No worktree isolation (red-team is read-only)

Assemble the prompt by **referencing** the inputs the sub-agent should fetch — do not paste the plan body or any source file into the prompt. The sub-agent has its own context window.

Prompt template:

> You are red-teaming an implementation plan for the plan-workflow plugin. Your output is a single JSON object per the format in `<absolute-path-to>/skills/SHARED/PLAN_RED_TEAM_FORMAT.md`. Read that file first, then read `<absolute-path-to>/skills/SHARED/PLAN_RED_TEAM_RUBRIC.md` for the rubric and stance. Apply the rubric category by category to the plan under review.
>
> **Impl plan (under review)**: #$0 in repo `<repo>`. Read the body for objective, scope, components, design notes, and success criteria.
> **Arch plan (parent context)**: #<arch-issue> in repo `<repo>`. Read for higher-level decisions already made — do not critique these.
> **Source issue (origin)**: #<source-issue> in repo `<repo>`. Read for the user-facing problem the plan must solve.
>
> Fetch the inputs you need with these commands (run them yourself):
>
> ```
> gh issue view $0 --repo <repo> --json title,body,comments
> gh issue view <arch-issue> --repo <repo> --json title,body
> gh issue view <source-issue> --repo <repo> --json title,body
> ```
>
> Read the relevant source files in the consuming project to verify the plan's assumptions. Paths the plan touches or assumes:
>
> <bullet list of paths from step 2>
>
> Use the Read, Glob, and Grep tools to verify that named files, types, functions, and routes exist with the shape the plan claims. If a named symbol does not exist, that is a `hidden-assumption` finding — cite the absence as evidence.
>
> Read the rubric's "Stance" and "Anti-patterns" sections before drafting findings. Use extended thinking. Walk every component spec, every TDD entry point, and every success criterion. Ask "what would falsify this?" and "what does this assume that I have not verified?" Trace contracts between components and check whether `phase-split` could produce truly independent phases.
>
> For each finding, populate every field in the Finding schema. Use `evidence` to quote the specific plan text, code excerpt, or named absence — the user must be able to verify in seconds.
>
> Return **only** the JSON object specified in `PLAN_RED_TEAM_FORMAT.md`. No surrounding prose, no code fence.

The sub-agent reads the rubric, the plan, the parent arch plan, the source issue, and the source files in its own context window. Do not pre-fetch any of these in the orchestrator beyond what step 1 already did to validate the issue.

### 4. Parse and validate the response

Parse the returned JSON. Validate:

- The shape matches `PLAN_RED_TEAM_FORMAT.md` (top-level keys: `summary`, `verdict`, `blocking`, `concerns`).
- `verdict` is consistent with the arrays — `approve` requires both empty, `revise` requires `blocking` non-empty, `discuss` requires `blocking` empty and `concerns` non-empty.
- Every finding has all required fields and a recognized `category`.

If validation fails, summarize the problem and re-spawn the sub-agent once with feedback. After two failures, stop and show the user the raw response.

### 5. Render the findings for review

Present a human-readable summary to the user:

```
Red-team of impl plan #<n> — <title>

Verdict: <verdict>

<summary paragraph from response>

Blocking (<count>):
  [B1] <category> — <section>
       <summary>
  [B2] ...

Concerns (<count>):
  [C1] <category> — <section>
       <summary>
  ...
```

Then ask: **"Post this as a comment on impl plan #$0? (yes / edit / cancel)"**

- `yes` — proceed to step 6.
- `edit` — show the full JSON; let the user remove or downgrade specific findings (by id). Re-render the summary after edits.
- `cancel` — stop. Do not post anything. The findings JSON is discarded.

### 6. Post the critique to the impl plan issue

Render the approved findings as a single markdown comment. Skip empty sections.

```markdown
## Plan Red-Team — verdict: <verdict>

<summary paragraph>

### Blocking

#### [B1] <category> — <section>
<summary>

<details>

**Suggested revision**: <suggested_revision>

_Evidence_: <evidence>

#### [B2] ...

### Concerns

#### [C1] <category> — <section>
<summary>

<details>

**Suggested revision**: <suggested_revision>

_Evidence_: <evidence>
```

Post:

```bash
gh issue comment $0 --repo <repo> --body "$(cat <<'EOF'
<rendered comment>
EOF
)"
```

### 7. Save the findings JSON

Cache the validated findings JSON so a future auto-revise loop can read it without re-running the red-team:

```bash
echo "<findings JSON>" | marvin findings cache <plan> red-teams <plan>
```

Where `<plan>` is the plan identifier from step 1 (e.g. `plan-00042`). The cache is stored under `.claude/cache/<plan>/red-teams/<plan>.json` and is gitignored by convention. The user can re-run `/red-team-plan` to regenerate.

### 8. Confirm

Report:
- Comment URL on impl plan issue
- Verdict
- Blocking count, concerns count
- Cached findings path (`.claude/cache/<plan>/red-teams/<plan>.json`)

Recommend the next step based on verdict:
- **`revise`**: "Address blocking findings on the plan, then re-run `/red-team-plan $0` for a fresh pass — or proceed to `/phase-split $0` if you disagree with a finding."
- **`discuss`**: "Concerns recorded. Proceed with `/phase-split $0`, or revise the plan first."
- **`approve`**: "Plan looks solid. Proceed with `/phase-split $0`."
