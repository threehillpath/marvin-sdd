---
name: phase-split
description: Break an implementation plan into phases and create GitHub issues for each
argument-hint: <impl-plan-issue-number>
allowed-tools: Bash, Read, Glob, Grep
model: sonnet
---

Break an approved implementation plan into phases sized by logical atomicity and estimated complexity. Each phase should be a coherent unit representing one branch, one PR, and one verifiable behavior change.

**Before starting**: Read `.claude/plan-workflow-config.yml` for project configuration (repo, owner). Read `../SHARED/GLOSSARY.md` for naming and status conventions.

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

Before creating any issues, check whether phases already exist for this plan:

```bash
marvin issue list --label "plan:phase" --title-prefix "[PLAN-XXXXX]" --state all
```

If the result is non-empty, show the existing phases and stop: "Phase issues already exist for [PLAN-XXXXX]. Run `/move-issue <issue-number> in-progress` to start one, or delete the existing phase issues manually before re-splitting."

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

Render the phase issue template:

```bash
marvin template render impl-phase --skeleton
```

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

For each approved phase, compose the title and body **together, as one atomic unit, immediately before creating that issue** — do not draft all bodies in a separate pass from titles, and do not hold titles and content as two lists tracked independently. Title/body pairing drift (a body describing a different phase than its own title) has happened before and is easy to introduce silently when title and content are generated in separate passes.

```bash
gh issue create --repo <repo> \
  --title "[PLAN-XXXXX-N] <Phase Title>" \
  --body "<phase content>" \
  --label "plan:phase,status:upcoming,<domain-labels>"
```

Capture each issue number as you go.

### 3b. Verify title/body pairing

Before moving to step 4, re-fetch every created issue and confirm each one's body actually describes its own title — not an adjacent phase's:

```bash
gh issue view <issue-number> --repo <repo> --json title,body
```

For each issue, check that the `## Objective` and `## Components` sections reference the same phase number and component(s) named in the title. If any issue's body describes a different phase, fix it immediately with `gh issue edit <issue-number> --repo <repo> --body-file <file>` before proceeding — do not defer this to a later skill.

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
