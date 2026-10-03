# Phase 2: Section map and conformance check

**Status**: Completed
**Phase issue**: [#134](https://github.com/threehillpath/marvin-sdd/issues/134)
**Pull request**: [#142](https://github.com/threehillpath/marvin-sdd/pull/142)
**Implementation plan**: [#132](https://github.com/threehillpath/marvin-sdd/issues/132)

## Objective

Phase 2 of PLAN-00112: build the section map and the single conformance check, plus the findings, Result and plain-text formatter. Every later entry point (render, validate, checked create and edit) calls this one check.

## Scope

**In scope (impl plan Component 2):**
- **Schema model:** add `type`, `title_prefix`, and a section field `named`.
  - `title_prefix` is required on every schema.
  - `named` (true or false) is required on every numbered section.
  - A missing field is a malformed-schema error (exit 1). The message names the origin, the field and the fix.
  - All four built-in schemas gain `named`: impl-plan `component: true` and `verification_steps: false`.
- **Expected title kind:** classify `title_prefix` with Phase 1's classifier after substituting `XXXXX` → `00000` and a standalone `N` → `1`. A prefix that doesn't classify is a malformed schema.
- **Section map:** title (with its source line on the YAML path), metadata, and sections. Sections map schema `id` to a list of entries, each with an optional `name` and `content`. The map also records its input origin and line numbers.
- **Finding:** severity, location (`section:<id>`, `metadata:<Key>`, `title` or `draft`), line, and message.
- **Result:** schema type, origin (`built-in` or `project override: <path>`) and findings. Loader findings, which Phase 3 adds, live in the Result too.
- **Formatter:**
  - a first line `schema: <type> (<origin>)`,
  - then `<severity> <location>[ line N]: <message>` per finding,
  - errors before warnings, each group in source order.
- **Rules:** the full error/warning table in impl plan §2, including:
  - a title with no recognizable identifier,
  - a multi-line title or metadata value,
  - a `## ` line outside a fence in section content,
  - metadata cross-checks keyed by key name (`Plan Number`, `Task Number`, `Source Issue` / `Architecture Plan` / `Implementation Plan`).
- **Actionable messages:** each message says what is wrong (quoted), what the schema expects, and the concrete fix for the input path.
- `overrideSchemaFixture` in `cli/template_render_test.go` gains `title_prefix` and `named`, so `go test ./...` stays green.

**Out of scope:** YAML loading and the `Render` / `Skeleton` rewrite (Phase 3), markdown parsing (Phase 4), CLI commands (Phase 5).

## TDD Entry Point

A test in `tool/internal/template/` with the built-in `impl-phase` schema. The section map has title `[PLAN-00112-1] X`, matching metadata, and every required section except `verification`. It returns exactly one error, at `section:verification`, whose message contains `Verification` and a fix. With `verification` restored, there are zero findings.

## Components

- `tool/internal/template/`: new section map, check, finding/Result and formatter files. The schema struct in `render.go` is widened.
- `tool/internal/template/schemas/*.yml`: add `named`.
- `tool/internal/cli/template_render_test.go`: `overrideSchemaFixture` update only.

## Verification

```bash
cd tool && go test ./internal/template/... ./internal/cli/...
cd tool && go test ./...
```

## Success Criteria

- [ ] The entry-point test passes: exactly one `section:verification` error, then zero findings once `verification` is restored.
- [ ] The four built-in schemas derive arch, impl, phase and task.
- [ ] An override missing `named` or `title_prefix` is a malformed-schema error naming the origin, the field and the fix.
- [ ] An arch-plan map titled `[PLAN-XXXXX-ARCH] X` fails at `title`. It does not pass via the `Arch` zero value.
- [ ] One test per error class in the §2 table, each asserting that the message quotes the offending value and contains a fix.
- [ ] `Plan Number`, `Task Number` and reference mismatches against the title are errors.
- [ ] A Result formats as `schema: impl-phase (built-in)` followed by `error section:verification …`, errors before warnings.
- [ ] `go test ./...` passes.

---

## Implementation

Phase 2 adds the section map and the conformance check for plan issue templates.

**Data model**

- `SectionMap`, `Entry`, `Field` and `Heading` represent a parsed draft.
- `template.Check` validates a map against a schema and returns a `Result` of `Finding`s.
- `Result.Format` prints findings as plain text: errors before warnings, ordered by line where known, then in schema order.
- `Check` implements the error and warning rules in §2 of the impl plan, plus metadata cross-checks against the title (`Plan Number`, `Task Number`, references).

**Fix text**

Every finding's fix depends on two things:
- the input path: YAML draft or markdown body;
- the section's shape: a `|` block, a list of `- |` blocks, or named entries with `name:` and `content: |`.

**Schema model**

- The schema model now requires `type`, `title_prefix` and `named` (on numbered sections).
- `template.LoadSchema` rejects a schema that lacks any of these, with an error that names the origin, the field and the fix.
- `LoadSchema` derives the expected title kind from `title_prefix` using Phase 1's classifier. So `[PLAN-XXXXX-ARCH] X` fails at the title check instead of passing via the Arch zero value.
- `Render` and `Skeleton` now load through `LoadSchema`, so a malformed override also fails `render --skeleton` with exit 1.
- `Check` reports an unset `Source` and any `Schema` that was not built by `LoadSchema`.

**Markdown-path checks**

These use `FindH2Lines`, a shared CommonMark-aware scanner that Phase 4's parser will reuse. It tracks fence character and length, accepts headings indented by up to 3 spaces, and returns 1-based lines and heading text without the `##` marker. It supports:
- out-of-order detection over every entry;
- unknown-heading warnings that list the expected headings;
- an error for a code fence left unclosed in section content. Content is data and must not alter the document's structure.

**Review**

Three review passes produced 16 corrections, mostly to the accuracy of fix text and the strength of tests. Three nits and several remaining structure-breaking content cases are deferred to Phases 3 and 4.

### Test Plan / Verification

- [x] Entry point: an impl-phase map missing `verification` gives exactly one `section:verification` error, and zero findings once it is restored
- [x] The built-in schemas derive arch, impl, phase and task
- [x] An override missing `named`, `title_prefix` or `type` is a malformed-schema error that names the origin, the field and the fix
- [x] `[PLAN-XXXXX-ARCH] X` fails at `title`, not via the Arch zero value
- [x] One test per error class, plus the cross-checks, formatter ordering, markdown-path warnings and fence rules
- [x] `go test ./...`, `go vet ./...` and `gofmt -l` are clean

### Decisions

- **An unclosed code fence in section content is now a `section:<id>` error. It was previously accepted.**
  - *Why:* Section content is pure data and must never alter the document's structure. An unclosed fence would render every later section as code and hide those sections from the parser, so the checker rejects it instead of accepting it silently (round 2 N3).
- **The schema model now requires `type`, `title_prefix` and `named`, and `LoadSchema` derives the expected title kind from `title_prefix` using Phase 1's classifier.**
  - *Why:* This makes the conformance check schema-driven and avoids a false pass from the Arch zero value. A missing field fails with an error that names the origin, the field and the fix.
- **`Render` and `Skeleton` load schemas through `LoadSchema`, so a malformed schema also fails `render --skeleton` with exit 1.**
  - *Why:* Every consumer goes through one validated load path.
- **`Check` reports an unset `Source` (the `SourceUnknown` zero value) and any `Schema` that was not built by `LoadSchema`.**
  - *Why:* Before this change, zero values silently selected YAML fixes or the arch kind, which breaks the no-silent-failures rule.
- **Out-of-order detection flags sections that fall outside the longest in-order run, computed over every entry by line.**
  - *Why:* One displaced section produces one warning, and interleaved component entries are no longer missed.
- **`FindH2Lines` is exported. It applies CommonMark's fence and H2 rules and returns 1-based lines and heading text without the `##` marker, so Phase 4's parser can reuse it.**
  - *Why:* With one shared scanner, the checker and the parser cannot drift apart on what counts as a heading.

### Scope Changes

- **added:** `FindH2Lines` is exported as a reusable API for Phase 4's parser. Phase 4 needs the same CommonMark-correct detection.
- **added:** The checker now reports an unclosed code fence as an error. Raised in round-2 review (N3) and promoted to an error by the author.
- **added:** `impl-plan.yml` gains `named: true` on `component` and `named: false` on `verification_steps`. The schema's new required `named` field needs it. `Render` output is unchanged until Phase 3.

### Deferred / Watch Items

- **Three nits from the third review pass: a placeholder leak in `checkNumbering`, a weak markdown assertion in the B1 test, and a content-H2 error that quotes a rebuilt heading.** Tracked in #135.
- **Section content that could still break document structure: setext headings, unclosed HTML comments and blocks (including `<pre>`), lines that look like metadata, and content that only forms a heading after rendering.** Tracked in #135, #136.

### Corrections

- **The fix text for a missing or empty section (B1)**
  - *Problem:* It always said `<id>: |` and used the placeholder heading, which is the wrong node type for `component` and `verification_steps`.
  - *Fix:* The fix is now built from the section's shape, and removal is offered only for optional sections.
- **The missing-title fix on the markdown path (B2)**
  - *Problem:* It told the agent to set a `title:` key in a draft that does not exist.
  - *Fix:* On the markdown path the fix says to supply the title with `--title`.
- **The fix for a missing `title_prefix` (B3)**
  - *Problem:* It always suggested the impl-plan prefix, so copying it into another override type loaded silently with the wrong kind.
  - *Fix:* The fix quotes the built-in prefix for the schema's type, or lists all four forms.
- **The order check (N1)**
  - *Problem:* It compared only each section's first entry.
  - *Fix:* It now computes the longest in-order run over all entries.
- **Warning messages (N2)**
  - *Problem:* They forced a second lookup.
  - *Fix:* They list the expected headings and name the heading to move after or before.
- **Fence and H2 detection (N3)**
  - *Problem:* It ignored fence length and info strings, and missed headings indented by up to 3 spaces.
  - *Fix:* It now follows CommonMark rules, exported as `FindH2Lines`.
- **Test coverage for a malformed override on `--skeleton` (N4)**
  - *Problem:* Only `LoadSchema` was tested.
  - *Fix:* A CLI regression test was added.
- **Title tests (N5)**
  - *Problem:* They checked keywords only.
  - *Fix:* They now assert the quoted value, the fix and an example.
- **The `named` comment in `impl-plan.yml` (N6)**
  - *Problem:* It split the `numbered` comment and described rendering that did not exist yet.
  - *Fix:* It was moved and now notes that `named` takes effect in Phase 3.
- **Handling of an unset `Source` or a `Schema` not loaded through `LoadSchema` (N7)**
  - *Problem:* Zero values silently selected YAML or arch.
  - *Fix:* `SourceUnknown` and the `loaded` flag are now reported by `Check`.
- **A missing `type` in `LoadSchema` (N8)**
  - *Problem:* The header and messages lost the schema name.
  - *Fix:* It is now an error that names the origin, the field and the built-in types.
- **Test helpers (N9)**
  - *Problem:* They discarded load errors.
  - *Fix:* They now fail the test, `mustSchema` is removed, and the numbering test is cleaned up.
- **An empty named entry (B1, round 2)**
  - *Problem:* With both name and content empty, it still got the `- |` fix and the placeholder heading.
  - *Fix:* Every named section now takes the named branch, and headings come from `templateHeading`.
- **The missing-`type` fix (N1, round 2)**
  - *Problem:* It suggested `type: quick-task`, which leads back to the wrong title kind.
  - *Fix:* It now points to the override file's base name and lists the built-in types.
- **`FindH2Lines` output (N2, round 2)**
  - *Problem:* It returned zero-based lines and text that still carried the `##` marker.
  - *Fix:* It now returns 1-based lines and heading text without the marker.
- **An unclosed code fence (N3, round 2)**
  - *Problem:* It passed with zero findings.
  - *Fix:* It is now a `section:<id>` error that names the section and the fence and says how to close it.
