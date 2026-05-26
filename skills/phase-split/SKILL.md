---
name: phase-split
description: Break an implementation plan into phases and create GitHub issues for each
argument-hint: <impl-plan-issue-number>
allowed-tools: Bash, Read, Glob, Grep
model: sonnet
---

Break an approved implementation plan into phases sized by logical atomicity and estimated complexity. Each phase should be a coherent unit representing one branch, one PR, and one verifiable behavior change.

**Before starting**: Read `.claude/plan-workflow-config.md` for project configuration (repo, owner). Read `../SHARED/GLOSSARY.md` for naming and status conventions.

If the impl plan has not been red-teamed yet, recommend running `/red-team-plan $0` first — phase boundaries depend on the plan being free of phase-ordering and missing-dependency issues, which red-team-plan surfaces. The user may skip and proceed if they prefer.

## Arguments

- `$0` — Impl plan issue number

## Steps

### 1. Fetch context

```bash
gh issue view $0 --repo <repo> --json number,title,body,comments
```

Extract the PLAN-XXXXX number. Fetch the arch plan and source issue if referenced.

### 2. Propose phase boundaries

Read `SUPPLEMENTS/PHASING.md` for sizing and boundary guidance. Analyze the implementation plan and propose phases.

Present the proposed phases to the user before creating any issues:

```
Phase 1: <title> — <one-line scope>
Phase 2: <title> — ...

Dependencies: Phase 2 requires Phase 1. Phases 3 and 4 can run in parallel.
```

Ask: "Does this phase breakdown look right? Any changes before I create the issues?"

Iterate until approved.

### 3. Create phase issues

Render the phase issue template:

```bash
marvin template render impl-phase
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

Use the rendered skeleton as the structural frame for each phase issue body, filling in phase-specific content.

Read `../SHARED/LABELS.md` for label conventions. Infer domain labels from the impl plan content — confirm with the user once before creating all issues ("I'll apply `plan:phase`, `status:upcoming`, `domain:backend` to all phases — correct?").

Ensure all required labels exist before creating issues:

```bash
marvin label ensure --builtins
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

For any domain labels not covered by `--builtins`, ensure each one individually:

```bash
marvin label ensure "<name>" --description "<desc>" --color "<hex>"
```

For each approved phase:

```bash
gh issue create --repo <repo> \
  --title "[PLAN-XXXXX-N] <Phase Title>" \
  --body "<phase content>" \
  --label "plan:phase,status:upcoming,<domain-labels>"
```

Capture each issue number as you go.

### 4. Link phases to impl plan

`gh issue comment --body` does not interpret backslash escapes, so use a HEREDOC to get real newlines:

```bash
gh issue comment $0 --repo <repo> --body "$(cat <<'EOF'
Phases created:
- #<N1> [PLAN-XXXXX-1]
- #<N2> [PLAN-XXXXX-2]
- ...
EOF
)"
```

### 5. Add all phases to board as Ready

For each phase issue:

```bash
marvin board move <phase-issue-number> ready
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

### 6. Confirm

Report: impl plan issue, list of created phase issues with titles, board status.

**Next step**: Move the first phase you intend to start to **In Progress** using `/move-issue <issue-number> in-progress`, then begin implementation.
