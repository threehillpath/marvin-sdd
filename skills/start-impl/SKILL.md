---
name: start-impl
description: Summarize an implementation plan, confirm phases, and create the implementation branch
argument-hint: <impl-plan-issue-number>
allowed-tools: Bash, Read
model: sonnet
---

Summarize what is about to be built, confirm phase readiness, and create the implementation branch.

**Before starting**: Read `.claude/plan-workflow-config.yml` for project configuration (repo, owner). Read `../SHARED/GLOSSARY.md` for branch/issue naming and the status state machine.

## Arguments

- `$0` — Impl plan issue number

## Steps

### 1. Fetch context

```bash
gh issue view $0 --repo <repo> --json number,title,body,comments,labels
```

Extract the PLAN-XXXXX number from the title. Then fetch the phase hierarchy for this plan:

```bash
marvin issue tree $0
```

This returns one pipe-delimited line per node — `<kind> | #<number> | <state> | <status> | <title>` — with `kind` one of `arch`, `impl`, `phase`. Filter for lines where `kind` is `phase`. If zero `phase` lines are found (not the same as an empty result — `issue tree` always emits the target's own node), this plan predates sub-issue linking; fall back to:

```bash
marvin issue list --label "plan:phase" --title-prefix "[PLAN-XXXXX-" --state all
```

Note the trailing hyphen and **no closing bracket** — this is a true string-prefix match against phase titles like `[PLAN-XXXXX-1] ...`, unlike the closing-bracket form. This returns one pipe-delimited line per issue — `<number> | <state> | <labels-comma-joined> | <title>`.

For a multi-impl track (`[PLAN-XXXXX-A]`, `[PLAN-XXXXX-B]` — see `../SHARED/GLOSSARY.md`), apply the same rule one level deeper and use `--title-prefix "[PLAN-XXXXX-<suffix>-"` (e.g. `"[PLAN-00042-A-"`), so the fallback does not also match the sibling track's phases.

All phases should have `state == OPEN`. If any phase is not open, surface the list to the user and ask whether to proceed.

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

### 2. Derive branch names

Parse the plan identifier from the impl plan title (fetched in step 1):

```bash
marvin parse title "<impl plan title>"
```

Read the `plan:` line (e.g. `2`) from the output.

Resolve `<type>`: check the labels fetched in step 1 for `bug` or `enhancement` — per `../SHARED/LABELS.md`, the impl plan issue carries forward the source issue's type label. If `bug` is present, `<type>` is `bug`; otherwise (including when neither label is present) `<type>` is `feature`.

Then derive branch names using that source-issue number and the resolved type:

```bash
marvin names derive <plan> --type <type>
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

Read the `main_branch:`, `title_prefix_arch:`, and `title_prefix_impl:` lines from the output (`title_prefix_phase:` is emitted only when `--phase` is passed, which this call does not). Phase branches follow the pattern `<type>/PLAN-XXXXX/phase-N` as documented in `../SHARED/GLOSSARY.md`.

### 3. Present summary for confirmation

```
Implementation Plan: #N [PLAN-XXXXX] <Title>
Trunk branch: <type>/PLAN-XXXXX/main (from main)

Phases (sequential):
  Phase 1: #N1 [PLAN-XXXXX-1] <Phase Title>  → <type>/PLAN-XXXXX/phase-1
  Phase 2: #N2 [PLAN-XXXXX-2] <Phase Title>  → <type>/PLAN-XXXXX/phase-2
  ...

Each phase will:
  - Branch from <type>/PLAN-XXXXX/main
  - Run the TDD implementation loop autonomously via /implement-phase
  - Open a PR back to <type>/PLAN-XXXXX/main for review
  - Wait for merge before the next phase begins

Ready to create the implementation branch?
```

Ask for confirmation before proceeding.

### 4. Check autonomous execution prerequisites

Use the Read tool to check both settings files:
- `.claude/settings.local.json`
- `~/.claude/settings.json`

If the test runner, `git commit`, `git push`, `gh pr`, and `marvin` are not present in either file's allow list, note: "For autonomous phase execution, pre-approve test, git, and marvin commands in your settings. You can proceed now and configure permissions before running `/implement-phase`."

### 5. Create implementation branch

```bash
git checkout main
git pull origin main
git checkout -b <main_branch>
git push -u origin <main_branch>
```

### 6. Move impl plan to In Progress

```bash
marvin board move $0 in-progress
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

### 7. Confirm

Report: impl plan issue, branch `<main_branch>` created and pushed, board status.

**Next step**: `/implement-phase <phase-1-issue-number>`
