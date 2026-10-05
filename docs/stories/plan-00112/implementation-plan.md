# [PLAN-00112] Schema-checked plan issue creation and edits

Source: https://github.com/threehillpath/marvin-sdd/issues/132

**Objective:** Make every skill-created or skill-edited plan issue pass one schema-driven conformance check. Skills draft issues as YAML, marvin checks the draft and renders it to markdown, and the same check runs on markdown bodies, on edits, and on demand.
**Architecture Plan:** #131 ([PLAN-00112-ARCH])
**Source Issue:** #112
**Author:** Claude (impl-plan)
**Status:** Draft
**Last Updated:** 2026-10-02

> **Revised 2026-10-02 after the red-team critique** (comment on this issue), addressing B1–B4 and C1–C11:
> - The YAML loader works on `yaml.Node` and turns yaml.v3's silent truncation into errors (B1, C5, C6).
> - The build order is now acyclic and the components were regrouped:
>   - The old Components 5 and 6 merged into one CLI component.
>   - The render rewiring moved into the YAML component.
>   - Skills form their own phase.
>
>   (B2, C7, C10)
> - `issue edit` gets success-path tests and reads its title from the configured repo (B3, B4).
> - The section map now carries the title (C2).
> - Title-kind derivation is spelled out (C1).
> - `task_number` matches GLOSSARY (C3).
> - Classification uses only the leading bracket (C4).
> - Override fields are required, and a missing one is a loud error (C8).
> - Inline `--body` is rejected with `--template` (C9).
> - phase-split validates every draft before creating anything (C11).
>
> **Revised again after the round-2 red-team**, addressing B1 and C1–C7:
> - The YAML loader rejects:
>   - repeated keys,
>   - a comment cut on *any* value,
>   - non-`|` section content,
>
>   and gives targeted fixes for unquoted titles, inner quotes and backslashes (B1, C2).
> - Loader findings sit in the Result under a `draft` location and exit 3 (C1).
> - A title with no identifier is an error, and the classifier returns `(Kind, bool)` (C3).
> - `parse title` keeps `found` and the plan fields on `PlanIdent` (C4).
> - phase-split's "compose immediately before creating" rule is reworded (C5).
> - Stale kind and exit-code docs are in scope (C6).
> - `render_test.go` and the override fixture are assigned to components (C7).

## Scope

**In scope:**
- Title identifier kinds (arch / impl / phase / task) and `[TASK-XXXXX]` parsing. `marvin parse title` gains a `kind:` line and Task fields.
- A canonical section map (title, metadata, sections), the conformance check over it, and a conformance result made of findings, with its plain-text format.
- A YAML draft loader built on `yaml.Node`, with line-numbered, actionable errors, and a markdown body parser. Both produce the section map.
- `Render` working from the section map, with its existing required and non-repeatable checks moved into the shared check. `--skeleton` emits an empty YAML draft. The JSON `--sections` and `--meta` flags are removed.
- Exit code `3` for a non-conforming draft.
- `marvin template validate <type>` and `marvin template render <type> --draft <file>`. The schema type list in help text is generated from the embedded schemas.
- `marvin issue create --template <type>` and a new `marvin issue edit <n> --template <type>`. Both take `--draft <file.yml>` or `--body-file <file.md>`.
- Adoption in four skills (`arch-plan`, `impl-plan`, `phase-split`, `quick-task`): YAML drafts, the `Write` tool, `--template` on every create, validation before creation, and phase-split's fix-up moved onto `issue edit`.
- Doc updates:
  - CLAUDE.md: exit-code contract and subcommands.
  - `skills/SHARED/CONFIG.md`: template precedence for validate, create and edit; the draft format; the required schema fields and a migration note for overrides.

**Out of scope:**
- Content grading, and enforcing the schemas' free-text `validation.rules`. Both are #130.
- Checking labels against `default_labels`.
- Making `--template` mandatory, or refusing unflagged `plan:*` issues.
- Re-checking existing issues retroactively.
- Edits made outside marvin (the web UI, raw `gh issue edit`).
- Moving the findings contracts (review, red-team, drift) off JSON. That's a separate follow-up.

**Decisions taken at impl-plan time (resolving the arch plan's open questions):**
- **Exit code:** a conformance failure, including a YAML draft that won't parse, exits `3`. A check that can't run (unknown type, unreadable file, malformed schema) exits `1`. Missing config exits `2`.
- **Edit type:** the type is explicit only (`--template` is required on `issue edit`). It is never inferred from the existing title.
- **Metadata cross-checks:** yes, as errors. Every finding must be specific enough for an agent to fix the draft without a second lookup (see Component 2).
- **Heading matching:** literal and case-sensitive, on the markdown path only.
- **Input flags:**
  - `--draft <file.yml>` and `--body-file <file.md>` are separate and mutually exclusive. marvin never sniffs the format.
  - With `--template`, the inline `--body` flag is a usage error (exit 1).
- **`--skeleton`:** emits an empty YAML draft, with schema guidance as YAML comments.
- **Help text:** the type list is generated from the embedded schemas.

## 1. Title identifier kinds

**Files:** `tool/internal/parse/parse.go`, `tool/internal/names/names.go` (Kind enum reuse), `tool/internal/cli/handlers.go` (`runParseTitle`), `tool/internal/issue/tree.go` (`kindOf`)

**Specifications:**
- **Classifier:** a new function that classifies a title into the existing four-value `names.Kind` (Arch, Impl, Phase, Task), returning `(names.Kind, bool)`. The bool is the only signal of "not found". Callers never rely on the zero value, because `names.Kind`'s zero value is `Arch`. No third kind enum is added. If importing `names` from `parse` would create a cycle, the enum moves to `parse`, and `names.Kind` becomes an alias of it.
- **Leading bracket only:** classification looks only at the bracket token at the start of the title (optional leading whitespace, then `[`), the same anchoring `TitleSlug` already uses. A later bracket in the title never affects the kind. Rules for that leading token:
  - `[PLAN-XXXXX-ARCH]` is arch.
  - `[PLAN-XXXXX]` and `[PLAN-XXXXX-A]` (a letter suffix) are impl.
  - `[PLAN-XXXXX-N]` and `[PLAN-XXXXX-A-N]` (numeric N) are phase.
  - **Suffix letters:** accepted with the same rules `PlanIdent` uses (any length, case-insensitive, normalized to upper case), so the two never disagree on a leading bracket.
  - `[TASK-XXXXX]` is task.
  - Anything else, including no leading bracket, is not found.
- `[TASK-XXXXX]` parsing returns the task number.

**Behavior:**
- `PlanIdent`, `Ident` and the existing `parse.KindImpl`/`parse.KindArch` semantics are unchanged. `pr/pr.go` relies on them, and so does the convention that `KindImpl` with `Phase != 0` means a phase.
- **`issue/tree.go` `kindOf`:**
  - If the classifier matches the leading bracket, `kindOf` uses its kind, and a task title maps to `task`. This is a deliberate change: today a TASK sub-issue under a plan is mislabelled `impl`.
  - Otherwise `kindOf` falls back to today's `PlanIdent`-based mapping, unchanged, including `impl` for an unparseable title.
  - Tree membership stays on `PlanIdent`.
  - So for every non-task title, `issue tree` output is byte-identical to today's.
- **`marvin parse title` output** adds lines but never renames or removes any. Seven skills read its existing keys.
  - **Rule:** `found:` and the plan fields stay driven by `PlanIdent`, exactly as today. `kind:` is printed only when the classifier matches the leading bracket.
    - `Notes on [PLAN-00042-1]` still prints `found: true` with plan fields and no `kind:` line.
    - `[TASK-…]` is the one case where `found: true` comes from the classifier.
  - A plan title whose leading bracket classifies prints its current fields plus `kind: arch|impl|phase`.
  - A task title prints `found: true`, `kind: task`, `task: <n>`, `task_number: TASK-XXXXX` and `slug:`. `task_number` comes from `names.TaskNumber`, uppercase, matching `names derive --task` and GLOSSARY. The plan-only fields are omitted.
  - `--json` gains the same fields.

**TDD Entry Point:** a table test in `parse/parse_test.go`. Cases:

| Title | Expected kind |
|---|---|
| `[TASK-00091] Fix X` | task, number 91 |
| `[PLAN-00112-ARCH] X` | arch |
| `[PLAN-00112-A-2] X` | phase |
| `[TASK-00140] Fix regression from [PLAN-00112-3]` | task |
| `[PLAN-00042-a] X` | impl (suffix `A`) |
| `Fix X` | not found |
| `[PLAN-XXXXX-ARCH] X` | not found (unreplaced placeholder) |

Plus a CLI test that `parse title "Notes on [PLAN-00042-1]"` prints today's `found: true` and plan fields with no `kind:` line.

## 2. Section map, conformance check, and conformance result

**Files:**
- `tool/internal/template/`: new files hold the section map, the check, findings and result formatting. The schema struct in `render.go` is widened.
- The four built-in schemas: gain `named`.
- `tool/internal/cli/template_render_test.go`: `overrideSchemaFixture` gains `title_prefix` and `named` in this component, so `go test ./...` stays green once the stricter schema loading lands here.

**Specifications:**
- **Schema model:** the parsed schema adds `type` and `title_prefix`, plus a new section field `named`.
  - `named: true` on a numbered section means its instance headings come from content. impl-plan's `component` sets it.
  - Every other numbered section renders its literal schema heading, so impl-plan's `verification_steps` sets `named: false` and renders `## N. Verification Steps`, which matches existing impl plans.
- **Required schema fields:** `title_prefix` is required on every schema. `named` is required (true or false) on every numbered section. If either is missing, the schema is malformed: every command that loads it exits 1, including `render --skeleton`. The message names the schema origin, the field and the fix, e.g. `project override <path>: section "component" is numbered but has no "named" field. Add "named: true" if headings come from content, else "named: false".` All four built-in schemas are updated to carry `named`.
- **Expected title kind:** derived by classifying the schema's `title_prefix` with Component 1's classifier, after substituting placeholders:
  - every run of `X` inside the bracket becomes `0`s of the same length,
  - a standalone `N` segment becomes `1`.

  `XXXXX` and `N` are the only placeholder tokens. A `title_prefix` that doesn't classify after substitution is a malformed schema (exit 1).
- **Section map:** contains
  - the **title** (with its source line on the YAML path),
  - metadata (key to value, keyed by the schema's display key),
  - sections (schema `id` to a list of entries, each with optional `name` (named sections only) and `content`),
  - which input it came from: a YAML draft, with line numbers per key, or a markdown body, with line numbers per heading.

  A markdown body has no title of its own, so its title is supplied by the caller (Component 4).
- **Finding:** severity (`error` | `warning`), a location (`section:<id>`, `metadata:<Key>`, `title`, or `draft` for draft-level problems such as YAML syntax errors and unknown top-level keys), the source line when known, and a message.
- **Result:** the schema type, the schema origin (`built-in` or `project override: <path>`) and the findings. **Loader findings** (Component 3) are returned as findings in the Result, with the same `schema:` line, never as a bare Go error. A loader error means no section map was built, so the Result holds only loader findings.
- **Result formatting** (plain text, owned here so it can be tested without the CLI):
  - a first line `schema: <type> (<origin>)`,
  - then one line per finding: `<severity> <location>[ line N]: <message>`,
  - errors before warnings, each group in source order.

**Behavior:** the check's rules, applied only to the section map:

| Severity | Condition |
|---|---|
| error | required section absent |
| error | required section present but content empty or whitespace-only |
| error | non-repeatable section has more than one entry |
| error | named numbered section entry has an empty name |
| error | a schema metadata key is absent or has an empty value (every schema metadata key is required) |
| error | title missing |
| error | title has no recognizable leading identifier, including an unreplaced `XXXXX` placeholder (the classifier's bool is false) |
| error | title kind ≠ schema's expected kind |
| error | title or a metadata value spans more than one line |
| error | section content contains a `## ` line outside a fenced code block (it would become a new top-level section when rendered; the fix says to use `###`) |
| error | metadata cross-check fails (see below) |
| warning | unknown `##` heading (markdown path only; on the YAML path unknown keys are errors, see Component 3) |
| warning | sections out of schema order (markdown only; YAML order is irrelevant because rendering imposes schema order) |
| warning | numbered headings not consecutive from 1 (markdown only) |
| warning | unknown metadata key (markdown path only) |
| warning | optional section present but empty |

The `## ` rule applies to both inputs. On the markdown path such a line is by definition a heading, so it can only be raised from YAML content in practice.

**Metadata cross-checks** are keyed by metadata key name rather than by schema type, so project overrides that keep these key names inherit them:
- `Plan Number`: must name the same plan number as the title, in the form `PLAN-XXXXX`.
- `Task Number`: must name the same task number as the title, in the form `TASK-XXXXX`.
- `Source Issue`, `Architecture Plan`, `Implementation Plan`: must begin with an issue reference `#<n>`. Trailing text is allowed (`#56 ([PLAN-00041-ARCH])`). If the trailing text contains a plan identifier, its plan number must match the title's.

**Actionable messages (user requirement):** every finding message states three things:
1. what is wrong, quoting the offending value or heading,
2. what the schema expects,
3. the concrete fix, phrased for the input path the content came from. A YAML-draft finding names the YAML key to add or change, and a markdown finding names the exact heading line.

Examples of the required specificity (the exact wording is the implementer's):
- `error section:verification: required section "Verification" is missing. Add a "verification: |" block under "sections:" in the draft.`
- `error metadata:Plan Number line 3: value "PLAN-00113" does not match the title's plan number PLAN-00112. Set "Plan Number" to "PLAN-00112", or fix the title.`
- `error title line 1: "[PLAN-00112-ARCH] X" is an arch title, but schema impl-phase expects a phase title like "[PLAN-XXXXX-N] <Phase Title>".`

A message never just says "invalid", never names only an internal id without the heading, and never omits the fix.

**TDD Entry Point:** a test in `template/` using the built-in `impl-phase` schema. The section map has a phase title (`[PLAN-00112-1] X`), matching metadata, and every required section except `verification`. It returns exactly one error, located at `section:verification`, whose message contains the heading `Verification` and a fix instruction. The same map with `verification` restored returns zero findings.

Further tests in the same component:
- The four built-in schemas derive arch, impl, phase and task respectively.
- An arch-plan section map titled `[PLAN-XXXXX-ARCH] X` fails at `title` (no recognizable identifier), not passing via the `Arch` zero value.
- An override missing `named` or `title_prefix` is a malformed-schema error.
- A Result with that one error formats as `schema: impl-phase (built-in)` followed by a line beginning `error section:verification`.
- One test per error class in the table above, asserting that the message quotes the offending value and contains a fix.

## 3. YAML draft: loader, renderer, skeleton, render rewiring

**Files:**
- `tool/internal/template/render.go` and a new draft file in `tool/internal/template/`, using `gopkg.in/yaml.v3` (already a dependency)
- `tool/internal/cli/handlers.go` (`runTemplateRender`) and `tool/internal/cli/root.go` (render flags)
- `tool/internal/cli/template_render_test.go`
- `tool/internal/template/render_test.go`: its 8 tests on the old `Render(schemaYAML, meta, sections)` / markdown `Skeleton` API move to the section-map API. `TestImplPlanNumberedSections`'s `## 3. Verify Step` expectation becomes `## 3. Verification Steps` (`named: false`).

**Specifications:** the draft format:
```yaml
title: "[PLAN-00112] Schema-checked plan issue creation"
metadata:
  Objective: "…"
  Architecture Plan: "#131 ([PLAN-00112-ARCH])"
sections:
  scope: |
    …markdown…
  component:            # named + numbered: list of {name, content}
    - name: "Title identifier kinds"
      content: |
        …
  verification_steps:   # numbered, not named: list of content blocks
    - |
      …
```
- A non-repeatable section is a block scalar.
- A repeatable section that isn't named is a list of block scalars.
- A named section is a list of `{name, content}`.
- Draft metadata order doesn't matter. Rendering always uses schema order.
- `Render` takes the section map, runs the Component 2 check first, and refuses to render on any error. The required and non-repeatable checks that live in `Render` today are deleted, not duplicated.
- Numbered sections keep one running ordinal across all numbered sections, as today.
- `Skeleton` emits an empty YAML draft. It contains:
  - a **double-quoted** `title:` placeholder from `title_prefix`,
  - every metadata key with `""`,
  - every section key with an empty block scalar, or a one-item list for repeatable sections,
  - each section's `guidance` as a YAML comment above its key.
- **Render rewiring in this component:** `runTemplateRender` switches to the new `Render`/`Skeleton`. The `--sections` and `--meta` flags and their JSON decoding are removed. The existing `template_render_test.go` cases that assert markdown headings in `--skeleton` output (`Problem Statement`, `OVERRIDE-MARKER-SECTION`) are updated to assert the YAML skeleton. (`overrideSchemaFixture` was already updated in Component 2.) `--draft` on render, `validate`, and help-text generation stay in Component 5.

**Behavior:**
- **Node-based loading:** the loader decodes into a `yaml.Node` tree, not a plain struct, so that nothing yaml.v3 accepts silently is lost. Node decoding gives up yaml.v3's built-in duplicate-key error, so the walker has to restore it. On the YAML path these are loader findings (exit 3), each with the line and the fix:
  - **Repeated key** in any mapping: top level, `metadata:`, `sections:`, or a named entry. Example: a model writing `component:` once per component instead of one list. The message names both lines and, for a repeatable section, says to make it one list.
  - **A comment attached to any scalar value**: title, metadata value, section content written as a plain scalar, or a named entry's `name`. An unquoted ` #` turned the rest of the value into a YAML comment. For example, `Implementation Plan: #132 (…)` loads as an empty value, `objective: Deliver X for #112` loads as `Deliver X for`, and `name: Fix #112 handling` loads as `Fix`. The error reports the cut, not the truncated or empty value: `line 3: the value of "Implementation Plan" was cut at "#". Wrap the whole value in double quotes.` For section content, the fix is to use a `|` block.
  - **Section content that isn't a literal `|` block scalar**, such as a plain scalar, a quoted scalar, or `>` folded (which silently reflows markdown lists). Fix: use `|`. Named-entry `content` follows the same rule.
  - **Any unknown key** at the top level, under `metadata:`, under `sections:`, or in a named entry. A content line that dedents to key level and contains `: ` produces exactly this, as a sibling key or a top-level key. The message names the key, the line, and says that a block-scalar line was probably under-indented. This is a deliberate difference from the markdown path, where an unknown heading is only a warning, because on the YAML path an unknown key almost always means lost content.
  - **A wrong node type**, such as a string where a list is expected.
- **Parser errors:** yaml.v3 errors are translated into findings at location `draft` with a fix, never passed through as raw parser text alone.
  - The line comes from yaml.v3's message when present. Otherwise it is the nearest line the loader can locate. Otherwise the message says `line unknown`.
  - Before mapping yaml.v3's message, the loader inspects the raw draft lines, because yaml.v3 gives the same message (`did not find expected key`, often with no line or the wrong line) for different causes:
    - **Unquoted title starting with `[`** (raw line matches `^title:\s*\[`): quote the title.
    - **A double-quoted value with an unescaped inner `"`** (a raw `key: "…"…"…"` line): name that line, and say to escape it as `\"` or switch to single quotes.
  - yaml.v3's `found unknown escape character` maps to: escape the backslash as `\\`, or use single quotes. The message names the line.
  - Any unmapped parser message falls back to: `the draft is not valid YAML (<parser text>). Check: section content uses "|" block scalars indented consistently; the title and metadata values are double-quoted with inner " and \ escaped; no tabs.`
- A draft never passes after being silently truncated.
- **Round-trip guarantee:** render a valid draft to markdown, parse it back with Component 4, and the result is an equivalent section map. Equivalence ignores trailing whitespace and newlines at the end of a section.

**TDD Entry Point:** a test that loads a minimal valid `impl-phase` draft, renders it, and asserts that the headings and metadata lines come out in schema order. Further cases, each asserting exactly one error with a line and a fix:
- `Implementation Plan: #132 ([PLAN-00112])` under `metadata:` (comment cut),
- `objective: Fix #112 thing` under `sections:` (comment cut on section content),
- `name: Fix #112 handling` in a component entry (impl-plan draft),
- two `component:` keys (repeated key; impl-plan draft),
- `scope: >` with a bullet list (non-`|` section content),
- a dedented `  Note: x` line inside `scope:` (unknown key under `sections:`),
- an unquoted `title: [PLAN-00112-1] X`,
- `title: "[PLAN-00112-1] Add "validate" command"` (inner quote: escape fix, not "quote the title"),
- `Status: "Match \d{5}"` (unknown escape),
- a `## Sub` line inside `scope: |` outside a fence (the Component 2 rule).

## 4. Markdown body parser

**Files:** a new parser file in `tool/internal/template/`.

**Specifications:** parses a markdown body into the section map, for a given schema and a **caller-supplied title** (from `--title`, or from the live issue on edit). The title is stored in the map with no source line.
- **Metadata:** `**Key:** value` lines before the first `##` heading.
- **Sections:** each `## ` heading starts a section, which runs until the next `## ` heading.
- **Numbered headings** (`## N. Text`):
  1. If Text equals a non-named numbered section's literal heading, the block maps to that section.
  2. Otherwise, if the schema has exactly one named numbered section, the block maps to it with name = Text.
  3. Otherwise, the heading is unknown.
- **Literal headings** (anything else): matched case-sensitively against schema headings. A non-matching heading is unknown.

**Behavior:**
- `##` lines inside fenced code blocks (```` ``` ```` or `~~~`) are content, not headings. `###` and deeper headings are always content.
- Non-metadata text before the first `##`, such as a revision blockquote, is ignored without a finding. Markdown bodies are checked, never rewritten, so that text survives.
- Each section keeps its source line number for findings.

**TDD Entry Point:** a test that parses an `impl-phase` markdown body with title `[PLAN-00112-1] X`, matching metadata, and no `## Verification`. It yields the same single error as Component 2's entry test, which proves the YAML and markdown paths give the same verdict. Further cases:
- a `## Foo` line inside a fenced block in `## Scope` is content,
- a round-trip test for each of the four built-in schemas: a valid draft (one containing a `###` sub-heading and a fenced `## ` line in section content) is rendered by Component 3, parsed back, and yields an equivalent section map with zero findings.

## 5. CLI: exit code 3, `template validate`, `template render --draft`

**Files:**
- `tool/internal/clierr/clierr.go` (new constructor; its doc comment's code list gains `3`), `tool/internal/cli/errors.go`
- `tool/internal/cli/root.go` (`newTemplateCmd`)
- `tool/internal/cli/handlers.go` (`resolveSchema`)

**Specifications:**
- **Exit code 3:** a new `clierr` constructor for conformance failures with `Code: 3`. `RunWithStreams` already maps a `*CLIError` to its Code, so code 3 needs no further plumbing there.
- **Schema origin:** `resolveSchema` returns the override's path, not just the string "project override", so the origin can be reported.
- **`marvin template validate <type> (--draft <file.yml> | --body-file <file.md>) [--title <t>] [--json]`.** It needs no config and makes no GitHub call.
  - With `--draft`, the title comes from the draft.
  - With `--body-file`, `--title` supplies it. If no title is given, a `title` error is reported.
  - The formatted result (Component 2) goes to stdout, because it's the command's data.
  - `--json` renders the same result as an object (`schema`, `origin`, `findings[]`). It's for non-agent callers only.
- **`marvin template render <type> --draft <file.yml>`:**
  - prints the rendered markdown body to stdout and warnings to stderr,
  - exits 3 with the findings on stderr on any error.
- **Help text:** the `Use` strings for `render` and `validate` list the types from the embedded schema set, so `quick-task` appears and can't drift out again.

**Behavior:**
- Exit 3 whenever the Result holds at least one error finding, including loader findings from a draft that won't load (Component 3).
- Warnings alone exit 0.
- Unknown schema type, unreadable input file, malformed schema or override: exit 1, never a pass and never a silent fallback to the built-in.
- `--draft` and `--body-file` are mutually exclusive. Passing both, or neither (without `--skeleton` on render), is a usage error, exit 1.
- Schema resolution is today's precedence: project override by CWD walk, then built-in.

**TDD Entry Point:** a CLI test (`cli_test`) running `template validate impl-phase --draft <missing-verification.yml>` asserts:
- exit code 3,
- first line `schema: impl-phase (built-in)`,
- a stdout line beginning `error section:verification`.

Further tests:
- A project override `impl-phase.yml` in a temp dir that drops `verification`: the same draft exits 0, and the first line reports `project override: <path>`.
- `template render --help` lists `quick-task`.

## 6. Checked `issue create` and new `issue edit`

**Files:**
- `tool/internal/gh/client.go`: new `IssueEdit`
- `tool/internal/issue/issue.go`: new `Edit`
- `tool/internal/cli/handlers_integrations.go`: create gains the flags; new edit command

**Specifications:**
- `issue create` gains `--template <type>` and `--draft <file.yml>`.
  - With `--template` and `--draft`, the title comes from the draft. `--title` is optional, and if it's given it must equal the draft's title, otherwise usage error, exit 1.
  - With `--template` and `--body-file`, `--title` is required.
  - With `--template` and the inline `--body` flag: usage error, exit 1. The message says to use `--draft` or `--body-file`.
  - Without `--template`, behavior is exactly as today, and `--draft` is rejected (exit 1).
- `marvin issue edit <n> --template <type> (--draft <file.yml> | --body-file <file.md>)`:
  - updates the issue's body. With a draft, it also updates the title.
  - `--template` is required.
  - There is no inline `--body` flag.
- `gh.Client.IssueEdit(ctx, repo string, number int, title, body string)` runs `gh issue edit <n> --repo R --body B`, plus `--title T` when the title is non-empty.

**Behavior:**
- **Order of operations:**
  1. load config (exit 2 if missing),
  2. resolve the schema,
  3. load the input,
  4. run the check,
  5. only then make the mutating `gh` call.
- On any error finding, no mutating `gh` call is made and the command exits 3 with the findings on stderr. Stdout stays empty, so a skill reading number/URL can't mistake a rejection for success.
- **`issue edit --body-file` title read:** the current title comes from `IssueRef(ctx, cfg.Repo, n)`, which passes `--repo`. `IssueJSON` is not used, because it ignores the configured repo. This read is the only call before the check. The no-call guarantee covers mutating calls.
- Warnings and the `schema:` line go to stderr. Stdout keeps today's contract: number then URL for create, and nothing for edit.
- A markdown body is sent unchanged. A draft is sent as rendered markdown.

**TDD Entry Point:** CLI tests with `exectest.FakeRunner` and `withConfigFixture`:
1. `issue create --template impl-phase --draft <missing-verification.yml> --label x` exits 3, and `len(fake.Calls) == 0`.
2. The same command with a conforming draft makes exactly one `gh issue create` call, whose `--body` is the rendered markdown and whose `--title` is the draft's title.
3. `issue edit 7 --template impl-phase --draft <ok.yml>` makes exactly one call: `gh issue edit 7 --repo <cfg repo> --body <rendered> --title <draft title>`.
4. `issue edit 7 --template impl-phase --body-file <ok.md>` makes:
   - a read call `gh issue view 7 --repo <cfg repo> …`,
   - then exactly one `gh issue edit 7 --repo <cfg repo> --body <file contents unchanged>`, with no `--title`.
5. `issue edit 7 --template impl-phase --draft <missing-verification.yml>` exits 3 with zero calls.
6. `issue create --template impl-phase --body "…"` exits 1 with zero calls.

## 7. Skill adoption

**Files:** `skills/arch-plan/SKILL.md`, `skills/impl-plan/SKILL.md`, `skills/phase-split/SKILL.md`, `skills/quick-task/SKILL.md`

**Specifications:**
- **Frontmatter:** each skill's `allowed-tools` adds `Write`.
- **Drafting step:**
  - Replace the "fill in `--skeleton` markdown" instruction with: render `marvin template render <type> --skeleton` (now a YAML draft), then `Write` the filled draft to a scratch `.yml` file.
  - State the format rules:
    - section content always as `|` block scalars, never `>` or inline,
    - **the title and every metadata value always double-quoted, with inner `"` written as `\"` and `\` as `\\`**,
    - each key written once (repeatable sections are one list),
    - no tabs,
    - no `## ` lines in section content (use `###`).
  - Remove the "has no `Write` tool" text and the body heredoc.
- **Review step:** show the user the rendered markdown (`marvin template render <type> --draft <file>`) in the reply text, not the raw YAML. A render that exits 3 is fixed before review.
- **Create step:** `marvin issue create --template <type> --draft <file> --label …`. quick-task uses `--template quick-task`, and phase-split uses `--template impl-phase` per phase.
- **On exit 3:** read each finding, fix the draft, and retry. After 3 failed attempts, stop and show the findings to the user. Exit 1 or 2 is surfaced, never retried.
- **phase-split:**
  - **Replace the "compose immediately before creating" rule** (today's SKILL.md ~line 92, "compose the title and body together, as one atomic unit, immediately before creating that issue"). The new rule: each phase's title and body are written together in one draft file; write and validate every draft, then create them in order. The rule's real concern, title/body pairing drift, is covered because a draft keeps both in one file.
  - **Validate everything before creating anything:** phase-split writes every phase's draft first, then runs `marvin template validate impl-phase --draft <file>` on each, applying the fix-and-retry rule. It makes the first `issue create` only when every draft passes. A conformance failure therefore can't leave some phase issues created and others not, which its re-run guard would then refuse.
  - **If a create still fails** (gh or network error), stop and report which phase issues were created, by number, so the user can finish or delete them.
  - **Step 3b fix-up:** the raw `gh issue edit … --body-file` becomes `marvin issue edit <n> --template impl-phase --draft <corrected-draft>`.
- Heredocs unrelated to issue bodies (`gh issue comment`, `gh pr comment`, `gh pr edit`) are left alone.

**Behavior:** no step in these four skills creates or edits a plan or Task issue body without `--template`.

**TDD Entry Point:** none. This is skill prose. **This component is its own phase** (Component 8 may join it, since docs add no TDD entry point either), with **TDD Entry Point: None**, so review-phase's structural pre-check (step 2b) runs. That step is skipped for any phase with a real TDD entry point. The pre-check greps:
- `allowed-tools` contains `Write` in all four,
- every `marvin issue create` line in all four contains `--template`,
- phase-split contains `marvin template validate impl-phase`,
- no `gh issue edit … --body` remains in phase-split,
- no "has no `Write` tool" text remains,
- no "immediately before creating that issue" text remains in phase-split.

## 8. Documentation

**Files:** `CLAUDE.md`, `skills/SHARED/CONFIG.md`, `skills/start-impl/SKILL.md`, `skills/review-impl/SKILL.md`, `skills/phase-split/SKILL.md` (kind list only)

**Specifications:**
- **CLAUDE.md:**
  - the exit-code contract becomes `0` success, `1` operational error, `2` config missing or malformed, `3` draft non-conforming (fix and retry). The repository-structure line `clierr/ ← Exit-code constants (0 / 1 / 2)` becomes `(0 / 1 / 2 / 3)`.
  - the `template/` line in the repository structure mentions validation,
  - the per-project config paragraph notes that validation follows the same precedence.
- **`skills/SHARED/CONFIG.md` "Plan Template Resolution":**
  - covers `render`, `validate`, `issue create --template` and `issue edit --template`,
  - documents the YAML draft as the skills' authoring format,
  - lists the required schema fields (`title_prefix` on every schema, `named` on every numbered section), with a short explanation of `named`,
  - adds a **migration note for existing overrides**: an override copied from an older built-in fails loudly (exit 1, naming the field) until it adds `named` and `title_prefix`, and the note says what values the built-ins use.
- **`issue tree` kind lists:** the four places that say `issue tree` emits "`kind` one of `arch`, `impl`, `phase`" add `task`:
  - `skills/start-impl/SKILL.md` (~line 31),
  - `skills/review-impl/SKILL.md` (~39),
  - `skills/phase-split/SKILL.md` (~54),
  - `skills/SHARED/CONFIG.md` (~123).

  No logic changes, since those skills filter on `phase`.

**TDD Entry Point:** none (documentation).

## 9. Verification Steps

```bash
# Unit and CLI tests
cd tool && go test ./...

# Title kinds
marvin parse title "[TASK-00091] Fix X"                          # found: true / kind: task / task: 91 / task_number: TASK-00091
marvin parse title "[TASK-00140] Fix regression from [PLAN-00112-3]"  # kind: task
marvin parse title "[PLAN-00112-ARCH] X"                         # … kind: arch (existing keys unchanged)
marvin parse title "Notes on [PLAN-00042-1]"                     # found: true + plan fields as today, no kind: line

# Skeleton is a YAML draft with a quoted title; help lists all four types
marvin template render impl-phase --skeleton    # title: "[PLAN-XXXXX-N] <Phase Title>", metadata/sections keys, guidance comments
marvin template render --help | grep quick-task

# Validate: conforming, non-conforming, silent-truncation cases
marvin template validate impl-phase --draft ok.yml; echo $?              # schema: impl-phase (built-in) / exit 0
marvin template validate impl-phase --draft no-verification.yml; echo $? # error section:verification … / exit 3
marvin template validate impl-phase --draft hash-in-value.yml; echo $?   # error metadata:Implementation Plan line N: … cut at "#" … double quotes / exit 3
marvin template validate impl-phase --draft dedented-line.yml; echo $?   # error draft line N: unknown key "Note" … under-indented / exit 3
marvin template validate impl-plan --draft dup-component.yml; echo $?    # error draft line N: key "component" repeated (first at line M) … one list / exit 3
marvin template validate impl-phase --draft folded-scope.yml; echo $?    # error section:scope line N: … use "|" / exit 3

# Existing issue bodies check out (markdown path)
gh issue view 131 --json body -q .body > /tmp/131.md
marvin template validate arch-plan --body-file /tmp/131.md --title "[PLAN-00112-ARCH] Schema-checked plan issue creation and edits"
# exit 0; any warnings are listed and explained

# Checked create refuses without touching GitHub
marvin issue create --template impl-phase --draft no-verification.yml --label plan:phase; echo $?  # exit 3, no issue created
marvin issue create --template impl-phase --title "[PLAN-00112-1] X" --body "x"; echo $?          # exit 1, use --draft or --body-file

# Skill-prose structural check
grep -n "allowed-tools" skills/{arch-plan,impl-plan,phase-split,quick-task}/SKILL.md   # all include Write
grep -n "marvin issue create" skills/{arch-plan,impl-plan,phase-split,quick-task}/SKILL.md  # all include --template
grep -n "marvin template validate impl-phase" skills/phase-split/SKILL.md   # present
grep -n "gh issue edit" skills/phase-split/SKILL.md   # no matches
```

## Design Notes

- **Why the YAML loader works on `yaml.Node`:** yaml.v3 accepts several malformed drafts with no error:
  - an unquoted `#` cuts a value into a comment,
  - an under-indented content line becomes a sibling key and truncates its section,
  - an unknown top-level key is dropped by struct decoding,
  - a `>` block reflows markdown.

  Only a node-level walk sees comments, scalar styles and every key. The cost is that Node decoding drops yaml.v3's duplicate-key error, so the walker checks for repeated keys itself. That's what makes "a draft never passes after being silently truncated" true rather than aspirational. Unknown keys are errors on the YAML path but only warnings on the markdown path, because on the YAML path they almost always mean lost content.
- **Why a `named` schema field, and why it's required:** whether a numbered section's heading comes from content is a structural fact. It belongs in the schema explicitly, where override authors can see it, not inferred from bracket syntax. Making it required means an override written for today's behavior fails loudly with the fix instead of silently rendering `## 1. <Component or Layer Name>`. It also fixes a latent bug: today `Render` takes every numbered block's heading from its first line, `verification_steps` included.
- **Why metadata cross-checks are keyed by key name:** keying by schema type would make project overrides silently lose them. Keying by `Plan Number` / `Task Number` / reference keys means any schema that uses those keys gets the checks, and a schema that renames them opts out visibly.
- **Why exit 3 covers YAML parse errors:** a broken draft is the agent's own fixable mistake, exactly like a missing section. Exit 1 stays reserved for "marvin or gh couldn't do its job", which an agent shouldn't retry blindly.
- **Why markdown bodies are checked, not re-rendered:** re-rendering would drop preamble text such as revision notes, and could normalize content the author intended. Only YAML drafts are rendered.
- **Why reuse `names.Kind`:** it already has exactly the four kinds. A third enum next to `parse.Kind` and `names.Kind` would be one more thing to keep in sync.
- **Stdout contracts don't move:** `issue create` keeps number/URL on stdout and puts findings on stderr. `template validate` puts findings on stdout because they're its data. This follows the existing stdout = data, stderr = diagnostics rule.
- **Version coupling:** skills and marvin ship together. A stale binary fails the first `--template` or `--draft` call with an unknown-flag error, exit 1. That's loud, and acceptable per the arch plan.
- **Dependency order for phase-split:** 1 → 2 → {3, 4} → 5 → 6 → 7 (+8).
  - 2 needs 1's classifier for the title-kind rule.
  - 3 and 4 both need 2's section map and check. They're independent of each other, but 4's round-trip tests use 3's renderer, so 3 goes first if they're separate phases.
  - 5 needs 3 and 4 for its two input paths.
  - 6 needs 5's exit code and schema origin.
  - 7 needs 5 and 6 merged, and must be its own phase with TDD Entry Point: None (see Component 7). 8 can join it.
  - 1 and 7 are unrelated and stay in separate phases.

## Success Criteria

- [ ] `marvin parse title` prints a `kind:` line classified from the leading bracket only. `found:` and the plan fields stay driven by `PlanIdent` and are unchanged for every non-task title, including `Notes on [PLAN-00042-1]`. Task titles print `task_number: TASK-XXXXX`. `issue tree` output is byte-identical for every non-task title.
- [ ] A title with no recognizable leading identifier, including an unreplaced `[PLAN-XXXXX-ARCH]` placeholder, is a `title` error for every schema, including arch-plan.
- [ ] The four built-in schemas derive the expected title kinds arch, impl, phase and task.
- [ ] For each of the four built-in schemas, a valid YAML draft renders, parses back to an equivalent section map, and validates with zero errors and zero warnings.
- [ ] The same structural defect gives the same finding whether it's supplied as a YAML draft or as a markdown body. The documented exception: unknown keys are errors in YAML and warnings in markdown.
- [ ] Every error finding names its location, quotes the offending value or heading, and states the concrete fix for the input path it came from. A test asserts this for each error class in Component 2's table.
- [ ] `Plan Number`, `Task Number` and reference metadata mismatches against the title are errors.
- [ ] Each of these exits 3 with a message giving the line (or `line unknown`) and the fix, and is never partially accepted:
  - an unquoted `#` in a metadata value, in plain-scalar section content, or in a named entry's `name`,
  - a repeated key in any mapping,
  - section content that isn't a `|` block (including `>`),
  - a dedented content line,
  - an unknown top-level key,
  - an unquoted `[`-leading title,
  - an unescaped inner `"` in a quoted value (fix names escaping, not quoting),
  - an unknown `\` escape,
  - a `## ` line in section content,
  - bad indentation,
  - a wrong node type.

  Loader findings use location `draft` (or the specific key when known) and come back in the Result with the `schema:` line.
- [ ] A schema or override missing `title_prefix`, or missing `named` on a numbered section, exits 1 with a message naming the origin, the field and the fix, for every command including `render --skeleton`.
- [ ] `issue create --template` and `issue edit --template` make zero mutating `gh` calls on any error and exit 3, as asserted via `exectest`.
- [ ] A conforming `issue edit` makes exactly one `gh issue edit` call with `--repo <cfg repo>`: rendered body plus title for `--draft`, the unchanged file body without `--title` for `--body-file`. Its title read also uses `--repo`.
- [ ] `--template` together with inline `--body` is a usage error (exit 1) with no `gh` call.
- [ ] Warnings never change the exit code. Unknown type, unreadable input and malformed schema or override exit 1. Missing config exits 2 where config is needed.
- [ ] Every check reports its schema origin, including the override path.
- [ ] `render --sections` and `--meta` are gone. `--skeleton` emits a YAML draft with a quoted title. Help text lists all four types from the embedded set.
- [ ] All four drafting skills have `Write`, draft YAML with a quoted, escaped title and metadata, and pass `--template` on every create. phase-split validates every phase draft before its first create, no longer says "immediately before creating that issue", and its fix-up uses `marvin issue edit --template`.
- [ ] CLAUDE.md (contract line and the `clierr/` structure line) and the `clierr.go` comment document exit code 3. The four `issue tree` kind lists include `task`. `skills/SHARED/CONFIG.md` documents the draft format, the required schema fields, the override migration note and the validation precedence.
- [ ] `go test ./...` passes.
