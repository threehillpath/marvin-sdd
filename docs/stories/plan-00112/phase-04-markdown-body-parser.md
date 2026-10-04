# Phase 4: Markdown body parser

**Status**: Completed
**Phase issue**: [#136](https://github.com/threehillpath/marvin-sdd/issues/136)
**Pull request**: [#144](https://github.com/threehillpath/marvin-sdd/pull/144)
**Implementation plan**: [#132](https://github.com/threehillpath/marvin-sdd/issues/132)

## Objective

Phase 4 of PLAN-00112: add the markdown body parser, which turns an existing issue body plus a caller-supplied title into the same section map. Round-trip tests prove the YAML path and the markdown path can't disagree.

## Scope

**In scope (impl plan Component 4):**
- **Input:** parses a markdown body for a given schema plus a caller-supplied title, stored with no source line.
- **Metadata:** `**Key:** value` lines before the first `##`.
- **Sections:** each `## ` heading starts a section, which runs until the next `## `.
- **Numbered `## N. Text`:**
  1. Text equals a non-named numbered section's literal heading → that section.
  2. Otherwise, if the schema has exactly one named numbered section → that section, with name = Text.
  3. Otherwise → unknown.
- **Literal headings:** matched case-sensitively. A non-match is unknown.
- `##` inside fenced blocks (```` ``` ```` or `~~~`) is content. `###` and deeper is always content.
- Non-metadata preamble before the first `##` (e.g. a revision blockquote) is ignored without a finding.
- Every section keeps its source line for findings.

**Out of scope:** CLI wiring (Phase 5). Rewriting markdown bodies: they are checked, never re-rendered.

## TDD Entry Point

Parse an `impl-phase` markdown body with title `[PLAN-00112-1] X`, matching metadata, and no `## Verification`. It yields the same single `section:verification` error as Phase 2's entry test.

## Components

- `tool/internal/template/`: a new markdown parser file and its tests.

## Verification

```bash
cd tool && go test ./internal/template/...
cd tool && go test ./...
```

## Success Criteria

- [ ] The entry-point test yields exactly the same single finding as Phase 2's YAML/section-map case.
- [ ] A `## Foo` line inside a fenced block in `## Scope` is content, not a section.
- [ ] `## N. Verification Steps` maps to `verification_steps`, and other `## N. <Name>` headings map to `component` with that name (impl-plan).
- [ ] Round-trip for each of the four built-in schemas: a valid draft is rendered (Phase 3) and parsed back. The draft contains a `###` sub-heading and a fenced `## ` line in content. Parsing back yields an equivalent section map with zero findings.
- [ ] Preamble text before the first `##` produces no finding.
- [ ] `go test ./...` passes.

---

## Implementation

Phase 4 adds the markdown-body path in `tool/internal/template`: `markdown.go` and `markdown_test.go`, plus a refactor in `check.go`.

`ParseMarkdown(sc, title, body)` turns a markdown issue body and a caller-supplied title into the same `SectionMap` the YAML path builds.

**Metadata.** Metadata comes only from `**Key:** value` lines above the first `## ` heading, and only where GitHub would show them as metadata:
- outside fences;
- first in the body, after a blank line, or directly below another accepted metadata line.

Indents under 4 columns are accepted. A repeated key keeps its first value and reports the repeat as an error. A blank line is one that contains only spaces and tabs.

**Sections.** Sections are split with the shared `FindH2Lines` scanner, so the parser and the checker agree on where sections begin and end. A numbered heading maps to a non-named numbered section whose literal text it matches, otherwise to the schema's only named numbered section, with the name taken from the heading text. For example, `## N. Verification Steps` → `verification_steps` and other `## N. <Name>` headings → `component`. Anything else is unknown. Content is stored with leading blank lines and trailing whitespace stripped, and CRLF is normalized to LF.

**Content guards.** Content under unknown `## ` headings is kept in `Heading.Content`, and the preamble is stored on the map. `checkContentStructure` now takes `(loc, what, content, line)`, so `Check` runs the content guards (raw HTML, fences, setext, `## `, lone CR) on known sections, unknown headings and the preamble. Preamble findings report "line N of the body".

**`CheckMarkdown`.** `CheckMarkdown(sc, origin, title, body)` runs `ParseMarkdown`, then `Check`. When `Check` finds no errors, it also runs the same goldmark+GFM `verifyBody` backstop that `Render` uses. A body that GitHub would structure differently from the parser is refused, including one with a level-1/2 heading or an HTML block in the preamble.

**Tests and wiring.** Round-trip tests (map → `Render` → `ParseMarkdown`, giving an equivalent map, zero findings and a clean `CheckMarkdown`) cover all four built-in schemas. The real issue bodies #131, #132, #133, #135 and #136 give zero findings through `CheckMarkdown`. The functions are not yet wired to the CLI; that is Phase 5.

Development took two review rounds, plus a drift audit that found the phase aligned. The PR was squash-merged.

### Test Plan / Verification

- [x] Entry point: an impl-phase body with no `## Verification` gives the single `section:verification` error
- [x] `## Foo` inside a fenced block in Scope is content (backtick, tilde and long fences)
- [x] `## N. Verification Steps` maps to `verification_steps`; other `## N. <Name>` headings map to `component` with that name
- [x] Round trip (render, parse back, equivalent map, zero findings, clean `CheckMarkdown`) for all four built-in schemas, with a `###` sub-heading and a fenced `## ` line in content
- [x] Preamble text before the first `##` gives no finding
- [x] `**Key:** value` in section content is content, not metadata
- [x] Raw `<details>`, a setext heading and an unclosed fence under an unknown heading are errors
- [x] `CheckMarkdown` refuses a list-item fence that ends early, a level-1 heading and an HTML block
- [x] Real bodies #131, #132, #133, #135 and #136 give 0 findings through `CheckMarkdown` (probe)
- [x] `go test ./...`, `go vet ./...` and `gofmt -l tool` are clean

### Decisions

- **Round-trip tests start from a section map (map → `Render` → `ParseMarkdown`), not from YAML draft text.** — The YAML loader is already covered by Phase 3's tests, so running it inside this loop would add no coverage of the new parser.
- **No new setext rule in the parser; Phase 3's `scanContent` check is reused through `Check`.** — `scanContent` already reports a setext underline directly under a non-blank line, and the parser splits sections on `## ` only.
- **`CheckMarkdown` runs goldmark's `verifyBody` over the whole body, preamble included.** — The verifier can't tell preamble from sections, so a level-1/2 heading or an HTML block in the preamble is refused. This keeps the parser in line with GitHub's rendering.
- **`checkContentStructure` now takes `(loc, what, content, line)`, so content under unknown headings can reuse it.** — That content needed the same guards. Messages for known sections are unchanged.
- **Section content is stored with leading blank lines and trailing whitespace stripped, and CRLF is normalized to LF.** — So that "line N of the section" counts from the first content line.
- **A `**Key:**` line is metadata only if it sits outside a fence and either comes first, follows a blank line, or directly follows another accepted metadata line. Fence state comes from `scanContent`, per line.** — This avoids a second scanner and matches where GitHub actually shows metadata (review B2).
- **The preamble is stored on the section map and checked by `checkContentStructure`; a body with no `## ` heading is treated as all preamble.** — This gives the preamble the shared content guards (review B3).

### Scope Changes

- **added:** The content guards (raw HTML, fences, setext, `## `, lone CR) now also run on content under unknown `## ` headings and on the preamble. — The orchestrator asked for the unknown-heading guards, and review B3 required the preamble ones. The drift audit confirmed both are within Component 4.

### Deferred / Watch Items

- **`ParseMarkdown`/`CheckMarkdown` are not yet wired to the CLI** — track in Phase 5 (#137)

### Corrections

- **Repeated `**Key:**` metadata line (review B1)**: the map silently kept the last value, so the parser and GitHub disagreed → keep the first value and report the repeat as an error naming both lines, worded like the YAML path's repeated-key error.
- **Metadata-shaped lines that GitHub doesn't show as metadata (review B2)**: lines inside a fence, or directly below a quote, list or paragraph line, were read as metadata → accept metadata only outside fences, and only first, after a blank line, or below another accepted metadata line; a schema key that fails this is an error with a fix.
- **Preamble skipped the content guards (review B3)**: `note\r## Fake`, a `---` under the metadata, and an unclosed fence in the preamble either passed or got misleading fixes → store the preamble on the map and run `checkContentStructure` on it, located as "above the first `## ` heading".
- **Garbled fix text for markdown locations (review N1)**: they used imperative "where" clauses, so messages read like "Edit it edit that text" → reworded to `above the first "## " heading` and `in the "**<Key>:**" line`.
- **Round-trip and preamble tests bypassed `CheckMarkdown` (review N2)**: the tests never exercised the goldmark backstop → they now also assert that `CheckMarkdown` is clean, and `ParseMarkdown`'s doc comment says that pairing it with `Check` skips goldmark verification.
- **Raw HTML in a metadata value was reported twice (orchestrator follow-up)**: both the metadata check and the preamble guard flagged it → that value is blanked in the scanned preamble, keeping line numbers, so only the metadata check reports it, naming the key.
- **Preamble messages said "line N of the section" (orchestrator follow-up)**: preamble lines are counted from the top of the body → they now say "line N of the body".
- **A rejected run of metadata lines gave conflicting errors (round 2 N1)**: each key got both a "missing" and a "directly below" error, quoting a long line → a key with a misplaced line is no longer also reported missing, and the error names the first non-metadata line of the run (quoted to 60 characters, with its line number) and says to add a blank line after it.
- **The misplaced-metadata message (round 2 N2)**: it claimed GitHub merges the line into a block that has in fact ended → it now states the parser's rule, and says "inside that quote or list item" only when the line above starts with `>` or a list marker.
- **Indented `**Key:**` lines (round 2 N3)**: they were neither read nor reported → indents under 4 columns (a tab counts as 4) are read as metadata, continuation lines may be indented further, and a line indented 4 or more columns after a blank line is an error.
- **NBSP counted as blank (round 2 N4)**: `TrimSpace` strips U+00A0, so a line containing only a non-breaking space counted as blank → a blank line is now `strings.Trim(line, " \t") == ""`.
- **A repeated metadata key outside the schema was an error (round 2 N5)**: keys outside the schema should only get the not-in-schema warning → repeats of those keys are skipped.
- **Process slip (`15d0399`)**: the N4 fix was committed while the N5 test committed just before it was still failing, and the NBSP test had a bug → the test was fixed in `095f9f9`, and the suite passes on every commit from the N5 fix onward.
