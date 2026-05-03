---
name: arch-plan
description: Interview and produce an architectural plan for a GitHub issue, stored as a new GitHub issue
argument-hint: <source-issue-number>
allowed-tools: Bash, Read, Glob, Grep
model: opus
---

Create an architectural plan for a GitHub issue. Arch plans focus on domain/system concerns — what to build and why, not how to implement it.

**Before starting**: Read `.claude/plan-workflow-config.md` for project configuration values and board management commands. Read `../SHARED/GLOSSARY.md` for naming conventions and the status state machine.

## Arguments

- `$0` — Source issue number

## Steps

### 1. Fetch context

```bash
gh issue view $0 --repo <repo> --json number,title,body,labels,comments
```

Also read any architecture decision records, domain model files, and guidance docs present in the repository.

### 2. Determine PLAN number

The plan number is `$0` zero-padded to 5 digits. See `../SHARED/GLOSSARY.md`.

### 3. Ask clarifying questions if needed

Evaluate: domain model impacts, integration points, cross-cutting concerns, scope boundaries, ADR candidates, TDD anchor. If any are unclear from context, ask before drafting. Group questions — do not interrogate one at a time.

For what qualifies as an ADR candidate, see `SUPPLEMENTS/ADR.md`.

### 4. Draft the plan

Read `SUPPLEMENTS/TEMPLATES.md` and use it to produce the arch plan document.

### 5. Present for review

Read `../SHARED/LABELS.md`. Infer domain labels from the plan content. Present the draft with proposed labels: "I'll apply: `plan:arch`, `status:upcoming`, `domain:backend` — correct?" Allow corrections before proceeding.

See `../SHARED/RENDERING.md` for rendering guidance. Ask for approval on both content and labels; iterate until confirmed.

### 6. Create the GitHub issue

```bash
gh issue create --repo <repo> \
  --title "[PLAN-XXXXX-ARCH] <Title>" \
  --body "<approved content>" \
  --label "plan:arch,status:upcoming,<domain-labels>,<source-issue-type-if-applicable>"
```

If a label does not exist, create it first — see `../SHARED/LABELS.md` for the create commands.

### 7. Link to source issue

```bash
gh issue comment $0 --repo <repo> \
  --body "Architecture plan created: #<new-issue> ([PLAN-XXXXX-ARCH])"
```

### 8. Add to board as Ready

Use the board management commands in `.claude/plan-workflow-config.md`. Set status to **Ready**.

### 9. Confirm

Report: source issue, new arch plan issue number and title, board status, ADR candidates (if any).

**Next step**: `/impl-plan <new-issue-number>`
