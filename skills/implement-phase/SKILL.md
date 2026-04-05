---
name: implement-phase
description: Run the TDD implementation loop for a phase in an isolated worktree, then open a PR to the implementation branch
argument-hint: <phase-issue-number>
allowed-tools: Bash, Read, Glob, Grep, Agent
model: sonnet
---

Implement a phase autonomously using a TDD loop in an isolated git worktree. The sub-agent writes tests, implements features, and opens a PR to the implementation branch when all success criteria pass.

**Before starting**: Read `.claude/plan-workflow-config.md` for project configuration, board management commands, and test commands.

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

Read relevant source files identified in the impl plan — handlers, workers, repos, components — to give the sub-agent current code context.

### 2. Derive branches

From PLAN-XXXXX-N:
- Phase branch: `feature/plan-XXXXX-N`
- Impl branch: `feature/plan-XXXXX`

Verify the impl branch exists on the remote:

```bash
git ls-remote --heads origin feature/plan-XXXXX
```

If not found, stop: "Implementation branch `feature/plan-XXXXX` not found. Run `/start-impl <impl-plan-issue>` first."

### 3. Move phase to In Progress

Use the board management commands in `.claude/plan-workflow-config.md`. Set status to **In Progress**.

### 4. Spawn implementation sub-agent

Read `SUPPLEMENTS/LOOP.md` for the TDD loop instructions.

Spawn a **general-purpose** agent with `isolation: "worktree"` and model **sonnet**. Assemble the task prompt from:

1. Phase issue content (title, objective, scope, TDD entry point, success criteria)
2. Impl plan content (full component specs, design notes)
3. Current source file contents (read in step 1)
4. Branch names: phase branch `feature/plan-XXXXX-N`, impl branch `feature/plan-XXXXX`
5. Repo: `<repo>`
6. Test commands from `.claude/plan-workflow-config.md`
7. Full instructions from `SUPPLEMENTS/LOOP.md`

The sub-agent must not pause for user confirmation except on unresolvable failure or ambiguity.

### 5. Report results

When the sub-agent returns, report:
- PR URL (if created successfully)
- Test results summary
- Success criteria completed
- Any failures or escalations requiring input

If the sub-agent stopped on failure, present the diagnostic and ask how to proceed.

**After the PR is merged**:
- If more phases remain: `/implement-phase <next-phase-issue-number>`
- If this was the last phase: `/finish-impl <impl-plan-issue-number>`
