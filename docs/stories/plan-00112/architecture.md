# [PLAN-00112-ARCH] Schema-checked plan issue creation and edits

Source: https://github.com/threehillpath/marvin-sdd/issues/131

**Source Issue:** #112
**Plan Number:** PLAN-00112
**Author:** Bryan Walker (drafted with Claude)
**Status:** upcoming
**Date:** 2026-10-02

> **Revised 2026-10-02** before impl-plan. Skills now draft issues as YAML (sections keyed by schema id, markdown in block scalars) that marvin checks and renders. Markdown parsing stays for edits and for checking existing issues. Both inputs produce one canonical section map, and only that map is checked. This replaces the original "check finished markdown only" decision (see Architectural Decisions §1). Also revised: no JSON on any model-facing path (Cross-Cutting Concerns).

## Problem Statement

The four issue-drafting skills (`arch-plan`, `impl-plan`, `phase-split`, `quick-task`) call `marvin template render <type> --skeleton` to get empty headings, write each section as prose, and hand the result to `marvin issue create --body-file` as an opaque string. Nothing checks that the finished issue matches its schema, either when it's created or when a skill later rewrites its body (phase-split's title/body drift fix-up uses a raw `gh issue edit`). Whether an issue conforms depends entirely on the drafting model being careful.

The schemas already declare everything a structural check needs: ordered metadata keys, sections with `required` / `repeatable` / `numbered` flags, and a `title_prefix`. `template.Render` already enforces part of this, but only for structured `--sections` JSON input, which no skill uses. So the rules exist but are never applied to real skill output.

The title side has a related gap. `marvin parse title` can't tell an arch title (`[PLAN-XXXXX-ARCH]`) from an impl title (`[PLAN-XXXXX]`), since both return the same fields, and it doesn't recognise `[TASK-XXXXX]` titles at all (`found: false`).

## Scope

**In scope:**
- **Structured drafting:** skills write each issue as a YAML draft. A draft holds the title, the metadata keys and values, and the sections keyed by schema `id`, with each section's markdown in a block scalar. Numbered repeatable sections (impl-plan's components and verification steps) are YAML lists. marvin checks the draft and renders it to the markdown body. The schema describes both the draft and the rendered body.
- **One conformance check per schema type** (arch-plan, impl-plan, impl-phase, quick-task) that takes a drafted body and title and returns a result made of **errors** and **warnings**:
  - Errors (block the operation): a required section is missing; a required section's heading is present but has no content; a non-repeatable section appears more than once; a required metadata key is missing; the title's identifier doesn't match the schema type (for example, an `impl-phase` body with an `-ARCH` title).
  - Warnings (operation goes ahead, reported on stderr): a `##` heading the schema doesn't define; sections out of schema order.
- **Checked creation:** `marvin issue create --template <type>` accepts either a YAML draft (the skills' path) or a markdown body (for callers still writing prose), runs the check, and refuses to create the issue on any error. Without the flag, `issue create` behaves exactly as today.
- **Checked edits:** a checked body-edit path (for example `marvin issue edit <n> --template <type>`) that takes the same two inputs and applies the same rules. phase-split's drift fix-up moves from raw `gh issue edit` onto it, re-rendering from its corrected YAML draft.
- **Standalone check:** `marvin template validate <type>` runs the same check against a YAML draft or a markdown body without touching GitHub. That lets a skill check a draft before using it, and lets anyone check an existing issue's body.
- **Title identifier kinds:** title parsing reports which kind of identifier a title carries (arch, impl, phase, task) and recognises `[TASK-XXXXX]`, so the check can match it against the schema type.
- **Skill adoption:** all four drafting skills switch from filling in a `--skeleton` to writing a YAML draft, and pass `--template` on every create (and phase-split on its fix-up edit). All four get the `Write` tool to write the draft file, replacing the Bash heredoc scratch-file workaround. User review still shows the rendered markdown, not raw YAML.
- **Housekeeping:** `marvin template render --help` lists `quick-task` (currently omitted).

**Out of scope:**
- Grading section *content* (is the objective clear, can the success criteria fail). That's #130. Conforming does not mean good, and the first pass isn't meant to be perfect.
- Enforcing the schemas' free-text `validation.rules` bullets (checkbox format, component sub-sections) as machine checks. Deferred to #130.
- Checking labels against `default_labels`.
- Making `--template` mandatory, or refusing `plan:*`-labelled issues created without it. The flag is opt-in, and the skills always use it.
- Checking issue comments (wrap-up comments, red-team critiques) against any format.

## Domain Model Impacts

- **The schema becomes the single conformance contract.** Today a schema drives rendering only. After this, the same schema definition drives rendering, creation checks, edit checks, and standalone checks. "Required", "repeatable" and "numbered" mean one thing everywhere, so `render` and the checks can't disagree about what a valid body is.
- **The section map is the canonical intermediate form.** It holds the metadata plus each section's content keyed by schema `id`, with lists for repeatable sections. Everything else is a way in or out of it: a YAML draft is read into it, a markdown body is parsed into it, and rendering produces markdown from it. The conformance check only ever sees the section map, so it doesn't depend on which way the content came in.
- **New concept: the YAML draft.** It's the authoring format for model-written issues, a file-level form of the section map plus the title. It's a working artifact, not a stored record. GitHub keeps the rendered markdown, and the draft isn't embedded in the issue, to avoid two copies of the content that can drift apart.
- **New concept: a conformance result.** It's a list of findings, each with a severity (error or warning), the section or metadata key it concerns (by schema `id`), and a message saying what to fix. The result is the stable output of the check. Its primary form is compact plain text that an agent can act on directly: one line per finding with severity, section id and the fix. `--json` exists only as an escape hatch for non-agent tooling, consistent with marvin's plain-text-by-default output (PLAN-00041), and skills never consume it. Its shape should be reusable by the #130 grader, which will add content findings alongside structural ones.
- **Title identifier kind.** Title parsing gains an explicit kind (arch / impl / phase / task) and Task support. This is a small extension of an existing concept, not a new one. Anything that parses titles today keeps its current fields.
- **Numbered repeatable sections.** impl-plan's `component` and `verification_steps` sections render as `## 1. <name>`, `## 2. <name>`, with the heading text coming from the content, not the schema. In a YAML draft they're lists of name and content entries. When markdown is parsed, they're matched by their numbered form, not by literal heading text.

## Integration Points

- **`marvin template` (render, and new validate):** shares the conformance definition. Schema resolution keeps today's precedence (project override `.claude/plan-workflow-templates/{type}.yml` via CWD walk, then the embedded built-in), and every check reports which schema it used.
- **`marvin issue create`:** gains `--template <type>`. Existing flags and the default path are unchanged.
- **`marvin issue edit` (new):** a checked body edit through the `gh` client wrapper, with the same `exec.Runner` injection as the other issue commands so it's unit-testable with `exectest`.
- **`marvin parse title`:** gains identifier kind and `[TASK-XXXXX]` parsing.
- **Skills:**
  - `arch-plan`, `impl-plan`, `phase-split`, `quick-task`: create calls pass `--template`, frontmatter adds `Write`, and the heredoc instructions are replaced.
  - `phase-split`: its fix-up edit uses the checked edit path.
- **Docs:** the CLAUDE.md marvin subcommand list, and `skills/SHARED/CONFIG.md` where it describes template precedence (validation follows the same precedence).

## Cross-Cutting Concerns

- **No silent failures (CLAUDE.md rule):**
  - A check that can't run (unknown type, unreadable body file, malformed schema or override) is an error with a non-zero exit, never a pass.
  - A rejected create or edit must make no GitHub call at all, so it can't leave a half-created issue behind.
  - Warnings go to stderr only; stdout stays data.
  - Reporting which schema was used makes a stale or misplaced project override visible, rather than looking identical to the built-in.
- **Project overrides:** a consuming project's override schema is checked against exactly the same rules as a built-in. A malformed override is a reported error, not a silent fallback to the built-in.
- **Exit-code contract (0 / 1 / 2):** conformance failures must fit the existing contract. Whether they need a code distinct from other operational errors is an open question below.
- **AI-first I/O, no JSON for Claude Code:**
  - Everything a skill writes for marvin (drafts) or reads from marvin (check results, errors) is YAML or compact plain text, never JSON.
  - `render --sections` JSON input is replaced by the YAML draft. No JSON authoring path remains.
  - `--json` output stays only as an escape hatch for non-agent callers, as elsewhere in marvin.
- **Backward compatibility:** callers that don't pass `--template` see no behavior change. Existing plan issues aren't re-checked retroactively.
- **Stale plugin installs:** skills will start calling flags that only exist in the new marvin. A skill paired with an older binary must fail loudly (an unknown-flag error) rather than skip the check.

## Architectural Decisions

1. **The section map is canonical, with two ways in, and skills draft in YAML.**
   - **YAML drafts for authoring.** Content keyed by schema `id` is checked exactly: there's no heading text to parse, and a model can't introduce a case or wording drift in a heading. YAML block scalars hold multi-line markdown without escaping, which removes the objection to structured input. JSON (the current `render --sections` format) is rejected for that reason, and more generally because JSON is token-heavy and not AI-first for anything Claude Code reads or writes.
   - **Markdown parsing for everything already on GitHub.** GitHub stores rendered markdown, and that's what skills and people edit. Checking an existing issue or an edit needs a parser no matter what. Embedding the YAML source in the issue for lossless round-tripping was rejected: two copies of the content would drift apart, and a web-UI edit would bypass the source copy without anyone noticing.
   - **Only the section map is checked.** Both inputs produce a section map, and that's the only thing the check sees, so the two paths can't apply different rules.
2. **One conformance definition, many entry points.** Render, validate, checked create and checked edit all call the same check on the section map. `Render`'s existing required-section and non-repeatable checks move into it rather than being duplicated.
3. **Two severities with a fixed split.** Errors cover things that make an issue structurally wrong for downstream skills (missing, empty or duplicated required content, missing metadata, wrong identifier kind). Warnings cover tolerable drift (extra headings, order). The first pass deliberately doesn't treat imperfection as failure.
4. **Opt-in at the CLI, mandatory in skill prose.** `--template` is a flag, not a default. Enforcement comes from every drafting skill passing it, which keeps marvin usable for ad-hoc and bug-report issues without coupling labels to schemas.
5. **Edits are checked like creates.** Any skill that rewrites a plan or Task body does it through the checked edit path, so a body can't drift out of conformance after creation through a marvin-mediated edit.

## ADR Candidates

- [ ] Schemas as the single conformance contract for rendering and checking. Sets the pattern for any future model-authored artifact that gets a schema (PR bodies, review findings).
- [ ] Model-authored artifacts are drafted as structured YAML (a canonical intermediate form) and rendered for storage. Stored markdown is parsed back into the same form for checking. This trade-off will recur for every artifact a skill drafts.
- [ ] AI-first I/O for model-facing contracts: YAML for structured input, compact plain text for output, and JSON only as a non-agent escape hatch. This extends PLAN-00041's output direction to inputs and to future contracts.
- [ ] Error/warning severity model for conformance results. The #130 grader and future checks should report findings in the same shape.

## Constraints and Trade-offs

- **The parser is strict about structure but permissive about prose.** Section content is opaque to this story. That means a body can pass while still being wrong in substance, for example phase-split's original drift case, where an Objective describes a different phase than the title. Catching that is #130's job. This story only guarantees the title kind and the section structure.
- **Heading matching is literal, on the markdown path only.** YAML drafts use schema ids, so headings can't drift. When parsing markdown (edits, existing issues), a reworded heading (`## Success criteria` vs `## Success Criteria`) counts as a missing required section plus an unknown heading. Literal matching is the safe default because downstream skills (wrap-phase, finish-impl) look for exact headings.
- **Drafting gets heavier for the skills.** Moving from filling in a skeleton to writing YAML changes the drafting instructions in all four skills, which makes the skill-prose part of this story bigger. YAML has pitfalls: a value containing `: ` or `#` breaks if it isn't quoted, and block scalar indentation must be consistent. The draft format should keep the model away from them, for example by always using block scalars for section content and quoted values for metadata. marvin must report YAML parse errors with the line and the fix, not a raw parser message.
- **A draft and a parsed body may not round-trip byte for byte.** Whitespace and trailing newlines can differ. The guarantee is that a section map rendered and then parsed back is equivalent, not that the text is identical.
- **New flags tie skills to the binary version.** Skill prose and marvin must ship together. The plugin's SessionStart build and `deploy.sh` already rebuild marvin, but a stale install with an old binary will fail at the first `--template` call. That's loud, but it's a hard stop.
- **Edit coverage is only as good as skill adoption.** Edits made outside marvin (manual `gh issue edit`, the web UI) aren't checked. This story covers every skill-driven edit, not every edit.

## TDD Strategy

**Entry point:** a Go unit test in the template package. Given the built-in `impl-phase` schema and a section map that contains every required section except `verification`, the check returns exactly one error naming the `verification` section. The same map with `verification` restored returns no errors.

Further anchors, all deterministic and network-free:
- **Both inputs, same verdict:** the same missing-`verification` case, supplied as a YAML draft and as a markdown body, produces the same single error.
- **Round-trip property:** for each of the four built-in schemas, a valid YAML draft rendered to markdown and parsed back yields an equivalent section map, and passes the check with zero errors and zero warnings. This is what proves the YAML path, the markdown path and render can't disagree.
- **Numbered sections:** an impl-plan draft with a two-item `component` list renders to `## 1. …` / `## 2. …` and parses back to the same two items. A draft or body with no components fails on `component`.
- **YAML errors:** a draft with an unquoted `: ` in a metadata value, or a block scalar with broken indentation, fails with an error naming the line. It never passes, and never gets silently truncated.
- **Severity split:** an unknown heading or out-of-order sections produce warnings and exit 0. An empty required section produces an error.
- **Titles:** an `impl-phase` body with a `[PLAN-00112-ARCH]` title fails on identifier kind, and `[TASK-00091] X` parses as kind `task`.
- **Checked create/edit (via `exectest`):** a non-conforming body makes no `gh` call and exits non-zero, and a conforming body makes exactly the expected call.
- **Overrides:** a project override schema in a temp dir is checked in place of the built-in, and the result reports the override as its origin. A malformed override is an error.

The skill-prose changes (flags, `Write` frontmatter, heredoc removal) have no TDD entry point. They fall under review-phase's structural pre-check for skill-prose-only phases.

## Open Questions

- **Exit code for conformance failure:** reuse `1` (operational error), or does a skill need to tell "your draft is non-conforming, fix it and retry" apart from "marvin or gh failed"? A distinct code would extend the 0/1/2 contract. A distinct code would let a skill branch on the code alone, without parsing output.
- **Edit type inference:** should `issue edit --template` be explicit only, or may it infer the type from the existing issue's title identifier kind? Inference is convenient, but a mismatch then becomes a check finding rather than a usage error.
- **Metadata value cross-checks:** should the check also confirm that metadata values agree with the title (for example, `**Plan Number:**` matches the title's `PLAN-XXXXX`)? It's cheap and catches real drift, but it moves toward content checking.
- **Heading case sensitivity (markdown path only):** keep literal matching (current decision), or accept case-insensitive matches with a warning?
- **Draft flag shape:** one input flag that detects YAML vs markdown, or separate flags (for example `--draft` vs `--body-file`)? Separate flags are explicit and avoid guessing. One flag is simpler for callers.
- **`--skeleton` shape:** should `--skeleton` emit an empty YAML draft (keys and block-scalar placeholders, with schema guidance as comments) instead of empty markdown headings?
- **Help text source:** fix `render --help` by hand, or generate the type list from the embedded schemas so it can't drift again? This overlaps #128's generated contract reference.
