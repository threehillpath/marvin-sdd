---
name: impl-plan
description: Create a technical implementation plan from an architecture plan issue
argument-hint: <arch-plan-issue-number>
allowed-tools: Bash, Read, Glob, Grep
model: opus
---

Create a technical implementation plan from an approved architecture plan. The impl plan is a specification — what to build and why, not how to code it.

**Before starting**: Read `../SHARED/CONFIG.md` for project configuration and board management commands.

## Arguments

- `$0` — Arch plan issue number (the `[PLAN-XXX-ARCH]` issue)

## Steps

### 1. Fetch context

```bash
gh issue view $0 --repo <repo> --json number,title,body,comments
```

Extract the PLAN-XXX number and source issue reference from the arch plan. Fetch the source issue too. Read relevant code files identified in the arch plan (handlers, workers, repos, schema).

### 2. Ask clarifying questions if needed

Evaluate: component sequencing, schema changes, layer boundaries, edge cases, verification approach, TDD entry points. If non-obvious, ask before drafting. Group questions.

### 3. Draft the plan

Read `SUPPLEMENTS/CONVENTIONS.md` for what to include and exclude. Read `SUPPLEMENTS/TEMPLATES.md` for the plan structure.

**TDD**: Each component section must include a TDD Entry Point. **Rendered UI components are exempt** — components and DOM-dependent code are not practical to unit test. Frontend logic without DOM dependency (reducers, utilities, validation helpers) is NOT exempt. See `SUPPLEMENTS/TDD.md` for full scope.

### 4. Present for review

Read `../SHARED/LABELS.md`. Infer domain labels from the plan content. Present the draft with proposed labels: "I'll apply: `plan:impl`, `status:upcoming`, `domain:backend` — correct?" Allow corrections before proceeding.

See `../SHARED/RENDERING.md` for rendering guidance. Ask for approval on both content and labels; iterate until confirmed.

### 5. Create the GitHub issue

```bash
gh issue create --repo <repo> \
  --title "[PLAN-XXX] <Title>" \
  --body "<approved content>" \
  --label "plan:impl,status:upcoming,<domain-labels>"
```

If a label does not exist, create it first — see `../SHARED/LABELS.md` for the create commands.

### 6. Link to arch plan

```bash
gh issue comment $0 --repo <repo> \
  --body "Implementation plan created: #<new-issue> ([PLAN-XXX])"
```

### 7. Add to board as Ready

Use the board management commands in `../SHARED/CONFIG.md`. Set status to **Ready**.

### 8. Confirm

Report: arch plan issue, new impl plan issue number and title, board status.

**Next step**: `/phase-split <new-issue-number>`
