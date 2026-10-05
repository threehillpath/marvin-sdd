# Phase 1: Title identifier kinds

**Status**: Completed
**Phase issue**: [#133](https://github.com/threehillpath/marvin-sdd/issues/133)
**Pull request**: [#140](https://github.com/threehillpath/marvin-sdd/pull/140)
**Implementation plan**: [#132](https://github.com/threehillpath/marvin-sdd/issues/132)

## Objective

Phase 1 of PLAN-00112: add title identifier kinds. A new classifier reads a title's leading bracket and returns arch / impl / phase / task, and `[TASK-XXXXX]` titles are parsed. `marvin parse title` and `issue tree` use it without changing any existing output for non-task titles.

## Scope

**In scope (impl plan Component 1):**
- A classifier returning `(names.Kind, bool)`, reusing the existing four-value `names.Kind`. If `parse` → `names` would be an import cycle, move the enum to `parse` and alias it in `names`. No third kind enum. The bool is the only "not found" signal, because `names.Kind`'s zero value is `Arch`.
- Leading bracket only (optional whitespace, then `[`), the same anchoring `TitleSlug` uses:
  - `[PLAN-XXXXX-ARCH]` is arch,
  - `[PLAN-XXXXX]` and `[PLAN-XXXXX-A]` are impl,
  - `[PLAN-XXXXX-N]` and `[PLAN-XXXXX-A-N]` are phase,
  - `[TASK-XXXXX]` is task,
  - anything else is not found, including an unreplaced `XXXXX`.
- Suffix letters follow `PlanIdent`'s rules: any length, case-insensitive, upper-cased.
- `[TASK-XXXXX]` parsing returns the task number.
- **`marvin parse title`:**
  - `found:` and the plan fields stay driven by `PlanIdent`, unchanged.
  - `kind:` is printed only when the classifier matches the leading bracket.
  - Task titles print `found: true`, `kind: task`, `task: <n>`, `task_number: TASK-XXXXX` (from `names.TaskNumber`) and `slug:`.
  - `--json` gains the same fields.
- **`issue/tree.go` `kindOf`:**
  - uses the classifier when it matches, so task maps to `task`,
  - otherwise falls back to today's `PlanIdent` mapping, unchanged,
  - tree membership stays on `PlanIdent`.

**Out of scope:** the conformance check and anything template-related (Phases 2–5). The doc kind lists that gain `task` are updated in Phase 7.

## TDD Entry Point

A table test in `tool/internal/parse/parse_test.go`:

| Title | Expected |
|---|---|
| `[TASK-00091] Fix X` | task, number 91 |
| `[PLAN-00112-ARCH] X` | arch |
| `[PLAN-00112-A-2] X` | phase |
| `[TASK-00140] Fix regression from [PLAN-00112-3]` | task |
| `[PLAN-00042-a] X` | impl (suffix `A`) |
| `Fix X` | not found |
| `[PLAN-XXXXX-ARCH] X` | not found |

## Components

- `tool/internal/parse/parse.go`: the classifier and TASK parsing.
- `tool/internal/names/names.go`: `Kind` reuse, or the alias if a cycle forces it.
- `tool/internal/cli/handlers.go` (`runParseTitle`, `parseTitleOutput`): the new `kind:` and task lines.
- `tool/internal/issue/tree.go` (`kindOf`): use the classifier, with the `PlanIdent` fallback.

## Verification

```bash
cd tool && go test ./...
marvin parse title "[TASK-00091] Fix X"                          # found: true / kind: task / task: 91 / task_number: TASK-00091
marvin parse title "[TASK-00140] Fix regression from [PLAN-00112-3]"  # kind: task
marvin parse title "[PLAN-00112-ARCH] X"                         # existing keys unchanged + kind: arch
marvin parse title "Notes on [PLAN-00042-1]"                     # found: true + plan fields as today, no kind: line
```

## Success Criteria

- [ ] The classifier returns `(names.Kind, bool)` and classifies from the leading bracket only. All table cases above pass.
- [ ] `parse title`'s `found:` and plan fields are unchanged for every non-task title, including `Notes on [PLAN-00042-1]` (CLI test). `kind:` appears only when the leading bracket classifies.
- [ ] Task titles print `task_number: TASK-XXXXX`, matching `names derive --task`.
- [ ] `issue tree` output is byte-identical for every non-task title (existing tree tests pass). A task sub-issue prints kind `task`.
- [ ] No new kind enum exists alongside `names.Kind`.
- [ ] `go test ./...` passes.


---

## Implementation

This phase adds title-kind classification to the `parse` package and uses it in the CLI and the issue tree.

**Classifier (`parse.Classify`):**
- `parse.Classify(title) (names.Kind, bool)` reads only the title's leading bracket, found with `leadingBracketRe`. That regex anchors the same way as `TitleSlug`.
- A `[TASK-XXXXX]` token matches the new `taskIdentRe` and returns `names.Task`.
- A PLAN token must first exactly match one of the accepted forms: an optional letters suffix such as `ARCH`, then an optional positive phase number. It is then passed to `PlanIdent`, so the classifier and `PlanIdent` always agree on the kind.
- Malformed tokens such as `[PLAN-000421]`, `[PLAN-00042-0]` and `[PLAN-00042-1-2]` return not found.
- The bool is the only "not found" signal, because the zero value of `names.Kind` is `Arch`.

**Related helpers:**
- `parse.TaskIdent` parses `[TASK-XXXXX]` titles.
- `names.Kind` gained a `String()` method, which renders an unknown value as `Kind(N)`. It replaced the duplicate mappings in `cli.kindName` and `issue.kindOf`.

**`marvin parse title` (`runParseTitle` in `cli/handlers.go`):**
- It prints a `kind:` line only when the leading bracket classifies.
- Task titles print `found: true`, `kind: task`, `task:`, `task_number:` and `slug:`. They leave out the plan fields, even when a later bracket is a PLAN token.
- `--json` output gains the same fields.
- Existing exact-output tests for `[PLAN-00042-A-3]` gained the extra `kind: phase` line.
- A CLI test confirms `Notes on [PLAN-00042-1]` keeps `found: true` and the plan fields with no `kind:` line.

**`issue tree` (`issue/tree.go`):**
- `kindOf` checks `Classify` first, so a TASK sub-issue is labelled `task`.
- Tree membership still comes from `PlanIdent`, and output is unchanged for non-task titles.

**Process:**
- The work followed TDD: each implementation commit was preceded by its `test:` commit. The entry point was `TestClassify`, and the first commit included stubbed `Classify` and `TaskIdent`. The later tests were `TestTreeTaskSubIssueKind`, `TestTreeArchRootTaskSubIssueKind` and a stubbed test for `names.Kind.String`.
- A final whitespace-only `gofmt` commit cleaned `names_test.go` and `render_test.go`.
- The plan-drift audit reported all six success criteria met.
- The code review returned verdict `comment`, with no blocking findings and 4 nits. N1–N3 were fixed in the PR. N4 was a process note: existing test expectations were changed in the implementation commit rather than the test commit.

### Test Plan / Verification

- [x] `TestClassify` table test (TDD entry point): every case from the phase issue, plus leading whitespace, a lowercase suffix and an unreplaced `XXXXX`
- [x] CLI tests: `Notes on [PLAN-00042-1]` keeps `found: true` and the plan fields with no `kind:` line; task output and `--json`
- [x] `TestTreeTaskSubIssueKind`: a task sub-issue is labelled `task`, and the existing tree tests are unchanged
- [x] `go test ./...` and `go vet ./...` pass

### Decisions

- **`names.Kind` is reused for classification, with no new enum or alias.** `parse` can import `names` without an import cycle, and SC5 rules out a second kind enum.
- **`Classify` only looks at the leading bracket, and it hands plan tokens to `PlanIdent`.** So the classifier and `PlanIdent` never disagree. `found:` and the plan fields still come from `PlanIdent`, and a bracket later in the title never changes the kind.
- **For a task title, `parse title` leaves out the plan fields, even when a `[PLAN-...]` bracket appears later in the title.** The spec's Verification section requires this.
- **Which issues belong to the tree is still decided by `PlanIdent`.** `kindOf` checks the classifier first, but only to pick each issue's label. So `issue tree` output is byte-identical for non-task titles, and TASK sub-issues are labelled `task`.
- **One `names.Kind.String()` method now produces the kind label.** It replaces the two separate mappings in `cli.kindName` and `issue.kindOf`, which fell back to different labels for an unknown kind, so a new kind would have been mislabelled without any warning.

### Scope Changes

- **added:** the `names.Kind.String()` method, which falls outside the original parse/CLI/tree scope. Review nit N2 raised it.
- **added:** `TestTreeArchRootTaskSubIssueKind`, for a TASK sub-issue under an arch plan. Review nit N3 raised it, and the test passed on its first run, so it serves as a regression guard.
- **added:** whitespace-only gofmt fixes to `names_test.go` and `render_test.go`. Both files were already unformatted on main.

### Corrections

- **`Classify` now checks that a leading PLAN token exactly matches `^\[PLAN-\d{5}(-[A-Za-z]+)?(-[1-9]\d*)?\]$` before calling `PlanIdent`**:
  - *What was wrong:* it passed any `[PLAN-...]` token to `PlanIdent`, which ignores trailing junk, so `[PLAN-000421]`, `[PLAN-00042-0]` and `[PLAN-00042-1-2]` all got a `kind:` (N1).
  - *Fix:* malformed tokens now return not-found, with new not-found rows in `TestClassify`. `found:` and the plan fields stay on `PlanIdent`.
- **A single `names.Kind.String()` replaces `cli.kindName` and the switch in `issue.kindOf`**:
  - *What was wrong:* the two copies fell back to different labels for an unknown kind ("task" in one, "impl" in the other) (N2).
  - *Fix:* both callers now use the one method, which got a stubbed test first.
- **The `TestTreeTaskSubIssueKind` comment is corrected, and an arch-rooted test is added**:
  - *What was wrong:* the comment said the old label was "impl", but it was "phase", because the trailing plan identifier was being read. The case the plan describes, a plain TASK sub-issue reached from the arch plan, had no test (N3).
  - *Fix:* the comment now says "phase", and `TestTreeArchRootTaskSubIssueKind` covers the arch-rooted case.
