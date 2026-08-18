# Retrospective Synthesis Rubric

You are synthesizing a durable retrospective for a completed implementation story. Your output is written verbatim (by the orchestrator, via the `Write` tool) to `docs/stories/<plan>/retrospective.md` — a permanent record read by humans long after the story's issues and PRs have scrolled off the board. Your job is to **synthesize**, not concatenate: read across every input and produce narrative prose grouped by theme, not a per-source dump of each comment's own structure.

## Inputs

You will be given, inlined in your prompt:

1. Zero or more **wrap-up comments** — one per phase, each the body of a `wrap-phase`-authored comment on the impl plan issue, identified by the prefix `## Phase wrap-up: [PLAN-XXXXX-`. Each covers one phase's decisions, scope changes, deferred items, and corrections (see `../../wrap-phase/SUPPLEMENTS/COMMENT_TEMPLATE.md` for their template).
2. At most one **red-team critique comment** — the body of a `red-team-plan`-authored comment on the impl plan issue, identified by the prefix `## Plan Red-Team — verdict:`. May be absent (it's optional). Covers blocking findings and concerns raised against the plan before implementation began.

The orchestrator has already filtered these (deduped by ident, most recent kept) before inlining them in your prompt — treat every comment you are given as the input set, in full.

## Synthesis instructions

- Read every comment in full before writing anything.
- Group your output **by theme or topic**, not by source comment. A single decision, correction, or deferred item mentioned across multiple phases should be synthesized into one entry, not repeated once per phase.
- Cite which phase (or the red-team critique) each point came from, using its `[PLAN-XXXXX-N]` ident inline in the prose (e.g. "In `[PLAN-00055-1]`, the team chose to..."). This is the traceability mechanism — there are no footnotes or links back to GitHub.
- Write narrative prose — full sentences, not a re-listing of the source bullets. A reader should be able to tell this was synthesized by someone who read everything, not extracted mechanically.
- **Do not copy a source comment's own `###`-level sub-headings verbatim into your output.** The source wrap-up comments use `### Decisions`, `### Scope changes`, `### Deferred / watch items`, and `### Corrections (N)` as their own internal structure — your output must not reproduce those exact strings anywhere in its body text. Your five `##`-level headings (below) are a different, higher level of structure than theirs, and your prose under each should not restate their bullet lists as-is.
- Be honest and specific. If a phase's wrap-up says a decision was made for a specific reason, carry that reason into your synthesis — don't flatten it to a generic statement.

## Output format

Return **only** the retrospective markdown body — no surrounding prose, no code fence. Use exactly these five `##`-level section headings, in this order, when the corresponding source material exists:

```markdown
## Decisions

<synthesized prose, citing phase idents>

## Scope changes

<synthesized prose, citing phase idents>

## Deferred / watch items

<synthesized prose, citing phase idents>

## Corrections

<synthesized prose, citing phase idents>

## Plan critique

<synthesized prose from the red-team comment, if one was supplied>
```

**Omit a section entirely — heading and body — when its source data is absent.** Do not render an empty or "(none)" section. For example: if no phase's wrap-up comments contained any corrections, omit `## Corrections` entirely; if no red-team comment was supplied, omit `## Plan critique` entirely. A retrospective with nothing notable to report in any category is a valid (if unusual) result — in that case, return a single short paragraph noting the absence of notable history, with no section headings at all.

## Named fixture

Use this fixture to dry-run the rubric before wiring it into the live `finish-impl` step. Write the result to the scratch path `/tmp/retro-fixture.md` — **never** to `docs/stories/<plan>/retrospective.md`, which is the real story's permanent record and is gated by `finish-impl`'s skip-if-exists check.

**Fixture input — two synthetic wrap-up comments, no red-team comment:**

Comment 1:

```markdown
## Phase wrap-up: [PLAN-00055-1] Fixture Phase One

Merged via #999.

### Corrections (1)

- **FIXTURE-CORRECTION-A**: the original approach missed an edge case → fixed by adding a guard clause.
```

Comment 2:

```markdown
## Phase wrap-up: [PLAN-00055-2] Fixture Phase Two

Merged via #1000.

### Decisions

- **FIXTURE-DECISION-B** — chosen because it kept the interface simpler than the alternative.
```

**Expected properties of `/tmp/retro-fixture.md`:**

- Contains the literal string `FIXTURE-CORRECTION-A` (from Comment 1's correction).
- Contains the literal string `FIXTURE-DECISION-B` (from Comment 2's decision).
- Cites both `[PLAN-00055-1]` and `[PLAN-00055-2]` by ident somewhere in the prose.
- Omits `## Plan critique` entirely (no red-team comment was supplied).
- Contains none of the four source sub-headings verbatim: `### Decisions`, `### Scope changes`, `### Deferred / watch items`, `### Corrections (1)` (or any `### Corrections (N)`).
