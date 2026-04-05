---
name: finish-impl
description: Open a PR from the implementation branch to main and move the impl plan to In Review
argument-hint: <impl-plan-issue-number>
allowed-tools: Bash, Read
model: sonnet
---

Close out a completed implementation: confirm all phases are merged, open a PR from the implementation branch to main, and move the impl plan issue to In Review.

**Before starting**: Read `.claude/plan-workflow-config.md` for project configuration and board management commands.

## Arguments

- `$0` — Impl plan issue number

## Steps

### 1. Fetch context

```bash
gh issue view $0 --repo <repo> --json number,title,body,comments
```

Extract PLAN-XXXXX and the phase list from the "Phases created:" comment.

### 2. Verify branch state

```bash
git branch --show-current
```

Must be on `feature/plan-XXXXX`. If not, warn and stop — instruct the user to check out the implementation branch first.

Pull the latest from the remote — phase PRs were merged on GitHub and the local branch may be behind:

```bash
git pull origin feature/plan-XXXXX
```

```bash
git status
```

If there are uncommitted changes, stop and ask. The impl branch should be clean — all work arrives via merged phase PRs.

### 3. Summarize what is being shipped

```bash
git log main..HEAD --oneline
```

Present the summary:

```
Implementation branch: feature/plan-XXXXX
Target: main

Commits since main:
  <list from git log>

Phases:
  [PLAN-XXXXX-1] <title>
  [PLAN-XXXXX-2] <title>
  ...

This will open a PR: feature/plan-XXXXX → main. Proceed?
```

Ask for confirmation before creating the PR.

### 4. Create PR

Read `../finish-phase/SUPPLEMENTS/TEMPLATES.md` for the PR body structure. The PR body should summarize what the full implementation delivers across all phases. Include `Closes #$0` to auto-close the impl plan issue on merge.

```bash
gh pr create --repo <repo> \
  --title "[PLAN-XXXXX] <Impl Plan Title>" \
  --base main \
  --body "<PR body>"
```

### 5. Move impl plan to In Review

Use the board management commands in `.claude/plan-workflow-config.md`. Set status to **In Review**.

### 6. Confirm

Report: PR URL, impl plan issue #$0 moved to In Review.
