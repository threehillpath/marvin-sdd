---
name: finish-phase
description: Commit, push, open a PR to the implementation branch, and move a phase issue to In Review
argument-hint: <phase-issue-number>
allowed-tools: Bash, Read
model: sonnet
---

Close out a completed phase: commit staged work, push, open a PR to the correct base branch, and move the issue to In Review.

The PR target is determined automatically from the current branch name:
- `feature/plan-XXX-N` → PR targets `feature/plan-XXX` (implementation branch)
- `feature/plan-XXX` → PR targets `main` (use `/finish-impl` instead for this case)

**Before starting**: Read `../SHARED/CONFIG.md` for project configuration and board management commands.

## Arguments

- `$0` — Phase issue number (the `[PLAN-XXX-N]` issue)

## Steps

### 1. Fetch phase context

```bash
gh issue view $0 --repo <repo> --json number,title,body
```

Extract the PLAN-XXX-N identifier and phase title to use in the commit message and PR.

### 2. Review current state

```bash
git status
git diff --stat HEAD
```

Show the output. If the working tree is clean and nothing is staged, stop — there is nothing to commit. Ask the user if they meant a different issue or branch.

### 3. Check branch and determine PR target

```bash
git branch --show-current
```

If on `main` or `master`, stop and warn.

Determine the PR target from the branch name:
- If branch matches `feature/plan-XXX-N` (phase branch) → PR target is `feature/plan-XXX`
- If branch matches `feature/plan-XXX` (impl branch) → PR target is `main`
- If branch does not match either pattern, ask the user which branch to target before proceeding.

### 4. Confirm before committing

Present a summary of what will be committed and the proposed commit message. Do not stage or commit without explicit user confirmation.

Proposed commit message format:
```
[PLAN-XXX-N] <phase title>
```

### 5. Stage and commit

```bash
git add <confirmed files>
git commit -m "[PLAN-XXX-N] <phase title>"
```

### 6. Push

```bash
git push -u origin <branch>
```

### 7. Create PR

Read `SUPPLEMENTS/TEMPLATES.md` for the PR body structure.

```bash
gh pr create --repo <repo> \
  --title "[PLAN-XXX-N] <Phase Title>" \
  --base <pr-target> \
  --body "<PR body>"
```

The PR body must include `Closes #$0` to auto-close the phase issue on merge.

### 8. Move issue to In Review

Use the board management commands in `../SHARED/CONFIG.md`. Set status to **In Review**.

### 9. Confirm

Report: commit hash, branch, PR URL, issue #$0 moved to In Review.

If more phases remain: "Next: merge this PR, then `/implement-phase <next-phase-issue-number>`"
If this was the last phase: "Next: merge this PR, then `/finish-impl <impl-plan-issue-number>`"
