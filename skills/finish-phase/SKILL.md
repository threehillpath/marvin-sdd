---
name: finish-phase
description: Commit, push, open a PR to the implementation branch, and move a phase issue to In Review
argument-hint: <phase-issue-number>
allowed-tools: Bash, Read
model: sonnet
---

Close out a completed phase: commit staged work, push, open a PR to the correct base branch, and move the issue to In Review.

The PR target is determined automatically from the current branch name:
- `feature/plan-XXXXX-N` → PR targets `feature/plan-XXXXX` (implementation branch)
- `feature/plan-XXXXX` → PR targets `main` (use `/finish-impl` instead for this case)

**Before starting**: Read `.claude/plan-workflow-config.yml` for project configuration (repo, owner). Read `../SHARED/GLOSSARY.md` for branch and status conventions.

## Arguments

- `$0` — Phase issue number (the `[PLAN-XXXXX-N]` issue)

## Steps

### 1. Fetch phase context

```bash
gh issue view $0 --repo <repo> --json number,title,body
```

Extract the PLAN-XXXXX-N identifier and phase title to use in the commit message and PR.

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

Resolve the PR base branch from the current branch:

```bash
marvin pr base "$(git branch --show-current)"
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."
If `marvin` exits with code 1 (branch does not match a known pattern), ask the user which branch to target before proceeding.

Read the `base` field from the JSON output to use as the PR target.

### 4. Confirm before committing

Present a summary of what will be committed and the proposed commit message. Do not stage or commit without explicit user confirmation.

Proposed commit message format:
```
[PLAN-XXXXX-N] <phase title>
```

### 5. Stage and commit

```bash
git add <confirmed files>
git commit -m "[PLAN-XXXXX-N] <phase title>"
```

### 6. Push

```bash
git push -u origin <branch>
```

### 7. Create PR

Read `../SHARED/PR_TEMPLATE.md` and use the **Phase PR** template.

```bash
gh pr create --repo <repo> \
  --title "[PLAN-XXXXX-N] <Phase Title>" \
  --base <pr-target> \
  --body "<PR body>"
```

The PR body must include `Closes #$0` to auto-close the phase issue on merge.

### 8. Move issue to In Review

```bash
marvin board move $0 in-review
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

### 9. Confirm

Report: commit hash, branch, PR URL, issue #$0 moved to In Review.

If more phases remain: "Next: merge this PR, then `/implement-phase <next-phase-issue-number>`"
If this was the last phase: "Next: merge this PR, then `/finish-impl <impl-plan-issue-number>`"
