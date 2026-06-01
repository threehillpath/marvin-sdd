---
name: implement-phase
description: Run the TDD implementation loop for a phase in an isolated worktree, then open a PR to the implementation branch
argument-hint: <phase-issue-number>
allowed-tools: Bash, Read, Glob, Grep, Agent
model: sonnet
---

Implement a phase autonomously using a TDD loop in a git worktree. The sub-agent writes tests, implements features, and opens a PR to the implementation branch when all success criteria pass.

**Before starting**: Read `.claude/plan-workflow-config.md` for project configuration (repo, owner) and test commands. Read `../SHARED/GLOSSARY.md` for branch, worktree, and status conventions.

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

```bash
marvin names derive $0 --phase <N>
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

Read the JSON output to obtain `impl_branch`, `phase_branch`, and `worktree_path`. Verify the impl branch exists on the remote:

```bash
git ls-remote --heads origin <impl_branch>
```

If not found, stop: "Implementation branch `<impl_branch>` not found. Run `/start-impl <impl-plan-issue>` first."

### 2b. Pre-create the phase branch and worktree

Create the phase branch from the impl branch and add a worktree for it. This gives the sub-agent a correctly-based working directory without relying on the Agent tool to set up the branch.

```bash
marvin worktree add <worktree_path> <phase_branch> <impl_branch>
```

`marvin worktree add` handles all three branch-state cases automatically:
- **Neither local nor remote exists**: creates the branch from `<impl_branch>` and pushes it.
- **Remote exists, local does not**: fetches and tracks the remote branch.
- **Local branch already exists**: exits 1 with a clear message — a previous run left it behind. Stop and ask the user how to proceed — options are (a) reuse the existing branch by running `git worktree add <worktree_path> <phase_branch>` directly (`marvin worktree add` refuses when a local branch exists without a matching worktree mapping), or (b) delete the branch (`git branch -D <phase_branch>`) and then re-run `marvin worktree add`.

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

If the worktree path already exists from a prior failed run, remove it first:

```bash
marvin worktree remove <worktree_path>
marvin worktree prune
```

After confirming with the user, then re-add the worktree.

Pass the absolute `worktree_path` to the sub-agent in step 4.

### 3. Move phase to In Progress

```bash
marvin board move $0 in-progress
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

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

**During implementation or before opening the PR** (optional):
- Run `/plan-drift <phase-issue-number>` to audit the worktree's diff against the phase spec — checks per-criterion coverage and flags out-of-scope or interface-divergent changes early.

**After the PR is opened**:
- Run `/plan-drift <phase-issue-number>` again if the PR has additional commits beyond the worktree audit, then `/review-phase <phase-issue-number>` for an opus-driven code review. Address blocking findings before merging.

**After the PR is merged**:
- Run `/wrap-phase <phase-issue-number> <impl-plan-issue-number>` to capture decisions, close the phase issue, move it to Done, and clean up the worktree.
- If more phases remain: `/implement-phase <next-phase-issue-number>`
- If this was the last phase: `/review-impl <impl-plan-issue-number>` for a comprehensive cross-phase review, then `/finish-impl <impl-plan-issue-number>`.
