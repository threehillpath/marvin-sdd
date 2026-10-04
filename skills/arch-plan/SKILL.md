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

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

Fill every key of the skeleton with substantive content from the arch analysis. In `title:`, replace `XXXXX` and `<Title>` with the real plan number and title. `Write` the filled draft to `<project-root>/.claude/cache/<plan>/arch-draft.yml`, where `<project-root>` is the repo root and `<plan>` is the lowercase plan number from step 3 (for example `plan-00112`); `Write` creates the directory. If the file already exists, `Read` it first (`Write` refuses to overwrite a file it has not read), then overwrite it. Use this same path in every command below. The draft rules below are the ones to follow. `--guidance` prints the quoting, `|` block, heading, comment, document-marker, code-fence and HTML rules plus the per-section guidance, but it does not mention the rules the loader enforces on keys, tabs, tags, anchors and aliases; those are in the list too. `skills/SHARED/CONFIG.md` describes the format:

- Section content is always a `|` block, never `>` or an inline value. A repeatable section is one list under one key, not the key repeated.
- The title and every metadata value are always double-quoted. Write `\"` for a quote and `\\` for a backslash inside them.
- Each key appears once. Indent with spaces, never tabs.
- No `#` or `##` heading lines in content; use `###` or deeper. Escape a literal `#` at the start of a line as `\#`.
- No YAML comments, no `---` or `...` at column 0, no tags, anchors or aliases.

### 6. Present for review

Show the user the rendered issue, not the YAML:

```bash
marvin template render arch-plan --draft <project-root>/.claude/cache/<plan>/arch-draft.yml
```

Stdout is the issue body as markdown. Paste the draft's `title:` and that markdown into your reply, because a Bash result is not shown to the user (see `../SHARED/RENDERING.md`), and ask for approval on the pasted text. If it exits 3 the draft does not conform: fix it per the findings on stderr and render again, within the retry limit in the create step. When the user asks for changes, edit the draft file and render again.

Read `../SHARED/LABELS.md`. Infer domain labels from the plan content. Present the rendered draft with proposed labels: "I'll apply: `plan:arch`, `status:upcoming`, `domain:backend` — correct?" Allow corrections before proceeding.

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
marvin issue create --template arch-plan --draft <project-root>/.claude/cache/<plan>/arch-draft.yml --label "plan:arch,status:upcoming,<domain-labels>,<source-issue-type-if-applicable>"
```

The title comes from the draft, so do not pass `--title` or `--body`. On success stdout is the new issue number, then its URL; capture both. Warnings on stderr are fine. Handle the exit code:

- **0** — created.
- **3** — the draft does not conform; nothing was created. The findings are on stderr, each naming a draft line and the fix. Fix the draft and run the same command again. Make at most 3 fix-and-retry attempts; if it still exits 3, show the user the findings and stop.
- **1** — a usage or operational error (for example an unreadable draft, or several usage problems listed together under a header like `issue create: 3 problems:`). Findings may be printed with it. Show stderr to the user and stop. Do not retry.
- **2** — configuration missing. Surface: "Configuration missing — run `/configure-plan-plugin` first." Do not retry.

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
