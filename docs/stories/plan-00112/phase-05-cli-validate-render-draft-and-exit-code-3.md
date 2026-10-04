# Phase 5: CLI validate, render --draft and exit code 3

**Status**: Completed
**Phase issue**: [#137](https://github.com/threehillpath/marvin-sdd/issues/137)
**Pull request**: [#146](https://github.com/threehillpath/marvin-sdd/pull/146)
**Implementation plan**: [#132](https://github.com/threehillpath/marvin-sdd/issues/132)

## Objective

Phase 5 of PLAN-00112: expose the check on the CLI. This adds exit code 3, `marvin template validate`, and `marvin template render --draft`, and schema origins report the override path. Help text lists the schema types from the embedded set.

## Scope

**In scope (impl plan Component 5):**
- **Exit code 3:**
  - a new `clierr` constructor with `Code: 3`, and the `clierr.go` doc comment lists code 3,
  - exit 3 whenever the Result holds at least one error finding, including loader findings,
  - warnings alone exit 0,
  - an unknown type, unreadable input, or malformed schema or override exits 1, never a silent fallback to the built-in.
- **`resolveSchema`:** returns the override's path, so the origin reads `project override: <path>`.
- **`marvin template validate <type> (--draft <file.yml> | --body-file <file.md>) [--title <t>] [--json]`:**
  - no config needed and no GitHub call,
  - the formatted Result goes to stdout,
  - with `--body-file`, `--title` supplies the title, otherwise a `title` error is reported,
  - `--json` gives an object (`schema`, `origin`, `findings[]`) for non-agent callers.
- **`marvin template render <type> --draft <file.yml>`:** rendered markdown to stdout and warnings to stderr. Any error exits 3 with the findings on stderr.
- `--draft` and `--body-file` are mutually exclusive. Both, or neither (without `--skeleton`), is a usage error (exit 1).
- **Help text:** the `Use` strings for `render` and `validate` list the types from the embedded schema set, so `quick-task` is included.

**Out of scope:** `issue create` / `issue edit` (Phase 6). Skills and docs (Phase 7).

## TDD Entry Point

A CLI test: `template validate impl-phase --draft <missing-verification.yml>` exits with code 3. Its first stdout line is `schema: impl-phase (built-in)`, and a later line begins `error section:verification`.

## Components

- `tool/internal/clierr/clierr.go`, `tool/internal/cli/errors.go`: exit code 3.
- `tool/internal/cli/root.go` (`newTemplateCmd`): `validate`, render `--draft`, generated `Use` strings.
- `tool/internal/cli/handlers.go`: `resolveSchema` origin path, validate and render handlers.

## Verification

```bash
cd tool && go test ./...
marvin template render --help | grep quick-task
marvin template validate impl-phase --draft ok.yml; echo $?               # schema: impl-phase (built-in) / exit 0
marvin template validate impl-phase --draft no-verification.yml; echo $?  # error section:verification … / exit 3
marvin template validate impl-phase --draft hash-in-value.yml; echo $?    # error metadata:Implementation Plan line N … double quotes / exit 3
marvin template validate impl-phase --draft dedented-line.yml; echo $?    # error draft line N: unknown key "Note" … / exit 3
marvin template validate impl-plan --draft dup-component.yml; echo $?     # error draft line N: key "component" repeated … / exit 3
marvin template validate impl-phase --draft folded-scope.yml; echo $?     # error section:scope line N: … use "|" / exit 3
gh issue view 131 --json body -q .body > /tmp/131.md
marvin template validate arch-plan --body-file /tmp/131.md --title "[PLAN-00112-ARCH] Schema-checked plan issue creation and edits"   # exit 0
```

## Success Criteria

- [ ] The entry-point test passes: exit 3, the `schema:` first line, and the `error section:verification` line.
- [ ] Loader findings (a draft that won't load) exit 3, not 1.
- [ ] Warnings alone exit 0. An unknown type, unreadable file, or malformed schema or override exits 1.
- [ ] A project override in a temp dir is used and reported as `project override: <path>` (CLI test).
- [ ] `render --draft` prints rendered markdown on success, and exits 3 with the findings on stderr on error.
- [ ] `render --help` and `validate --help` list all four types, generated from the embedded schemas.
- [ ] Arch plan #131's existing body validates with exit 0 via `--body-file`.
- [ ] `go test ./...` passes.

---

## Implementation

This phase adds the CLI surface for plan-issue template validation. A new exit code, `clierr.NonConforming` (3), is returned for any error finding, including loader findings. Warnings alone exit 0, and an unknown type, unreadable input, malformed schema or override, or a usage error exits 1. `marvin template validate <type> (--draft <file.yml> | --body-file <file.md>) [--title <t>] [--json]` makes no config or GitHub call. A shared `checkInput` helper in `tool/internal/cli/handlers.go` runs `template.Render` (including the goldmark backstop) for YAML drafts and `CheckMarkdown` for markdown bodies, then prints a formatted `Result` whose first line is `schema: <type> (<origin>)`. The origin is `built-in` or `project override: <path>`, with `resolveSchema` and `findSchemaOverride` returning the override path. `--json` output uses the new exported `Result.Sorted()`, the same errors-before-warnings ordering `Format` uses.

`marvin template render <type> --draft` writes the rendered markdown to stdout and warnings to stderr, and exits 3 with the findings on stderr when there are errors. `--skeleton` and `--guidance` are unchanged. Supplying none of the modes, or more than one, is a usage error (exit 1), and `--draft` combined with `--skeleton` or `--guidance` is rejected. On `validate`, `--title` is rejected with `--draft` (including an explicit empty value, detected through `Flags().Changed`) because the title comes from the draft's `title:` key; with `--body-file` it supplies the title. The `render` and `validate` `Use` strings and the missing-type message are generated from `template.DefaultSchemaNames()`, replacing a hardcoded `builtInTypes`. The `RunWithStreams` doc, the `template` group Short and the `--guidance` help were updated for the exit-3 stdout exception, and the CLAUDE.md exit-code line gained code 3.

Tests in `template_validate_test.go` and `template_render_test.go` cover the entry point (a missing-verification `impl-phase` draft exits 3), loader findings, warnings and operational errors, override origin and use, `--body-file` and `--title`, `--json` ordering, mode exclusivity, and the goldmark backstop on both paths (mutation-verified), with a shared `conformingPhaseBody` fixture. Several tests arrived green because the first implementation commit was broader than criterion 1 required, and they were kept as regression guards. The PR went through a request-changes review (B1, B2, N1–N6) and a re-review (N1–N3), with each fix landing as its own commit and Notes entry before merge.

### Test Plan / Verification

- [x] Entry point: `validate impl-phase --draft <no-verification>` exits 3, with `schema: impl-phase (built-in)` first and an `error section:verification` line
- [x] `go test ./...`, `go vet ./...` and an empty `gofmt -l`
- [x] Manually: `validate arch-plan --body-file` on #131's body exits 0 (and #132's body as `impl-plan`)

### Decisions

- **Schema origin label is `built-in` or `project override: <path>`** — the spec's entry-point line (`schema: impl-phase (built-in)`) requires it; it replaces `built-in schema` and is recorded in the PR Notes after drift audit C1.
- **`render` has no `--body-file`; the both/neither usage error lives on `validate`** — a markdown body is already rendered, so `render` has no use for it. `render` instead rejects `--draft` combined with `--skeleton` or `--guidance`.
- **Exit code 3 (`clierr.NonConforming`) covers any error finding, loader findings included** — it separates non-conforming content from operational and config failures. Warnings alone exit 0, and an unknown type, unreadable input or malformed schema or override stay exit 1. The CLAUDE.md exit-code line was updated to match.
- **`Result.Sorted()` is the single ordering for text `Format` and `--json`** — `--json` kept raw `Check` order, so a warning could precede an error, unlike the text output.
- **The built-in type list in the missing-type message comes from `template.DefaultSchemaNames()`** — the hardcoded `builtInTypes` duplicated it and could drift. The types now appear in sorted order, and no test or doc depended on the old order.
- **The `render` and `validate` `Use` strings are generated from the embedded schema set** — help stays in sync with the embedded types, so `quick-task` is listed.

### Scope Changes

- **added:** the CLAUDE.md exit-code contract line gained code 3 — the new exit code changes the documented contract.
- **added:** `template.DefaultSchemaNames()` and `Result.Sorted()` as exported helpers beyond the spec's CLI surface — needed for generated help and type lists and for consistent ordering. The drift audit judged both in scope.

### Deferred / Watch Items

- **No `render --draft` goldmark-backstop test of its own** — it shares `checkInput` with `validate`, so the validate tests cover it. Track in phase 6 (#138), which adds the second caller of exit 3.
- **No `RunWithStreams` unit test for exit code 3 specifically** — track in phase 6 (#138).
- **`TestNamesDeriveTaskJSON` fails when `tool/` is built outside the repo** (it needs a plan-workflow config in a parent directory). It predates this PR. Track in #147, a standalone bug outside PLAN-00112.

### Corrections

- **Goldmark backstop tests added for `validate --draft` and `--body-file`** (`<div>` block in Scope) (review B1): no test failed if `checkInput` used `Check` or `ParseMarkdown` instead of `Render` or `CheckMarkdown` → two tests now fail under that mutation, confirmed before reverting.
- **`validate --draft --title` is now a usage error (exit 1)** (review B2): `--title` was silently dropped on the draft path and exited 0 → the title comes from the draft's `title:` key, so a supplied `--title` is refused with a message saying so.
- **Override test asserts exit 0 and the verdict change** (review N2): it discarded the exit code and only showed the override was reported, not used → an `impl-phase` override without `verification` now makes the missing-verification draft exit 0.
- **`Result.Sorted()` added and used by `--json`** (review N3): `--json` kept raw `Check` order → one exported ordering shared by `Format` and the JSON output, with a test that errors come first.
- **`render --draft` exclusivity tests and a stale comment** (review N4): `--draft` with `--skeleton` or `--guidance` was untested and a comment said draft input was unavailable → tests added, comment fixed.
- **`RunWithStreams` doc and template help text** (review N5): a duplicate `*CLIError` bullet, a claim that stdout is empty on error (contradicted by exit 3), a stale `template` Short and `--guidance` usage → merged, exit-3 exception noted, help updated.
- **`builtInTypes` removed** (review N6): it duplicated `DefaultSchemaNames()` → the missing-type message uses the single source, types in sorted order.
- **`validate --draft --title ""` is now a usage error** (re-review N1): the B2 guard tested the value, so an explicit empty `--title` was still dropped → the guard uses `Flags().Changed("title")`, and `--body-file --title ""` still reports a title error (exit 3).
- **`TestTemplateValidateBodyFile` uses the shared `conformingPhaseBody` fixture** (re-review N2): it duplicated the body byte for byte, so the copies could drift silently → one const, moved above the test.
- **`resolveSchema`/`findSchemaOverride` docs and the `render` Short** (re-review N3): the docs named the old origin labels and omitted the returned path, and the `render` Short omitted `--draft` → updated.
- **PR Notes amended** (review N1, drift C1): they only said criteria 2 and 3 were green on arrival and omitted the origin rename → they now describe commit 1d857b8's extra implementation, the `--json` registration that was ignored until 4b352de, and the `built-in` label change.
