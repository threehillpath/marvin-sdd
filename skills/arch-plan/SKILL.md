---
name: arch-plan
description: Interview and produce an architectural plan for a GitHub issue, stored as a new GitHub issue
argument-hint: <source-issue-number>
allowed-tools: Bash, Read, Glob, Grep
model: opus
---

Create an architectural plan for a GitHub issue. Arch plans focus on domain/system concerns — what to build and why, not how to implement it.

**Before starting**: Read `.claude/plan-workflow-config.yml` for project configuration values (repo, owner). Read `../SHARED/GLOSSARY.md` for naming conventions and the status state machine.

## Arguments

- `$0` — Source issue number

## Steps

### 1. Fetch context

```bash
gh issue view $0 --repo <repo> --json number,title,body,labels,comments
```

Also read any architecture decision records, domain model files, and guidance docs present in the repository.

### 2. Parse issue body

Inspect the fetched issue body. Read `.github/ISSUE_TEMPLATE/feature.yml` in the consuming project's repository to obtain the canonical form field labels.

**Form-originated body** (presence of `### <Field Label>` headings matching the feature form's labels):
- Extract each labelled section by heading into named working-context variables.
- The fields **Problem Statement** and **Scope** are required for the clarifying-question evaluation in step 4. If either is absent (optional field not filled), treat it as missing and add a clarifying question rather than failing.
- Any additional form fields (e.g. **Goals**, **Context**) are extracted and available for use in drafting.

**Legacy / free-form body** (no matching `### <Field Label>` headings found):
- Read the whole body as prose.
- Proceed to step 4 exactly as before — no extraction, no failure.

This step is read-side only. It never modifies the source issue or applies form structure to bot-created issues.

### 3. Determine PLAN number

The plan number is `$0` zero-padded to 5 digits. See `../SHARED/GLOSSARY.md`.

### 4. Ask clarifying questions if needed

Evaluate: domain model impacts, integration points, cross-cutting concerns, scope boundaries, ADR candidates, TDD anchor. If any are unclear from context, ask before drafting. Group questions — do not interrogate one at a time.

For what qualifies as an ADR candidate, see `SUPPLEMENTS/ADR.md`.

### 5. Draft the plan

Render the arch plan template:

```bash
marvin template render arch-plan
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

Use the rendered skeleton as the structural frame for the draft, filling in each section with substantive content from the arch analysis.

### 6. Present for review

Read `../SHARED/LABELS.md`. Infer domain labels from the plan content. Present the draft with proposed labels: "I'll apply: `plan:arch`, `status:upcoming`, `domain:backend` — correct?" Allow corrections before proceeding.

See `../SHARED/RENDERING.md` for rendering guidance. Ask for approval on both content and labels; iterate until confirmed.

### 7. Create the GitHub issue

Ensure all required labels exist before creating the issue:

```bash
marvin label ensure --builtins
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

For any domain or source-type labels not covered by `--builtins`, ensure each one individually:

```bash
marvin label ensure "<name>" --description "<desc>" --color "<hex>"
```

Then create the issue:

```bash
gh issue create --repo <repo> \
  --title "[PLAN-XXXXX-ARCH] <Title>" \
  --body "<approved content>" \
  --label "plan:arch,status:upcoming,<domain-labels>,<source-issue-type-if-applicable>"
```

### 8. Link to source issue

```bash
gh issue comment $0 --repo <repo> \
  --body "Architecture plan created: #<new-issue> ([PLAN-XXXXX-ARCH])"
```

### 9. Add to board as Ready

```bash
marvin board move <new-issue-number> ready
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

### 10. Confirm

Report: source issue, new arch plan issue number and title, board status, ADR candidates (if any).

**Next step**: `/impl-plan <new-issue-number>`
