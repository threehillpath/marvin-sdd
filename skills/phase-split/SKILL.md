---
name: phase-split
description: Break an implementation plan into phases and create GitHub issues for each
argument-hint: <impl-plan-issue-number>
allowed-tools: Bash, Read, Write, Glob, Grep
model: opus
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
marvin issue tree $0
```

This returns one pipe-delimited line per node — `<kind> | #<number> | <state> | <status> | <title>` — with `kind` one of `arch`, `impl`, `phase`, `task`. Filter for lines where `kind` is `phase`. If zero `phase` lines are found (not the same as an empty result — `issue tree` always emits the target's own node), this plan predates sub-issue linking; fall back to:

```bash
marvin issue list --label "plan:phase" --title-prefix "[PLAN-XXXXX-" --state all
```

Note the trailing hyphen and **no closing bracket** — this is a true string-prefix match against phase titles like `[PLAN-XXXXX-1] ...`, unlike the closing-bracket form. This returns one pipe-delimited line per issue — `<number> | <state> | <labels-comma-joined> | <title>`.

For a multi-impl track (`[PLAN-XXXXX-A]`, `[PLAN-XXXXX-B]` — see `../SHARED/GLOSSARY.md`), apply the same rule one level deeper and use `--title-prefix "[PLAN-XXXXX-<suffix>-"` (e.g. `"[PLAN-00042-A-"`), so the fallback does not also match the sibling track's phases.

If any phase issues are found by either method, show the existing phases and stop: "Phase issues already exist for [PLAN-XXXXX]. Run `/move-issue <issue-number> in-progress` to start one, or delete the existing phase issues manually before re-splitting."

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

Get the empty YAML draft and the rules for filling it:

```bash
marvin template render impl-phase --skeleton
marvin template render impl-phase --guidance
```

If either command exits 1 (for example a malformed project override, or an unknown type), show stderr to the user and stop. Neither command reads the config, so neither returns exit 2.

Fill every key of the skeleton with phase-specific content, except that `tdd_entry_point` is optional: for a phase with no TDD entry point, write `None.` followed by the reason (an empty block leaves a bare `## TDD Entry Point` heading, which does not trigger review-phase's structural pre-check). In `title:`, replace `XXXXX`, `N` and `<Phase Title>` with the real plan number, phase number and title. The draft rules below are the ones to follow. `--guidance` prints the quoting, `|` block, heading, comment, document-marker, code-fence and HTML rules plus the per-section guidance, but it does not mention the rules the loader enforces on keys, tabs, tags, anchors and aliases; those are in the list too. `../SHARED/CONFIG.md` describes the format:

- Section content is always a `|` block, never `>` or an inline value. A repeatable section is one list under one key, not the key repeated.
- The title and every metadata value are always double-quoted. Write `\"` for a quote and `\\` for a backslash inside them.
- Every metadata value must be non-empty. `Implementation Plan` starts with the issue reference, `#<n>` (for example `#132 ([PLAN-00112])`; if it names a plan, that plan must be the title's), and `Plan Number` is the title's plan number without brackets (`PLAN-00112` for the title `[PLAN-00112-3] ...`).
- Each key appears once. Indent with spaces, never tabs.
- No `#` or `##` heading lines in content; use `###` or deeper. Escape a literal `#` at the start of a line as `\#`, except inside a code fence, where a `#` line is code and must stay unescaped (`\#` would print as is).
- No YAML comments, no `---` or `...` at column 0, no tags, anchors or aliases.

Once a phase's sections are filled in, `Write` it to that phase's own draft file, `<project-root>/.claude/cache/<plan>/phase-N-draft.yml`, where `<project-root>` is the repo root, `<plan>` is the lowercase PLAN-XXXXX number (for example `plan-00112`) and `N` is the phase number (for a multi-impl track use `phase-<suffix>-N-draft.yml`, e.g. `phase-A-1-draft.yml`); `Write` creates the directory. If the file already exists, `Read` it first (`Write` refuses to overwrite a file it has not read), then overwrite it. Use each phase's own path in every command below.

Write each phase's title and body together in its one draft file, never as two separate lists. Write every phase's draft first, then validate every one of them **before the first `issue create`**:

```bash
marvin template validate impl-phase --draft <project-root>/.claude/cache/<plan>/phase-N-draft.yml
```

`validate` makes no config or GitHub call and creates nothing. Exit codes:

- **0** — the draft conforms. Any warnings are printed on stdout after the `schema:` line; a warning alone does not fail the draft.
- **3** — the draft does not conform. The findings are on stdout, most naming a draft line, all saying how to fix it. Rewrite that draft file with `Write` to fix the findings and validate it again. Make at most 3 fix-and-retry attempts per draft; if it still exits 3, show the user the findings and stop, with no issue created.
- **1** — a usage or operational error (for example an unreadable draft). Show stderr to the user and stop. Do not retry.

Create issues only once every draft has exited 0.

Do not show phase bodies by default; the user approved the phase list in step 2. If any draft produced warnings, or the user asks, show that phase's rendered markdown and wait for approval before the first create:

```bash
marvin template render impl-phase --draft <project-root>/.claude/cache/<plan>/phase-N-draft.yml
```

Paste the draft's `title:` and that markdown into your reply, because a Bash result is not shown to the user (see `../SHARED/RENDERING.md`). If the user asks for changes, rewrite that draft file with `Write` and validate it again.

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

Then create the phase issues in phase order, each from its draft file, capturing the returned number and URL:

```bash
marvin issue create --template impl-phase --draft <project-root>/.claude/cache/<plan>/phase-N-draft.yml --label "<labels>"
```

`<labels>` is one comma-joined string of only the labels that exist: `plan:phase`, `status:upcoming`, and each domain label. Leave out any part that is absent; never leave an empty entry or a leading or trailing comma (marvin would pass an empty `--label` to `gh`). Example: `--label "plan:phase,status:upcoming,domain:backend"`.

The title comes from the draft, so do not pass `--title` or `--body`. On success stdout is the new issue number, then its URL; capture both. Warnings on stderr are fine. Handle the exit code:

- **0** — created.
- **3** — the draft does not conform; nothing was created. The findings are on stderr, most naming a draft line, all saying how to fix it. Rewrite the draft file with `Write` to fix the findings and run the same command again. Make at most 3 fix-and-retry attempts; if it still exits 3, show the user the findings and stop.
- **1** — a usage or operational error (for example an unreadable draft, or several usage problems listed together under a header like `issue create: 3 problems:`). Findings may be printed with it. Show stderr to the user and stop. Do not retry.
- **2** — configuration missing. Surface: "Configuration missing — run `/configure-plan-plugin` first." Do not retry.

If a create stops the run (exit 1, 2, or 3 after the retries), tell the user: which phase issues were already created (number and title for each) and whether `link-parent` ran for each; which phases were not created, with each one's draft file path; and that steps 3b (title/body check), 4 (phases-created comment) and 5 (board moves) did not run. That lets the split be finished by hand. Do not delete the created issues and do not start over: step 3's existing-phase check would refuse a second run.

Immediately after each phase issue is created, set a real GitHub-native sub-issue link so `marvin issue tree` can resolve this plan's hierarchy without relying on title matching:

```bash
marvin issue link-parent <new-phase-issue> $0
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

Capture each issue number as you go.

### 3b. Verify title/body pairing

Before moving to step 4, re-fetch every created issue and confirm each one's body actually describes its own title — not an adjacent phase's:

```bash
gh issue view <issue-number> --repo <repo> --json title,body
```

For each issue, check that the `## Objective` and `## Components` sections reference the same phase number and component(s) named in the title. If any issue's body describes a different phase, fix it immediately: rewrite that phase's draft file with `Write` and run `marvin issue edit <issue-number> --template impl-phase --draft <corrected-draft>` (it replaces the body and the title, and prints nothing on stdout when it succeeds) before proceeding — do not defer this to a later skill. You handle the exit code of the fix-up:

- **0** — fixed.
- **3** — the corrected draft does not conform; nothing was changed. The findings are on stderr. Rewrite the draft file with `Write` to fix them and run the same command again. Make at most 3 fix attempts; if it still exits 3, show the user the findings and stop.
- **1** — a usage or operational error. Show stderr to the user and stop. Do not retry.
- **2** — configuration missing. Surface: "Configuration missing — run `/configure-plan-plugin` first." Do not retry.

### 4. Post the phases-created comment

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
