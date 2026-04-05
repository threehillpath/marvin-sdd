---
name: start-impl
description: Summarize an implementation plan, confirm phases, and create the implementation branch
argument-hint: <impl-plan-issue-number>
allowed-tools: Bash, Read
model: sonnet
---

Summarize what is about to be built, confirm phase readiness, and create the implementation branch.

**Before starting**: Read `.claude/plan-workflow-config.md` for project configuration and board management commands.

## Arguments

- `$0` — Impl plan issue number

## Steps

### 1. Fetch context

```bash
gh issue view $0 --repo <repo> --json number,title,body,comments
```

Extract the PLAN-XXXXX number from the title. Find phase issues from the comments (look for the "Phases created:" comment posted by `phase-split`). Fetch each phase issue to confirm it is open.

```bash
gh issue view <phase-issue> --repo <repo> --json number,title,state
```

### 2. Derive branch names

From the PLAN-XXXXXXX number (zero-padded to 5 digits):
- Implementation branch: `feature/plan-XXXXX`
- Phase branches: `feature/plan-XXXXX-1`, `feature/plan-XXXXX-2`, etc.

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

If the test runner, `git commit`, `git push`, `gh pr`, and `gh project` are not present in either file's allow list, note: "For autonomous phase execution, pre-approve test and git commands in your settings. You can proceed now and configure permissions before running `/implement-phase`."

### 5. Create implementation branch

```bash
git checkout main
git pull origin main
git checkout -b feature/plan-XXXXX
git push -u origin feature/plan-XXXXX
```

### 6. Move impl plan to In Progress

Use the board management commands in `.claude/plan-workflow-config.md`. Set status to **In Progress**.

### 7. Confirm

Report: impl plan issue, branch `feature/plan-XXXXX` created and pushed, board status.

**Next step**: `/implement-phase <phase-1-issue-number>`
