---
name: arch-plan
description: Interview and produce an architectural plan for a GitHub issue, stored as a new GitHub issue
argument-hint: <source-issue-number>
allowed-tools: Bash, Read, Write, Glob, Grep
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

Get the empty YAML draft and the rules for filling it:

```bash
marvin template render arch-plan --skeleton
marvin template render arch-plan --guidance
```

If either command exits 1 (for example a malformed project override, or an unknown type), show stderr to the user and stop. Neither command reads the config, so neither returns exit 2.

Fill every key of the skeleton with substantive content from the arch analysis. In `title:`, replace `XXXXX` and `<Title>` with the real plan number and title. Follow every rule `--guidance` prints and the rules in `../SHARED/DRAFT_AUTHORING.md`; the skill-specific metadata rule is:

- `Source Issue` starts with the issue reference, `#<n>` (if it also names a plan, that plan must be the title's), and `Plan Number` is the title's plan number without the `-ARCH` suffix or brackets (`PLAN-00112` for the title `[PLAN-00112-ARCH] ...`).

Once every section is filled in, `Write` the filled draft to `<project-root>/.claude/cache/<plan>/arch-draft.yml`, where `<project-root>` is the main checkout's git root, as defined in `../SHARED/CONFIG.md` (not a linked worktree) and `<plan>` is `plan-` plus the 5-digit plan number from step 3 (for example `plan-00112`) (write rules: `../SHARED/DRAFT_AUTHORING.md`).

### 6. Present for review

Show the user the rendered issue, not the YAML:

```bash
marvin template render arch-plan --draft "<project-root>/.claude/cache/<plan>/arch-draft.yml"
```

Render and present the draft as `../SHARED/DRAFT_AUTHORING.md` describes (title, rendered markdown, `schema:` line), then take the user's changes the same way.

Read `../SHARED/LABELS.md`. Infer domain labels from the plan content. Present the rendered draft with proposed labels: "I'll apply: `plan:arch`, `status:upcoming`, `domain:backend` — correct?"

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

Then create the issue from the approved draft file, capturing the returned number and URL:

```bash
marvin issue create --template arch-plan --draft "<project-root>/.claude/cache/<plan>/arch-draft.yml" --label "<labels>"
```

`<labels>` follows the comma-joined-string rule in `../SHARED/DRAFT_AUTHORING.md` and contains only the labels that exist: `plan:arch`, `status:upcoming`, each domain label, and the source issue's type label (`bug` or `enhancement`) if it has one. Example: `--label "plan:arch,status:upcoming,domain:backend,enhancement"`.

The title comes from the draft, so do not pass `--title` or `--body`. Handle the exit code as `../SHARED/DRAFT_AUTHORING.md` describes.

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
