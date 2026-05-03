---
name: implement-phase
description: Run the TDD implementation loop for a phase in an isolated worktree, then open a PR to the implementation branch
argument-hint: <phase-issue-number>
allowed-tools: Bash, Read, Glob, Grep, Agent
model: sonnet
---

Implement a phase autonomously using a TDD loop in a git worktree. The sub-agent writes tests, implements features, and opens a PR to the implementation branch when all success criteria pass.

**Before starting**: Read `.claude/plan-workflow-config.md` for project configuration, board management commands, and test commands. Read `../SHARED/GLOSSARY.md` for branch, worktree, and status conventions.

## Arguments

- `$0` — Phase issue number

## Steps

### 1. Fetch context

```bash
gh issue view $0 --repo <repo> --json number,title,body
```

Extract PLAN-XXXXX-N, the phase title, the success criteria checklist, and the impl plan issue number from the body.

```bash
gh issue view <impl-plan-issue> --repo <repo> --json number,title,body
```

From the impl plan, identify the **paths** of source files relevant to this phase — handlers, workers, repos, components, and any neighbours that will need to be edited or referenced. Do **not** read these files into your own context. The sub-agent will read them in its own window.

### 2. Derive branches

Per `../SHARED/GLOSSARY.md`: phase branch is `feature/plan-XXXXX-N`, impl branch is `feature/plan-XXXXX`. Verify the impl branch exists on the remote:

```bash
git ls-remote --heads origin feature/plan-XXXXX
```

If not found, stop: "Implementation branch `feature/plan-XXXXX` not found. Run `/start-impl <impl-plan-issue>` first."

### 2b. Pre-create the phase branch and worktree

Create the phase branch from the impl branch and add a worktree for it. This gives the sub-agent a correctly-based working directory without relying on the Agent tool to set up the branch.

Check whether the phase branch already exists locally and/or remotely:

```bash
git rev-parse --verify --quiet refs/heads/feature/plan-XXXXX-N      # local
git ls-remote --heads origin feature/plan-XXXXX-N                   # remote
```

Decide which case applies:

- **Neither exists**: create from the impl branch and push.
  ```bash
  git fetch origin feature/plan-XXXXX
  git branch feature/plan-XXXXX-N origin/feature/plan-XXXXX
  git push -u origin feature/plan-XXXXX-N
  ```
- **Remote exists, local does not**: fetch the remote ref; the worktree-add step below will create the local branch tracking it.
  ```bash
  git fetch origin feature/plan-XXXXX-N
  ```
- **Local exists** (with or without remote): a previous run left it behind. Stop and ask the user how to proceed — options are (a) reuse the existing branch, (b) delete it (`git branch -D feature/plan-XXXXX-N`) and recreate from the impl branch. Do not silently overwrite.

Once the branch state is resolved, create the worktree:

```bash
git worktree add .claude/worktrees/phase-XXXXX-N feature/plan-XXXXX-N
```

If `.claude/worktrees/phase-XXXXX-N` already exists from a prior failed run, remove it first (`git worktree remove --force .claude/worktrees/phase-XXXXX-N && git worktree prune`) before re-adding, after confirming with the user.

Note the absolute worktree path — pass it to the sub-agent in step 4.

### 3. Move phase to In Progress

Use the board management commands in `.claude/plan-workflow-config.md`. Set status to **In Progress**.

### 4. Spawn implementation sub-agent

Read `SUPPLEMENTS/LOOP.md` for the TDD loop instructions.

Spawn a **general-purpose** agent **without** `isolation: "worktree"` (the worktree was created in step 2b) and model **sonnet**. Assemble the task prompt by **referencing** the inputs the sub-agent should fetch — do not paste full file contents into the prompt:

1. Phase issue number `$0` (and `gh issue view` command for it). The sub-agent fetches the title, objective, scope, TDD entry point, and success criteria itself.
2. Impl plan issue number (and `gh issue view` command for it). The sub-agent fetches the full component specs and design notes itself.
3. **Paths only** for the relevant source files identified in step 1 — the sub-agent reads them inside the worktree.
4. Branch names: phase branch `feature/plan-XXXXX-N`, impl branch `feature/plan-XXXXX`.
5. Absolute worktree path: `<repo-root>/.claude/worktrees/phase-XXXXX-N`.
6. Repo: `<repo>`.
7. Test commands from `.claude/plan-workflow-config.md`.
8. Full instructions from `SUPPLEMENTS/LOOP.md` (paste this verbatim — it is the agent's primary procedural guide).

The sub-agent must not pause for user confirmation except on unresolvable failure or ambiguity.

### 5. Report results

When the sub-agent returns, report:
- PR URL (if created successfully)
- Test results summary
- Success criteria completed
- Any failures or escalations requiring input

If the sub-agent stopped on failure, present the diagnostic and ask how to proceed.

The worktree at `.claude/worktrees/phase-XXXXX-N` is **left in place** until `/wrap-phase` runs after merge — reviewers may want to test the branch locally, and the user may push correction commits from it.

**After the PR is merged**:
- Run `/wrap-phase <phase-issue-number> <impl-plan-issue-number>` to capture decisions, close the phase issue, move it to Done, and clean up the worktree.
- If more phases remain: `/implement-phase <next-phase-issue-number>`
- If this was the last phase: `/finish-impl <impl-plan-issue-number>`
