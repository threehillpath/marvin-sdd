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
gh issue view $0 --repo <repo> --json number,title,body,comments
```

Extract the PLAN-XXXXX number from the title. Then fetch all phase issues for this plan in one call:

```bash
marvin issue list --label "plan:phase" --title-prefix "[PLAN-XXXXX]"
```

This returns a JSON array of `{number, title, state, labels}`. All items should have `state == "OPEN"`. If any phase is not open, surface the list to the user and ask whether to proceed.

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

### 2. Derive branch names

Parse the plan identifier from the impl plan title (fetched in step 1):

```bash
marvin parse title "<impl plan title>"
```

Read the `plan` integer field (e.g. `2`). Then derive branch names using that source-issue number:

```bash
marvin names derive <plan>
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

Read the JSON output to obtain `impl_branch` and the title prefix values. Phase branches follow the pattern `feature/plan-XXXXX-N` as documented in `../SHARED/GLOSSARY.md`.

### 3. Present summary for confirmation

```
Implementation Plan: #N [PLAN-XXXXX] <Title>
Implementation branch: feature/plan-XXXXX (from main)

Phases (sequential):
  Phase 1: #N1 [PLAN-XXXXX-1] <Phase Title>  → feature/plan-XXXXX-1
  Phase 2: #N2 [PLAN-XXXXX-2] <Phase Title>  → feature/plan-XXXXX-2
  ...

Each phase will:
  - Branch from feature/plan-XXXXX
  - Run the TDD implementation loop autonomously via /implement-phase
  - Open a PR back to feature/plan-XXXXX for review
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
git checkout -b <impl_branch>
git push -u origin <impl_branch>
```

### 6. Move impl plan to In Progress

```bash
marvin board move $0 in-progress
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

### 7. Confirm

Report: impl plan issue, branch `<impl_branch>` created and pushed, board status.

**Next step**: `/implement-phase <phase-1-issue-number>`
