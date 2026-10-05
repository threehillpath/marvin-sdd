# Phase 6: Checked issue create and issue edit

**Status**: Completed
**Phase issue**: [#138](https://github.com/threehillpath/marvin-sdd/issues/138)
**Pull request**: [#148](https://github.com/threehillpath/marvin-sdd/pull/148)
**Implementation plan**: [#132](https://github.com/threehillpath/marvin-sdd/issues/132)

## Objective

Phase 6 of PLAN-00112: add schema-checked issue creation and editing. `marvin issue create --template` and the new `marvin issue edit <n> --template` run the conformance check first, and make no mutating GitHub call when it finds an error.

## Scope

**In scope (impl plan Component 6):**
- **`issue create`** gains `--template <type>` and `--draft <file.yml>`:
  - with `--template` and `--draft`, the title comes from the draft. `--title` is optional, and if given it must equal the draft's title (otherwise exit 1),
  - with `--template` and `--body-file`, `--title` is required,
  - with `--template` and the inline `--body` flag: usage error (exit 1), telling the caller to use `--draft` or `--body-file`,
  - without `--template`: behavior is exactly as today, and `--draft` is rejected (exit 1).
- **`marvin issue edit <n> --template <type> (--draft <file.yml> | --body-file <file.md>)`:**
  - `--template` is required, and there is no inline `--body`,
  - with a draft, the edit sets the rendered body and the title,
  - with `--body-file`, it sets the unchanged body only.
- **`gh.Client.IssueEdit(ctx, repo, number, title, body)`:** runs `gh issue edit <n> --repo R --body B`, plus `--title T` when the title is non-empty. `issue.Edit` wraps it.
- **Order of operations:**
  1. load config (exit 2 if missing),
  2. resolve the schema,
  3. load the input,
  4. run the check,
  5. only then make the mutating `gh` call.
- On any error: zero mutating calls, exit 3, findings on stderr, and empty stdout.
- **`issue edit --body-file` title read:** uses `IssueRef(ctx, cfg.Repo, n)`, which passes `--repo`. `IssueJSON` is not used.
- Warnings and the `schema:` line go to stderr. Stdout stays number then URL for create, and empty for edit.

**Out of scope:** skill changes (Phase 7).

## TDD Entry Point

A CLI test with `exectest.FakeRunner` and `withConfigFixture`: `issue create --template impl-phase --draft <missing-verification.yml> --label x` exits 3 with `len(fake.Calls) == 0`.

## Components

- `tool/internal/gh/client.go`: `IssueEdit`, with tests in `client_test.go` in the `TestIssueCreateArgs` style.
- `tool/internal/issue/issue.go`: `Edit`.
- `tool/internal/cli/handlers_integrations.go`: the create flags, `newIssueEditCmd`/`runIssueEdit`, and registration in `newIssueCmd`.

## Verification

```bash
cd tool && go test ./...
marvin issue create --template impl-phase --draft no-verification.yml --label plan:phase; echo $?   # exit 3, no issue created
marvin issue create --template impl-phase --title "[PLAN-00112-1] X" --body "x"; echo $?          # exit 1, use --draft or --body-file
```

## Success Criteria

- [ ] A bad draft on create exits 3 with zero `gh` calls (entry-point test).
- [ ] A conforming create draft makes exactly one `gh issue create` call, whose `--body` is the rendered markdown and whose `--title` is the draft's title.
- [ ] `issue edit 7 --template impl-phase --draft <ok.yml>` makes exactly one call: `gh issue edit 7 --repo <cfg repo> --body <rendered> --title <draft title>`.
- [ ] `issue edit 7 --template impl-phase --body-file <ok.md>` makes a read `gh issue view 7 --repo <cfg repo> …`, then exactly one `gh issue edit 7 --repo <cfg repo> --body <unchanged>` with no `--title`.
- [ ] A bad draft on edit exits 3 with zero calls.
- [ ] `--template` with inline `--body` exits 1 with zero calls.
- [ ] Create without `--template` behaves exactly as before (existing `issue_create_test.go` passes unchanged).
- [ ] `go test ./...` passes.

---

## Implementation

This phase wires the schema conformance check into issue creation and adds issue editing. `marvin issue create --template <type>` (with `--draft` or `--body-file`) now runs the check first. Any error finding exits 3 with the findings on stderr, empty stdout and zero `gh` calls. A conforming draft makes exactly one `gh issue create` with the draft title and the rendered markdown body. The new `marvin issue edit <n> --template <type> (--draft | --body-file)` command sets the rendered body and title from a draft; with `--body-file` it reads the current title with `gh issue view --repo` and sets the unchanged body only. Supporting code is `gh.Client.IssueEdit` and `issue.Edit`/`issue.Title`.

Every input file is read once (`readInputFile`) and checked with `checkDraftBytes` (which renders, so the goldmark backstop runs) or `checkBodyBytes`, the same checks `validate` and `render` use. The checked bytes are the ones sent. Template names are limited to the fixed four built-in types before any override path is opened, and an override's declared `type:` must match its file name. Explicitly passed flags are detected with `Flags().Changed`, so an empty value is an error, not an absent flag.

Validation was reworked during review to collect all usage problems into one exit-1 error, with the prefixes `issue create:`, `issue edit:`, `template validate:` and `template render:`. The same behaviour applies to `validate` and `render`. A missing config is the deliberate exception: it exits 2 alone before anything else is collected. Cobra-level argument errors were turned into reported problems with advice, and a stderr note says when the title-dependent checks did not run. Two phase 5 regression tests were carried over: the `RunWithStreams` exit-3 test and the `render --draft` goldmark backstop test.

### Test Plan / Verification

- [x] Entry point: `issue create --template impl-phase --draft <missing-verification.yml> --label x` exits 3 with `len(fake.Calls) == 0`
- [x] Conforming create draft: exactly one `gh issue create`, `--title` the draft's title, `--body` the rendered markdown
- [x] `issue edit 7 --template impl-phase --draft` makes one `gh issue edit 7 --repo R --body B --title T`
- [x] `issue edit 7 --template impl-phase --body-file` makes `gh issue view 7 --repo R …`, then one edit with no `--title`
- [x] A bad draft on edit exits 3 with zero calls; `--template` with an inline `--body` exits 1 with zero calls
- [x] Existing `issue_create_test.go` passes unchanged
- [x] `go test -count=1 ./...`, `go vet ./...` and an empty `gofmt -l`

### Decisions

- **`--template` / `<type>` is a fixed set of four built-in types** (arch-plan, impl-phase, impl-plan, quick-task) — the names carry business-rule weight, so an unknown, empty or path-like name is an error even if an override file exists. Overriding one of the four with `.claude/plan-workflow-templates/<type>.yml` still works. User decision during review.
- **A missing config is reported alone, with exit 2** — it is an environment problem that blocks everything, so it is the one deliberate exception to the report-all rule below. User decision, 2026-10-04.
- **Report every problem in one pass** — flag and usage validation on `create`, `edit`, `validate` and `render` collects all problems into one exit-1 error instead of stopping at the first. A bad draft plus a usage problem exits 1 with the findings still printed; a bad draft alone exits 3. The invocation being wrong takes precedence over the input being non-conforming. User rule, 2026-10-04.
- **A missing title on `create --template --body-file` is reported once** — the duplicate title finding is dropped, the body is checked without a title, and a stderr note says the title-dependent checks did not run. User decision.
- **On `create --template --draft`, `--title` is optional but must equal the draft's title if given** (an explicit empty one included); `validate --draft` keeps the stricter rule that any `--title` is a usage error.
- **Every input file is read once, and the bytes checked are the bytes sent** — pipes, FIFOs and `/dev/stdin` are consumed by the first read.
- **An override's declared `type:` must match its file name**, otherwise exit 1 naming the file, the declared type and the expected type.
- **`edit --body-file` reads the current title with `gh issue view --repo` and sends only the unchanged body** — the view call runs only when no other problem exists. `edit --draft` sets body and title.

### Scope Changes

- **added:** extra flag rules beyond the spec — `--draft` together with `--body-file` is an error on `create`, `edit` and `validate`, and the `--title` requirement on `create --template --body-file` is skipped when `--draft` is given — needed to make mode conflicts and title rules unambiguous.
- **added:** cobra-level argument and flag errors on `issue edit` (argument count, `--body`, `--title`, `--label`) and a stray positional on `issue create` became reported problems with advice — they failed before problems were collected, hiding the rest.
- **added:** help text updates (`--template` lists the valid types, `--body` says it is not for `--template`, `edit`'s Short says `--draft` also sets the title, the `issue` group Short no longer says Read) — review found the text stale or misleading.
- **added:** two phase 5 regression tests, `RunWithStreams` exit 3 and the `render --draft` goldmark backstop — gaps carried over from phase 5; both passed immediately and were committed as guards.

### Deferred / Watch Items

- **`--json` output on `issue edit`** — track in Phase 7 (#139), only if the skills need machine output.
- **Skill prose for the new commands** — track in Phase 7 (#139).
- **`--body-file -` for stdin is not supported** (`gh` accepts it, marvin does not; `/dev/stdin` works), **issue numbers are parsed with a bare `Atoi` across marvin** (`issue edit 0` reaches `gh`, which fails loudly), and **`CheckMarkdown`/`Render` run the goldmark verification only when `Check` has no errors** (phase 4 design, so a body with both kinds of problem takes two runs) — track in #150.
- **An explicit template-file flag** — track in #149.
- **`TestNamesDeriveTaskJSON` depends on a plan-workflow config in a parent directory** — track in #147.

### Corrections

- **Explicitly empty `--template`, `--draft` or `--body-file`** (review B1): `create --template "" --title T --body-file bad.md` tested the value, so it created an unchecked issue and exited 0 → passed flags are detected with `Flags().Changed`. An empty `--template` is an error listing the valid types, and an empty `--draft` or `--body-file` is an error naming the flag, on `create`, `edit`, `validate` and `render`.
- **Template name resolution** (`resolveSchema`): it accepted any name with a project override file, including path-like names such as `../x` → the name must be one of the four built-in types, checked before any override path is built or opened, and the error lists the types.
- **Override type declaration**: `impl-phase.yml` with `type: custom-thing` was accepted and reported as `schema: custom-thing` → a declared type that differs from the file name is an exit-1 error naming the file, the declared type and the expected type, on all four commands.
- **Usage problems reported one at a time**: they returned on the first hit, and a bad draft with a mismatched `--title` hid the mismatch → all usage problems are collected into one exit-1 error (a bullet per problem under a header; a lone problem stays a bare message), loaded input is checked anyway with its findings on stderr, and a loaded draft's title is compared with `--title` whether or not it conforms.
- **Input files read twice**: a pipe, FIFO or `/dev/stdin` body was consumed by the first read, so a conforming body was reported as missing everything (a regression from an earlier fix in this PR), and `create` could send bytes it had not checked → every input is read once (`readInputFile`) and the checked bytes are the sent bytes, tested with once-readable FIFOs and probed on a built binary with `/dev/stdin` and `<(...)`. The FIFO tests build only on platforms that support them.
- **Cobra-level errors on `issue edit` and `issue create`**: a wrong argument count, `--body`, `--title` or `--label` on `edit`, and a stray positional on `create`, failed before any problem was collected → each is a reported problem with advice (needs exactly one `<issue-number>`; there is no inline `--body`: use `--draft` or `--body-file`; `--draft` sets the title).
- **`create --draft` without `--template`**: it was reported alongside unrelated problems → it reports only that problem, naming the valid types.
- **Title-dependent checks when the title is unavailable**: a missing title produced a duplicate title finding and the skipped checks were not announced → the duplicate is dropped, the body is checked without a title, and a stderr note says the title-dependent checks did not run. `edit` does the same when it cannot read the title.
- **`edit --body-file` ordering**: the title lookup ran before the body file was read and checked → the body file is read first and checked with other problems; the non-mutating `gh issue view` runs only when no other problem exists.
- **`validate` with no loadable schema**: inputs were not read when the schema failed to load, so input problems were hidden → `validate` reads its inputs even without a loadable schema and reports all problems.
- **`validate --title` with `--body-file`**: a false `--title` problem was reported → it is a problem only when the input is a draft.
- **`render --draft` with an empty value**: it was not reported together with the mode conflict → `render` reports both.
- **Help text and doc comments**: they did not describe `--template` or the new rules → updated across `create`, `edit` and the `issue` group.
- **Test gaps found in review**: coverage missed goldmark backstop cases for every create and edit input path, exact `gh` argv and stream assertions, exit codes and missing config, and some tests did not plant the path-like override files → added `<div>`-in-Scope backstop tests for every input path, exact-argv and stdout/stderr assertions, exit-code and missing-config tests, and planted the override files where the path-like names resolve.
