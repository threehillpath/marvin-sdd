---
name: impl-plan
description: Create a technical implementation plan from an architecture plan issue
argument-hint: <arch-plan-issue-number>
allowed-tools: Bash, Read, Glob, Grep, Agent
model: opus
---

Create a technical implementation plan from an approved architecture plan. The impl plan is a specification — what to build and why, not how to code it.

**Before starting**: Read `.claude/plan-workflow-config.md` for project configuration (repo, owner). Read `../SHARED/GLOSSARY.md` for naming and status conventions.

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
> Keep the total response under ~3000 words. Skip files that turn out to be trivial (constants, re-exports). If you find a file that should also be read but wasn't on the list, mention it by path with a one-sentence reason — don't read it.

Capture the digest. This becomes your reference material for drafting; you do not need to read the underlying files yourself unless the digest flags something that needs deeper inspection.

### 2. Ask clarifying questions if needed

Evaluate: component sequencing, schema changes, layer boundaries, edge cases, verification approach, TDD entry points. If non-obvious, ask before drafting. Group questions.

### 3. Draft the plan

Read `SUPPLEMENTS/CONVENTIONS.md` for what to include and exclude.

Render the impl plan template:

```bash
marvin template render impl-plan
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

Use the rendered skeleton as the structural frame for the draft, filling in each section with substantive content from the arch plan analysis.

**TDD**: Each component section must include a TDD Entry Point. The only exemption is for **rendered controls** — the JSX/template markup, styling, and rendering itself. All logic that lives inside a component (event handlers, derived state, validation, formatting, conditional-render predicates) must be extracted to a non-component module and given a TDD entry point. The litmus test: if it can be tested with the DOM removed, it is logic. See `SUPPLEMENTS/TDD.md` for full scope.

### 4. Present for review

Read `../SHARED/LABELS.md`. Infer domain labels from the plan content. Present the draft with proposed labels: "I'll apply: `plan:impl`, `status:upcoming`, `domain:backend` — correct?" Allow corrections before proceeding.

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

Then create the issue:

```bash
gh issue create --repo <repo> \
  --title "[PLAN-XXXXX] <Title>" \
  --body "<approved content>" \
  --label "plan:impl,status:upcoming,<domain-labels>"
```

### 6. Link to arch plan

```bash
gh issue comment $0 --repo <repo> \
  --body "Implementation plan created: #<new-issue> ([PLAN-XXXXX])"
```

### 7. Add to board as Ready

```bash
marvin board move <new-issue-number> ready
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

### 8. Confirm

Report: arch plan issue, new impl plan issue number and title, board status.

**Next step**: `/red-team-plan <new-issue-number>` to critique the plan with a fresh-context opus sub-agent before splitting it. Address blocking findings (or accept the risk), then `/phase-split <new-issue-number>`.
