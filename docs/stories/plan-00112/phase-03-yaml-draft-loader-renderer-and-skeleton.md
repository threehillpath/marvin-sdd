# Phase 3: YAML draft loader, renderer and skeleton

**Status**: Completed
**Phase issue**: [#135](https://github.com/threehillpath/marvin-sdd/issues/135)
**Pull request**: [#143](https://github.com/threehillpath/marvin-sdd/pull/143)
**Implementation plan**: [#132](https://github.com/threehillpath/marvin-sdd/issues/132)

## Objective

Phase 3 of PLAN-00112: add the YAML draft. A loader built on `yaml.Node` turns a draft into the section map and reports every case of silent truncation as a finding. `Render` works from the section map, `--skeleton` emits an empty YAML draft, and the old JSON `--sections` / `--meta` render path is removed.

## Scope

**In scope (impl plan Component 3):**
- **Draft format:**
  - `title:`, a `metadata:` map, and a `sections:` map keyed by schema id,
  - a non-repeatable section is a `|` block,
  - a repeatable section that isn't named is a list of `|` blocks,
  - a named section is a list of `{name, content}`.
- **Node-based loader findings** (exit 3 later, line plus fix in each):
  - a repeated key in any mapping (Node decoding drops yaml.v3's duplicate-key error),
  - a comment attached to any scalar (an unquoted `#` cut the value): title, metadata, plain-scalar section content, `name`,
  - section or named `content` that isn't a literal `|` block (plain, quoted, or `>` folded),
  - an unknown key at the top level, under `metadata:` or `sections:`, or in a named entry (an error here, unlike the markdown path),
  - a wrong node type.
- **Parser-error mapping** (location `draft`):
  - an unquoted title is detected from the raw line `^title:\s*\[` → quote the title,
  - an unescaped inner `"` in a quoted value → escape as `\"` or use single quotes,
  - `found unknown escape character` → escape as `\\`,
  - the line comes from yaml.v3's message, else the nearest locatable line, else `line unknown`,
  - unmapped messages get the fallback text in impl plan §3,
  - never raw parser text alone.
- **`Render`:** takes the section map, runs Phase 2's check first, and refuses on error. The old required and non-repeatable checks are deleted. One running ordinal across numbered sections. `named: false` sections render their literal heading.
- **`Skeleton`:** emits an empty YAML draft with a double-quoted title placeholder, every metadata key as `""`, every section key, and guidance as YAML comments.
- **Render rewiring:** `runTemplateRender` uses the new `Render`/`Skeleton`. `--sections` and `--meta` are removed with their JSON decoding. `template_render_test.go`'s skeleton assertions are updated.
- **`template/render_test.go`:** its 8 tests move to the section-map API. `## 3. Verify Step` becomes `## 3. Verification Steps`.

**Out of scope:** the markdown parser and round-trip tests (Phase 4). `render --draft`, `template validate`, exit code 3 and help text (Phase 5).

## TDD Entry Point

A test that loads a minimal valid `impl-phase` draft, renders it, and asserts that the headings and metadata lines come out in schema order. Then one test per loader case, each asserting exactly one finding with a line and a fix:
- `Implementation Plan: #132 ([PLAN-00112])` (comment cut),
- `objective: Fix #112 thing` under `sections:`,
- `name: Fix #112 handling` (impl-plan),
- two `component:` keys (impl-plan),
- `scope: >`,
- a dedented `  Note: x` line inside `scope:`,
- unquoted `title: [PLAN-00112-1] X`,
- `title: "[PLAN-00112-1] Add "validate" command"`,
- `Status: "Match \d{5}"`,
- a `## Sub` line in `scope: |` outside a fence.

## Components

- `tool/internal/template/render.go` and a new draft-loader file in `tool/internal/template/` (`gopkg.in/yaml.v3`).
- `tool/internal/template/render_test.go`: migrated.
- `tool/internal/cli/handlers.go` (`runTemplateRender`) and `tool/internal/cli/root.go` (render flags): `--sections` and `--meta` removed.
- `tool/internal/cli/template_render_test.go`: skeleton assertions.

## Verification

```bash
cd tool && go test ./...
marvin template render impl-phase --skeleton   # YAML: quoted title placeholder, metadata keys, section keys, guidance comments
marvin template render impl-plan --sections x.json 2>&1; echo $?   # unknown flag, exit 1
```

## Success Criteria

- [ ] A valid draft renders with headings and metadata in schema order. Numbered sections share one ordinal, and `verification_steps` renders `## N. Verification Steps`.
- [ ] Every loader case listed in the TDD entry point yields exactly one finding with a line (or `line unknown`) and a concrete fix, and none passes after being truncated.
- [ ] The inner-quote case gives an escaping fix, not "quote the title".
- [ ] `Render` refuses on any check error. No duplicate required or non-repeatable logic remains in `render.go`.
- [ ] `--skeleton` emits a YAML draft with a double-quoted title placeholder and guidance comments.
- [ ] `--sections` and `--meta` no longer exist.
- [ ] `go test ./...` passes, including the migrated `render_test.go` and `template_render_test.go`.

---

## Implementation

Phase 3 of PLAN-00112 replaces the JSON input path of `marvin template render` with a YAML draft path. `LoadDraft` decodes a draft through `yaml.NewDecoder` into a `yaml.Node` tree and walks it into the section map. Every case where yaml.v3 would accept the input but lose content becomes a finding with a line and a fix: document markers and second documents, any head/line/foot comment, repeated keys, tags/anchors/aliases, non-`|` section content, unknown keys, and wrong node types. yaml.v3 parser errors are mapped to targeted fixes: inner quotes in a title or value, invalid backslash escapes, and undefined aliases located by a raw-line search.

`Render` now takes the section map and runs `Check` first. It refuses on any error, so the duplicate required/non-repeatable logic is gone. It renders in schema order with one running ordinal and `## N. Verification Steps`. Before returning, it parses its own body with goldmark v1.8.6 + GFM (`verify.go`). It refuses unless the level 1 and 2 headings are exactly the emitted ones, no heading is swallowed, and there is no HTML block or inline comment or banned tag. Findings are located through a rendered-line origin map at `section:<id>` or `metadata:<Key>`.

The structure guards live in the shared `scanContent` scanner, which `scanFences` and `FindH2Lines` now wrap. It bans raw HTML (`<!--`, `details`/`pre`/`script`/`style`/`textarea`) outside fences and CommonMark-paired code spans (`maskCodeSpans`), flags any setext underline after a non-blank line, and rejects stray `\r`/`\n` and empty headings. `Check` also carries the three phase-2 nits: `checkNumbering` uses `templateHeading`, the markdown empty-entry test asserts `is empty` and `## 1. <Name>`, and the content-H2 error quotes the actual line.

The CLI gains `--skeleton`, which emits a comment-free YAML draft (a double-quoted title placeholder, every metadata key as `""`, every section key), and `--guidance`, which prints the draft-writing rules as plain text. The two flags are mutually exclusive. `--sections`, `--meta` and the unused `template.KV` are removed. Plain `render <type>` without `--skeleton` exits 1 with a pointer until Phase 5 wires `--draft`.

Development ran as test-first commit pairs through four review rounds, and the PR was squash-merged. The drift audit found every success criterion met, with SC5 amended by the author's comment-free skeleton decision.

### Test Plan / Verification

- [x] `go -C tool test ./...` passes (12 packages with tests), `go vet ./...` is clean, and `gofmt -l tool` prints nothing
- [x] TDD entry point: a minimal `impl-phase` draft written with metadata and sections out of order renders in schema order (`TestRenderMetadataAndHeadingsInSchemaOrder`)
- [x] One loader test per case: comment cuts (metadata, section content, `name:`, title), repeated keys (including two `component:` keys), `scope: >`, a dedented `  Note: x`, an unquoted `title: [..]`, an inner `"` (escape fix, not "quote the title"), a `\d` escape, and `## Sub` in content
- [x] `marvin template render impl-phase --skeleton` prints a YAML draft; `marvin template render impl-plan --sections x.json` exits 1 with `unknown flag`
- [x] Structure guards and parser backstop: tests for raw HTML, setext headings, HTML blocks of every type, swallowed and extra headings, and list-item fences. The no-finding cases are fenced content, a thematic break after a blank line, inline code, tables and task lists.

### Decisions

- **Drafts accept no YAML document markers (`---`/`...`); any column-0 marker is an error and a second document is an error.** — Round 1 B1: `yaml.Unmarshal` reads one document, so everything after a marker was silently dropped. Author decision: drafts never need markers. `LoadDraft` decodes with `yaml.NewDecoder`.
- **Drafts accept no YAML comments; every head, line and foot comment on any node is a finding.** — Round 1 B2: a dedented `#113` or a trailing `### Extra` was silently dropped as a head or foot comment. Author decision: comments are errors, not preserved or ignored.
- **Explicit YAML tags, anchors and aliases are findings, not parsed.** — Round 2 N3: `!draft`, `&draft` and `!Important` silently stripped leading text from values.
- **The setext guard is conservative: any `-`/`=` underline directly after any non-blank line outside a fence is an error.** — Round 1 B3: the paragraph-start heuristics missed `#112 ...` above `---`. Author decision: a conservative rule that costs a little formatting is acceptable. A legitimate `---` under a list item or quote now needs a blank line.
- **Raw HTML is banned instead of tracked: any `<!--` or opening/closing `details`/`pre`/`script`/`style`/`textarea` tag outside a fence or inline code is an error in content, metadata, entry names and the title.** — Round 2 B1, B2, N1: the comment, `<details>` and raw-block state machines kept hiding later lines from the other guards. Author decision: stop parsing raw HTML and ban it. The tag match requires a boundary, so `<prefix>` and `<stylesheet>` stay allowed.
- **`Render` parses its own output with goldmark v1.8.6 + GFM as a backstop.** It refuses if the level 1/2 headings differ from the emitted ones, if any HTML block exists, or if an inline `<!--` or banned tag appears. — Round 3 B1: three rounds of gaps in the regex scanner (unclosed `<?php`, CDATA, `DOCTYPE`, list-item fences). Author decision: add a CommonMark parser backstop.
  - Findings map back to section/metadata and line through a rendered-line origin map.
  - The scanner checks stay in `Check` for their specific messages.
  - A lone inline tag on its own line is refused, an accepted conservative trade-off.
- **`--skeleton` is comment-free and guidance moves to a new `--guidance` flag.** — Follows from the no-YAML-comments decision. The two flags are mutually exclusive, and passing both or neither exits 1.
- **The structure guards live in the shared `scanContent` scanner.** — Carried work from the #135 comments. `scanFences` and `FindH2Lines` wrap it rather than a second scanner existing, so both the YAML and markdown input paths get the guards.
- **Plain `template render <type>` without `--skeleton` now exits 1 with a message pointing at `--skeleton`.** — `render --draft` is Phase 5. The old JSON input is gone and the new input isn't wired yet.
- **A parser error's "nearest locatable line" is yaml.v3's reported line, else the raw line found by title/inner-quote/escape inspection, else `line unknown`.** — yaml.v3's line is unreliable for `did not find expected key`, so that error blames the first line that fails to parse alone, with no limit on how far from the reported line it is.
- **Inline-code spans are masked using CommonMark pairing.** — Round 3 B1: the regex pairing differed from CommonMark, so an unwrapped `<details>` passed after an unmatched ``` run. A backtick run pairs only with the next run of equal length, unmatched and backslash-escaped backticks are literal, and spans can cross lines.
- **`verification_steps` renders as `## N. Verification Steps` under one running ordinal, and the unused `template.KV` type is removed.** — Follows from the section-map render. `KV` had no users once the JSON path went.

### Scope Changes

- **added:** `--skeleton` no longer carries guidance comments, which changes SC5, and a `--guidance` flag is added. — Author decision after round 1 B2: drafts take no YAML comments, so guidance moved to a separate plain-text flag.
- **added:** `--guidance` output with ten reworded rules covering headings, list fences, HTML blocks, setext and markers. — Rounds 2 N5 and 4 N4: the guidance omitted rules that the checker and backstop enforce.
- **added:** goldmark v1.8.6 (with `extension.GFM`) as a dependency, used only by `verify.go`. — Round 3 author decision to add a parser backstop to `Render`.
- **added:** the raw-HTML ban extends to metadata values (including unknown markdown keys), entry names and the title. — Rounds 2 B2 and 3 N3: values outside section content skipped the guards and could collapse later sections.
- **removed:** the `--sections` and `--meta` flags and `template.KV`. — Phase spec (SC6): YAML drafts replace the JSON input path, and `KV` was dead code afterwards.
- **removed:** the raw-HTML state machines in `scanContent` and the setext paragraph-start exceptions (`nonParagraphStartRe`). — Replaced by the raw-HTML ban and the conservative setext rule.

### Deferred / Watch Items

- **`template validate` must run the same rendered-body verification.** The parser backstop lives in `Render`, not `Check`, so `validate` should call `Render` on the loaded draft and discard the body. — track in Phase 5 (#137)
- **Plain `template render <type>` exits 1 only until `render --draft` is wired.** — track in Phase 5 (#137)
- **Skills that still call the removed `--sections`/`--meta` flags need updating to `--skeleton`/`--guidance`.** — track in Phase 7 (#139)
- **Accepted conservative trade-offs:** a lone inline HTML tag on its own line is refused, and a `---` directly under a list item or quote now needs a blank line above it. — revisit only if these cause problems in practice

### Corrections

- **Column-0 `---`/`...` YAML document markers are rejected with line and fix, and `LoadDraft` uses `yaml.NewDecoder` so a second document is an error (round 1 B1)**: `yaml.Unmarshal` reads one document, so everything after a marker was silently dropped → reject any marker line, with a second-document check as a backstop.
- **Every head, line and foot YAML comment is reported with line and fix, and the skeleton and tests lost their comments (round 1 B2)**: only `LineComment` was read, so a dedented `#113` line, a trailing `### Extra` or a `#...` continuation was dropped as a head or foot comment → report every comment on any node, with a fix to indent it into the `|` block, quote the whole value, or delete it.
- **The setext guard flags any underline directly after a non-blank line, and the `=` fix no longer offers a horizontal rule (round 1 B3, N5)**: `nonParagraphStartRe` skipped paragraphs starting with `#112`, a `<tag>`, an indented continuation or `2.` → drop the exceptions and check every line outside a fence.
- **Inner-quote and escape messages made precise (round 1 N1, N2)**: a valid quoted key was blamed, a partly quoted value got the escape fix, and the message hard-coded a `\d{5}` example → blame a line only if it fails to parse alone and is at most one line past yaml.v3's reported line (no limit for `did not find expected key`), quote the offending line with the invalid backslashes doubled, and also cover `\U`/`\x` hex errors.
- **`Check` rejects `\r` and `\n` in the title, metadata values and entry names, with the line (round 1 N3)**: only `\n` in titles and metadata was caught, and names only by `Render` with no line → reject them in `Check`, and `Render`'s re-parse finding carries the entry line.
- **HTML guards ignore complete comments and see comments after tags (round 1 N4)**: a `<details>` after a complete comment was skipped, a `<!--` after a tag was missed, and a `<details>` inside a comment was counted → strip complete comments and inline code before counting. The raw-HTML ban later superseded this.
- **Raw HTML is banned across content, metadata, names and the title (round 2 B1, B2, N1)**: the state machines hid later lines from the setext check, skipped the text after `-->` and `</pre>`, and never looked at metadata, names or the title (a `<details>` in a metadata value collapsed the body) → remove the state machines and error on the banned constructs outside fences and inline code, with a backtick-or-remove fix.
- **One finding per comment line, located exactly (round 2 N2)**: only the first line of a multi-line comment was reported, and a content line with the same text could be blamed → own-line comments match the whole trimmed line, trailing comments match by suffix, block-scalar content is never matched, and each raw line is reported once.
- **YAML tags, anchors and aliases reported as findings (round 2 N3)**: `!draft Upcoming`, `&draft Upcoming` and `- name: !Important first one` silently dropped the marker text → a finding at its line for any node with a tag, anchor or alias, and `kindName` learns alias nodes.
- **Document-marker advice and detection (round 2 N4)**: indenting a `---` under text led straight into the setext error, and `... text` had no line → the advice says to indent it and leave a blank line above, and the pattern `^(?:---|\.\.\.)(?:[ \t]|$)` catches a marker followed by text.
- **`--guidance` lists every enforced rule and the schema is resolved before the flag errors (round 2 N5, N6)**: the guidance omitted rules the checker enforces, and `render nosuch` suggested `--skeleton`, which then failed → the guidance states the blank line before `---`, no `---`/`===` under text, closing code fences in the same section and the raw-HTML ban, with each asserted in the test; an unknown schema is reported alone.
- **Inline-code pairing follows CommonMark via `maskCodeSpans` (round 3 B1)**: `inlineCodeRe` stripped every backtick pair, so an unwrapped `<details>` passed after unmatched runs, inside multi-line spans, or between escaped backticks → a run pairs only with the next run of equal length, unmatched and escaped backticks are literal, and spans work over a whole paragraph.
- **`Render` verifies its body with a goldmark + GFM parse and maps findings back to section/metadata and line (round 3 B1, N1, N2)**: the line scanner kept missing markdown's long tail (an unclosed `<?php`, `<![CDATA[`, `<!DOCTYPE`, or a list-item fence ended early by a dedented line) → the parser backstop replaces the `FindH2Lines` re-check inside `Render`.
- **The raw-HTML check also runs on unknown markdown metadata keys (round 3 N3)**: values of keys not in the schema skipped the ban → run the check next to the not-in-schema warning.
- **A block's content indent comes from its indentation indicator or its first non-blank text line (round 3 N4)**: `rawLines` blanked every line indented more than the key, so a `#` line between the header and deeper text was reported as `line unknown` → blank only lines indented at least as far as the block's content.
- **An undefined alias is located and quoted (round 3 N5)**: yaml.v3's unknown-anchor error has no line → search the raw lines for `*name` and quote that line.
- **Stale `SchemaSection.Guidance` doc comment fixed (round 3 N6)**: it said `Skeleton` emits YAML comments → it now says the `Guidance` function prints the text as plain text.
- **Empty headings (`#`, `- #`, `> #`) are located and the message suggests `\#` (round 4 N1)**: they produce a heading node with no text line, so the finding had no section or line → fall back to the node's position.
- **`Check` rejects a lone `\r` left after CRLF normalization (round 4 N2)**: goldmark ignores a lone CR but GitHub renders it as a line break, so `a\r## X` got through → reject it with its line within the section and a fix to replace it with a line break.
- **The undefined-alias search matches only where a YAML value can begin, with a name boundary (round 4 N3)**: it blamed an earlier quoted line containing the same `*name` → anchor the match at the start of the line or after `:`, `-`, `,`, `[` or `{` plus whitespace.
- **`--guidance` states the backstop's rules, reworded as ten rules (round 4 N4)**: it omitted rules the backstop enforces (never `#` or `##` anywhere, list-item fence indentation, no line starting with an HTML tag, `<?` or `<!`) → rules added, wording cleaned up, and the test asserts short distinctive phrases.
