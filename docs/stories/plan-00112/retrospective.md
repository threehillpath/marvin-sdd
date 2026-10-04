## Decisions

The central theme of PLAN-00112 was making plan issues impossible to create or edit without passing a schema check. Most of the design choices came out of an unusually long series of review rounds, and several were made by the user directly.

**Title classification.** In `[PLAN-00112-1]` the team reused `names.Kind` for classifying titles rather than adding a second enum. `Classify` looks only at the leading bracket and hands plan tokens to `PlanIdent`. That keeps the classifier and `PlanIdent` from disagreeing about `found:` and the plan fields, and task titles omit plan fields even when a later bracket looks like a plan. Membership in the issue tree is still decided by `PlanIdent`. A single `names.Kind.String()` replaced two separate label mappings that fell back to different labels for an unknown kind.

**Drafts treated as pure data.** This principle drove most of `[PLAN-00112-2]`, `[PLAN-00112-3]` and `[PLAN-00112-4]`. Section content must never be able to change the document's structure. Several choices followed:
- In `[PLAN-00112-2]`, an unclosed code fence became a `section:<id>` error, because it would render every later section as code and hide those sections from the parser.
- In `[PLAN-00112-3]`, the author chose to ban rather than tolerate. Drafts accept no YAML document markers, no YAML comments, and no tags, anchors or aliases. Each of these had been silently dropping text.
- Also in `[PLAN-00112-3]`, a conservative setext rule rejects any `-`/`=` underline directly after a non-blank line, accepting a small formatting cost, and raw HTML is banned instead of tracked. Two rounds of hand-written state machines for comments, `<details>` and raw blocks kept hiding later lines from other guards.
- After round 3 still found gaps in the regex scanner (unclosed `<?php`, CDATA, `DOCTYPE`, list-item fences), the author added a backstop. `Render` now parses its own output with goldmark and GFM and refuses if the headings differ or an HTML block appears.
- Because YAML comments are errors, `--skeleton` became comment-free and guidance moved to a separate `--guidance` flag.
- `[PLAN-00112-4]` carried the same guards over to the markdown path, covering the preamble and content under unknown headings. It also restricted `**Key:**` metadata lines to positions where GitHub would show them as metadata. It deliberately did not add a new setext rule, since `scanContent` already covers it.

**One scanner, one load path.** In `[PLAN-00112-2]`, `FindH2Lines` was exported so the checker and the parser could not drift on what counts as a heading, and `Render`, `Skeleton` and `Check` all went through a validated `LoadSchema`. `Check` also now reports an unset `Source` or a schema not built by `LoadSchema`. That follows the no-silent-failures rule, since zero values had been silently selecting YAML fixes or the arch kind.

**CLI contract.** In `[PLAN-00112-5]`, exit code 3 (`clierr.NonConforming`) was introduced for any error finding, including loader findings. Warnings alone exit 0, and unknown types, unreadable input and malformed schemas stay exit 1. Origin labels became `built-in` or `project override: <path>`. `Result.Sorted()` became the single ordering for text and `--json`, and `template.DefaultSchemaNames()` became the single source for type lists and help text.

**User-directed rules in `[PLAN-00112-6]`.**
- The template type is a fixed set of four (arch-plan, impl-phase, impl-plan, quick-task), because the names carry business-rule weight. An unknown, empty or path-like name is an error even if an override file exists. Overriding one of the four still works.
- Flag and usage validation reports every problem in one pass, so a bad draft plus a usage problem exits 1 with findings still printed. A wrong invocation takes precedence over non-conforming input.
- A missing config is the deliberate exception and is reported alone with exit 2, since it blocks everything.
- Input files are read exactly once, so the bytes checked are the bytes sent.
- An override's declared `type:` must match its file name.
- On `create --template --draft`, `--title` is optional but must equal the draft's title if given.

**Skill adoption in `[PLAN-00112-7]`.**
- Draft files moved from the shared `/tmp/<skill>-draft.yml` to `.claude/cache/<key>/`, keyed by plan or task, because an overlapping run could swap in another plan's title and body. A config override for the location was deliberately left out.
- After an exit 3 at create, re-approval is needed only when a fix changes what the draft says, not how it is written.
- `phase-split` shows rendered bodies only on warnings or request.
- Retries are capped at 3 per round of user changes; exit 1 and 2 are never retried.
- Drafts are fixed by rewriting with `Write`, and `tdd_entry_point` is always kept (as `None.` plus a reason), because omitting it would silently skip `review-phase`'s skill-prose pre-check.

## Scope changes

Scope grew mainly in response to review. In `[PLAN-00112-1]` it grew by the `Kind.String()` method, an arch-rooted TASK sub-issue regression test, and gofmt fixes to two files already unformatted on main.

In `[PLAN-00112-2]`, `FindH2Lines` was exported for reuse and unclosed fences were promoted to errors. `impl-plan.yml` also gained `named` flags to satisfy the new required schema field.

`[PLAN-00112-3]` made the biggest changes. It added the `--guidance` flag, with ten reworded rules, and the goldmark v1.8.6 dependency. It extended the raw-HTML ban to metadata values, entry names and the title. It removed `--sections`, `--meta` and the dead `template.KV` type, as the spec's JSON-to-YAML replacement required. The raw-HTML state machines and setext exceptions were also deleted. The `--skeleton` change altered the original SC5 criterion.

`[PLAN-00112-4]` extended the content guards to the preamble and to unknown headings. The orchestrator asked for the unknown-heading guards, review required the preamble ones, and the drift audit confirmed both were in scope.

`[PLAN-00112-5]` added code 3 to the documented exit-code contract and introduced the exported helpers `DefaultSchemaNames()` and `Sorted()`. The drift audit judged both in scope.

`[PLAN-00112-6]` added extra flag rules, such as rejecting `--draft` with `--body-file`. It turned cobra-level argument errors on `issue edit` and a stray positional on `issue create` into collected problems, refreshed stale help text, and added two regression tests carried over from phase 5.

`[PLAN-00112-7]` added documentation beyond the issue's list: a filled example draft that passes `marvin template validate`, an exit-code table, flag summaries and the title-prefix rule. It added `quick-task` cache cleanup via `marvin findings clear`, a multi-impl suffix convention (`impl-<suffix>-draft.yml`), and fixes to stale `CLAUDE.md` lines left inaccurate after phases 5 and 6.

## Deferred / watch items

Section content that could still break document structure was deferred in `[PLAN-00112-2]` to #135 and #136. This covers setext headings, unclosed HTML comments and blocks, metadata-lookalike lines, and content that forms a heading only after rendering. `[PLAN-00112-2]` also logged three review nits in #135. `[PLAN-00112-3]` accepted two conservative trade-offs, to be revisited only if they cause problems: a lone inline HTML tag on its own line is refused, and a `---` under a list item or quote needs a blank line above it.

Wiring work was pushed forward in sequence. `[PLAN-00112-3]` and `[PLAN-00112-4]` left `template validate` running the rendered-body verification, `render --draft`, and the CLI hookup of `ParseMarkdown`/`CheckMarkdown` to Phase 5 (#137). `[PLAN-00112-3]` also deferred updating skills that still call the removed flags to Phase 7 (#139). `[PLAN-00112-5]` deferred a dedicated `render --draft` backstop test and a `RunWithStreams` exit-3 test to Phase 6 (#138), which then added both.

The following items remain open:
- `[PLAN-00112-6]` deferred `--json` output on `issue edit` and skill prose, both to #139. It also deferred items in #150: `--body-file -` is unsupported (`/dev/stdin` works), issue numbers go through a bare `Atoi`, and goldmark verification runs only when `Check` has no errors (so a body with both kinds of problem takes two runs).
- An explicit template-file flag is tracked in #149.
- `TestNamesDeriveTaskJSON` fails when `tool/` is built outside the repo because it needs a plan-workflow config in a parent directory. It is tracked as standalone bug #147, raised in `[PLAN-00112-5]` and again in `[PLAN-00112-6]`.
- `[PLAN-00112-7]` deferred Go fixes found in review to #152: `--guidance` completeness, single-quote advice, a `title_prefix`-matches-type loader check, trailing-comma label handling, an unknown subcommand exiting 0, and exit-2 wording for a malformed config.
- `[PLAN-00112-7]` also deferred pre-existing skill-prose gaps to #153.
- One watch item has no ticket: it is unverified whether Claude Code prompts for `Write` under `.claude/` in the permission modes users run. If a prompt appears, the draft location should move or become a config setting.

## Corrections

Review rounds produced a long correction list. Grouped by theme, most of it comes down to a few recurring failure modes.

**Silent loss of input.** This was the dominant theme, and every fix made the loader or parser refuse rather than quietly drop content. `[PLAN-00112-3]` fixed document markers, YAML comments that were dropped as head, line or foot comments, and tags, anchors and aliases that stripped leading text. It also fixed inline-code pairing that did not follow CommonMark, a lone CR left after CRLF normalization, and unclosed raw-HTML constructs that hid later lines. `[PLAN-00112-4]` fixed a repeated `**Key:**` line where the last value silently won, metadata-shaped lines that GitHub does not show as metadata, and a preamble that skipped the content guards. It also fixed a non-breaking space being counted as blank.

**Precise, correctly located error messages.** A large share of corrections concerned findings that blamed the wrong line or gave the wrong fix. Examples:
- A valid quoted key was blamed, and an unquoted-title fix was offered for a title that was already quoted (`[PLAN-00112-3]`).
- The missing-title fix on the markdown path told the agent to set a draft key that did not exist (`[PLAN-00112-2]`).
- The missing-`type` fix pointed back to the wrong title kind (`[PLAN-00112-2]`).
- Markdown-location wording read as garbled imperatives, and a rejected run of metadata lines produced conflicting errors (`[PLAN-00112-4]`).
- Raw HTML in a metadata value was reported twice (`[PLAN-00112-4]`).
- Undefined-alias and empty-heading findings had no usable location (`[PLAN-00112-3]`).

**Input handling and single reads.** `[PLAN-00112-6]` found that explicitly empty `--template`, `--draft` or `--body-file` values were tested by value, so `create --template ""` created an unchecked issue and exited 0. The fix was detection with `Flags().Changed`. The same bug recurred for `--title ""` in `[PLAN-00112-5]`. Reading input files twice consumed pipes, FIFOs and `/dev/stdin`, and `create` could send bytes it had not checked. The fix was a single `readInputFile` call, tested with once-readable FIFOs and probed on a built binary. In `[PLAN-00112-5]`, `--title` was silently dropped on `validate --draft`.

**Report-all-problems and ordering.** `[PLAN-00112-6]` corrected usage problems that returned on the first hit, cobra-level errors that fired before any problem was collected, and `validate` hiding input problems when no schema loaded. It also corrected duplicate title findings, undeclared skipped checks, and `edit --body-file` ordering. The body is now read and checked first, and the non-mutating `gh issue view` runs only when no other problem exists.

**The fixed template set.** In `[PLAN-00112-6]`, `resolveSchema` accepted any name with an override file, including `../x`. It now checks against the four built-ins before building any path. An override with a mismatched `type:` is rejected.

**Test quality.** Tests repeatedly proved too weak.
- In `[PLAN-00112-5]`, no test failed if `checkInput` used `Check` instead of `Render`, so two mutation-checked backstop tests were added. The override test discarded its exit code.
- In `[PLAN-00112-4]`, round-trip tests bypassed `CheckMarkdown`.
- In `[PLAN-00112-2]`, helpers discarded load errors and title tests checked keywords only.
- In `[PLAN-00112-6]`, coverage lacked exact `gh` argv, stream assertions and missing-config cases.
- `[PLAN-00112-4]` admitted a process slip: one commit landed while the preceding test was still failing. That was repaired, and the suite passes on every commit from the N5 fix onward.

**Single sources of truth.** The duplicate `builtInTypes` list, the two kind-label mappings, and the byte-for-byte duplicated test fixture were each collapsed to one definition.

**Documentation accuracy.** `[PLAN-00112-7]` carried most of this. Skill prose had a doubled `TASK` prefix in the `quick-task` title, wrong exit-code handling, and `Edit` used where `Write` was required. A `<domain-labels>,<type-label>` placeholder made a trailing comma likely, which marvin passes to `gh` as an empty `--label`. The `\#` escape was stated as universal but does not apply inside a code fence. The `arch-plan` Plan Number wrongly kept `-ARCH`. `--guidance` was described as the complete rule list although it omits the loader's own rules. `<project-root>` and `<plan>` were undefined. `[PLAN-00112-5]` fixed stale help and doc comments and amended its PR Notes after drift audit C1. `[PLAN-00112-1]` and `[PLAN-00112-6]` fixed misleading code comments and help text, and `[PLAN-00112-2]` fixed a misplaced `named` comment.

## Plan critique

The round-2 red-team critique returned a verdict of **revise**. Most first-pass findings (B2, B3, B4 and C1 through C10 except C4) were resolved. The build order was now acyclic, `issue edit` had success-path tests, the title read went through `IssueRef` with `--repo`, overrides failed loudly, and the skills got their own TDD-None phase. One blocking finding and several concerns remained.

**B1 (blocking): the loader still lost content.** Decoding into `yaml.Node` dropped yaml.v3's duplicate-key error, so a repeated `component:` key passed with err=nil and silently lost components. A ` #` in a plain-scalar section value or entry name (`objective: Deliver X for #112 so it works`) was cut at the `#`. A `>` folded section was silently reflowed. All three were verified in scratch runs against yaml.v3 v3.0.1, and they contradicted the plan's guarantee that a draft never passes after being silently truncated. The critique asked for three exit-3 loader errors: repeated keys, a comment on any scalar, and non-literal section content. It also asked for matching TDD cases. This finding shaped what `[PLAN-00112-3]` later did, though the author went further than the suggestion: all comments, markers, tags and anchors became errors, followed by the goldmark backstop.

**C1: loader findings and exit codes.** Loader errors had no Finding location, and Component 5's rule ("exit 3 only when the check ran") would have sent parse errors to exit 1, which breaks the retry loop. The critique proposed a `draft` location and exit 3 whenever the Result holds an error finding, which is how `[PLAN-00112-5]` defined exit 3.

**C2: unreliable yaml.v3 messages.** `did not find expected key` is not unique to an unquoted `[` title. An inner double quote in an already-quoted title produces the same message and so gets the wrong fix. Unknown escapes like `\d` are also mishandled. The critique asked for detection from the raw line and for mapped inner-quote and escape messages. `[PLAN-00112-3]`'s correction rounds on exactly these messages echo this concern.

**C3: the Arch zero value.** `names.Kind`'s zero value is Arch, so an unclassifiable title such as an unreplaced `[PLAN-XXXXX-ARCH]` placeholder could pass the arch-plan kind check. The critique asked for a `(names.Kind, bool)` classifier and an explicit error row. `[PLAN-00112-2]` later reported zero values explicitly for the same reason.

**C4: classifier versus `PlanIdent`.** It was unstated which function drives `found` and the plan fields when the anchored classifier and the unanchored `PlanIdent` disagree. The proposed rule, which `[PLAN-00112-1]` adopted, was that `PlanIdent` keeps driving `found` and the plan fields while `kind:` comes from the classifier.

**C5: phase-split ordering.** The "validate every draft first" rule contradicted phase-split's existing "compose each body immediately before creating it" instruction, so the line needed rewording.

**C6: stale docs.** Four docs still described the tree kinds as arch, impl and phase, and `CLAUDE.md` and `clierr.go` still said exit codes 0/1/2. The critique asked that these be assigned to components so drift audits would not flag them. This is the same documentation drift that `[PLAN-00112-5]` and `[PLAN-00112-7]` later had to repair.

**C7: orphaned test file and fixture timing.** `render_test.go`, with 8 tests on the old `Render`/`Skeleton` API, belonged to no component. The `overrideSchemaFixture` update was also scheduled too late for the new strict schema validation.
