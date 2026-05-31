---
name: finish-impl
description: Open a PR from the implementation branch to main and move the impl plan to In Review
argument-hint: <impl-plan-issue-number>
allowed-tools: Bash, Read
model: sonnet
---

Close out a completed implementation: confirm all phases are merged, open a PR from the implementation branch to main, and move the impl plan issue to In Review.

**Before starting**: Read `.claude/plan-workflow-config.md` for project configuration (repo, owner). Read `../SHARED/GLOSSARY.md` for branch and status conventions.

## Arguments

- `$0` — Impl plan issue number

## Steps

### 1. Fetch context

```bash
gh issue view $0 --repo <repo> --json number,title,body,comments
```

Extract PLAN-XXXXX from the title:

```bash
marvin parse title "<issue title>"
```

Read the `plan` field (an integer) from the JSON output. Format it as `plan-XXXXX` — lowercase, 5-digit zero-padded (e.g. `plan-00042` for `plan=42`) — and use that string as `<plan>` in subsequent steps.

Extract the phase list from the "Phases created:" comment in the issue's comments:

```bash
echo "<phases-created-comment-body>" | marvin parse phase-list
```

This emits a JSON object `{"found": bool, "issues": [int, ...]}` — read the `issues` field for the phase issue numbers.

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

Read `../SHARED/PR_TEMPLATE.md` and use the **Implementation PR** template. Include `Closes #$0` to auto-close the impl plan issue on merge.

```bash
gh pr create --repo <repo> \
  --title "[PLAN-XXXXX] <Impl Plan Title>" \
  --base main \
  --body "<PR body>"
```

### 5. Move impl plan to In Review

```bash
marvin board move $0 in-review
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

### 6. Clear the findings cache

Now that the impl PR is open, clear the plan's findings cache — any accumulated review, drift, and red-team findings are superseded by the impl-level review that `/review-impl` will produce:

```bash
marvin findings clear <plan>
```

Where `<plan>` is the plan identifier from step 1 (e.g. `plan-00042`). This removes `.claude/cache/<plan>/` entirely. If the directory is already absent, this is a no-op.

### 7. Confirm

Report: PR URL, impl plan issue #$0 moved to In Review, findings cache cleared.

**Next step**: `/review-impl $0` to review the impl PR and post findings directly on it.
