---
name: impl-plan
description: Create a technical implementation plan from an architecture plan issue
argument-hint: <arch-plan-issue-number>
allowed-tools: Bash, Read, Write, Glob, Grep, Agent
model: opus
---

Create a technical implementation plan from an approved architecture plan. The impl plan is a specification — what to build and why, not how to code it.

**Before starting**: Read `.claude/plan-workflow-config.yml` for project configuration (repo, owner). Read `../SHARED/GLOSSARY.md` for naming and status conventions.

## Arguments

- `$0` — Arch plan issue number (the `[PLAN-XXXXX-ARCH]` issue)

## Steps

### 1. Fetch context

```bash
gh issue view $0 --repo <repo> --json number,title,body,comments
```

Extract the PLAN-XXXXX number and source issue reference from the arch plan. Fetch the source issue too.

From the arch plan and source issue, identify the **paths** of source files that are likely relevant — handlers, workers, repos, schema files, components, configs. List them; do not read them yourself. Reading every relevant file inline would balloon this conversation's context before the drafting step where it matters most.

### 1b. Delegate code digest to a sub-agent

Spawn an **Explore** subagent to read those files and return a structured digest. Use this prompt template:

> Goal: produce a digest that a planner will use to write a technical implementation plan. Do not draft the plan — just describe what currently exists.
>
> For each of the following files, return:
>
> - **Path**
> - **Purpose** (1 sentence)
> - **Key types / function signatures / exports** (signatures only, not bodies)
> - **Notable behavior** the planner needs to know — invariants, side effects, hidden coupling, error-handling patterns, tests that exist
> - **Likely change surface** for the work described below — which functions or types would need to be touched, added, or replaced
>
> Files:
> <bullet list of paths>
>
> Work to plan (from arch plan #<N>):
> <paste the arch plan body, or the relevant excerpt>
>
> Keep the digest to what the planner needs: signatures and notable behavior, never function bodies or long excerpts — it lands in the planner's context ahead of drafting. Skip files that turn out to be trivial (constants, re-exports). If you find a file that should also be read but wasn't on the list, mention it by path with a one-sentence reason — don't read it.

Capture the digest. This becomes your reference material for drafting; you do not need to read the underlying files yourself unless the digest flags something that needs deeper inspection.

### 2. Ask clarifying questions if needed

Evaluate: component sequencing, schema changes, layer boundaries, edge cases, verification approach, TDD entry points. If non-obvious, ask before drafting. Group questions.

### 3. Draft the plan

Read `SUPPLEMENTS/CONVENTIONS.md` for what to include and exclude.

Get the empty YAML draft and the rules for filling it:

```bash
marvin template render impl-plan --skeleton
marvin template render impl-plan --guidance
```

If either command exits 1 (for example a malformed project override, or an unknown type), show stderr to the user and stop. Neither command reads the config, so neither returns exit 2.

Fill every key of the skeleton with substantive content from the arch plan analysis. In `title:`, replace `XXXXX` and `<Title>` with the real plan number and title. The draft rules below are the ones to follow. `--guidance` prints the quoting, `|` block, heading, comment, document-marker, code-fence and HTML rules plus the per-section guidance, but it does not mention the rules the loader enforces on keys, tabs, tags, anchors and aliases; those are in the list too. `../SHARED/CONFIG.md` describes the format:

- Section content is always a `|` block, never `>` or an inline value. A repeatable section is one list under one key, not the key repeated (`component:` is a list of entries, each with `name: ""` and `content: |`).
- The title and every metadata value are always double-quoted. Write `\"` for a quote and `\\` for a backslash inside them.
- Each key appears once. Indent with spaces, never tabs.
- No `#` or `##` heading lines in content; use `###` or deeper. Escape a literal `#` at the start of a line as `\#`.
- No YAML comments, no `---` or `...` at column 0, no tags, anchors or aliases.

**TDD**: Each component section must include a TDD Entry Point. The only exemption is for **rendered controls** — the JSX/template markup, styling, and rendering itself. All logic that lives inside a component (event handlers, derived state, validation, formatting, conditional-render predicates) must be extracted to a non-component module and given a TDD entry point. The litmus test: if it can be tested with the DOM removed, it is logic. See `SUPPLEMENTS/TDD.md` for full scope.

Once every section is filled in, `Write` the filled draft to `<project-root>/.claude/cache/<plan>/impl-draft.yml`, where `<project-root>` is the repo root and `<plan>` is the lowercase PLAN-XXXXX number from the arch plan (for example `plan-00112`); `Write` creates the directory. If the file already exists, `Read` it first (`Write` refuses to overwrite a file it has not read), then overwrite it. Use this same path in every command below.

### 4. Present for review

Show the user the rendered issue, not the YAML:

```bash
marvin template render impl-plan --draft <project-root>/.claude/cache/<plan>/impl-draft.yml
```

Stdout is the issue body as markdown. Paste the draft's `title:` and that markdown into your reply, because a Bash result is not shown to the user (see `../SHARED/RENDERING.md`), and ask for approval on the pasted text. If it exits 1 (for example a malformed project override), show stderr to the user and stop. If it exits 3 the draft does not conform: rewrite the draft file with `Write` to fix the findings on stderr and render again. Make at most 3 fix-and-render attempts per round of user changes; if it still exits 3, show the user the findings and ask how to proceed. When the user asks for changes, rewrite the draft file with `Write` and render again.

Read `../SHARED/LABELS.md`. Infer domain labels from the plan content. Present the rendered draft with proposed labels: "I'll apply: `plan:impl`, `status:upcoming`, `domain:backend` — correct?" Allow corrections before proceeding.

See `../SHARED/RENDERING.md` for rendering guidance. Ask for approval on both content and labels; iterate until confirmed.

### 5. Create the GitHub issue

Ensure all required labels exist before creating the issue:

```bash
marvin label ensure --builtins
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

For any domain labels not covered by `--builtins`, ensure each one individually:

```bash
marvin label ensure "<name>" --description "<desc>" --color "<hex>"
```

Then create the issue from the approved draft file, capturing the returned number and URL:

```bash
marvin issue create --template impl-plan --draft <project-root>/.claude/cache/<plan>/impl-draft.yml --label "plan:impl,status:upcoming,<domain-labels>"
```

The title comes from the draft, so do not pass `--title` or `--body`. On success stdout is the new issue number, then its URL; capture both. Warnings on stderr are fine. Handle the exit code:

- **0** — created.
- **3** — the draft does not conform; nothing was created. The findings are on stderr, each naming a draft line and the fix. Rewrite the draft file with `Write` to fix the findings. If the fix changes only how the draft is written (quoting, escaping, block style, heading depth) and not what it says, run the same command again. If it changes what the draft says (content added, removed or reworded, or the title), render the draft again, show the user, and get approval before running the command again. Make at most 3 fix attempts per round of user changes; if it still exits 3, show the user the findings and stop.
- **1** — a usage or operational error (for example an unreadable draft, or several usage problems listed together under a header like `issue create: 3 problems:`). Findings may be printed with it. Show stderr to the user and stop. Do not retry.
- **2** — configuration missing. Surface: "Configuration missing — run `/configure-plan-plugin` first." Do not retry.

### 6. Link to arch plan

```bash
gh issue comment $0 --repo <repo> \
  --body "Implementation plan created: #<new-issue> ([PLAN-XXXXX])"
```

Also set a real GitHub-native sub-issue link so `marvin issue tree` can resolve this plan's hierarchy without relying on title matching:

```bash
marvin issue link-parent <new-issue> $0
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

### 7. Add to board as Ready

```bash
marvin board move <new-issue-number> ready
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

### 8. Confirm

Report: arch plan issue, new impl plan issue number and title, board status.

**Next step**: `/red-team-plan <new-issue-number>` to critique the plan with a fresh-context opus sub-agent before splitting it. Address blocking findings (or accept the risk), then `/phase-split <new-issue-number>`.
