# Phase 7: Skill adoption and docs

**Status**: Completed
**Phase issue**: [#139](https://github.com/threehillpath/marvin-sdd/issues/139)
**Pull request**: [#151](https://github.com/threehillpath/marvin-sdd/pull/151)
**Implementation plan**: [#132](https://github.com/threehillpath/marvin-sdd/issues/132)

## Objective

Phase 7 of PLAN-00112: switch the four drafting skills to YAML drafts. Every create and edit goes through `--template`, and phase-split validates every phase draft before creating any issue. The docs are updated for exit code 3, the draft format, the required schema fields, and the `task` kind.

## Scope

**In scope (impl plan Components 7 and 8):**
- **`arch-plan`, `impl-plan`, `phase-split`, `quick-task`:**
  - `allowed-tools` adds `Write`.
  - **Drafting:** `marvin template render <type> --skeleton` (now a YAML draft), then `Write` the filled draft to a scratch `.yml` file.
  - **Format rules:**
    - section content is always a `|` block, never `>` or inline,
    - the title and every metadata value are always double-quoted, with inner `"` written as `\"` and `\` as `\\`,
    - each key appears once (repeatable sections are one list),
    - no tabs,
    - no `## ` lines in content (use `###`).
  - Remove the "has no `Write` tool" text and the body heredoc.
  - **Review:** show the rendered markdown (`marvin template render <type> --draft <file>`), not the YAML.
  - **Create:** `marvin issue create --template <type> --draft <file> --label …`.
  - **On exit 3:** fix the draft per the findings and retry, at most 3 times, then show the findings to the user. Exit 1 or 2 is surfaced and never retried.
- **phase-split:**
  - Replace the line-92 rule ("compose … immediately before creating that issue") with: each phase's title and body are written together in one draft file; write and validate every draft, then create them in order.
  - Run `marvin template validate impl-phase --draft` on every phase draft before the first `issue create`.
  - If a create still fails, report which phase issues were created.
  - The step-3b fix-up becomes `marvin issue edit <n> --template impl-phase --draft <corrected-draft>`.
- Heredocs not used for issue bodies (`gh issue comment`, `gh pr comment`, `gh pr edit`) are left alone.
- **Docs:**
  - CLAUDE.md: the exit-code contract adds `3`, the `clierr/` structure line becomes `(0 / 1 / 2 / 3)`, the `template/` line mentions validation, and the config paragraph notes that validation follows the same precedence.
  - `skills/SHARED/CONFIG.md` "Plan Template Resolution": covers render, validate, `issue create --template` and `issue edit --template`, the YAML draft format, and the required `title_prefix` / `named` fields with an override migration note.
  - **`issue tree` kind lists:** add `task` in start-impl (~31), review-impl (~39), phase-split (~54) and CONFIG.md (~123).

**Out of scope:** any Go code change.

## TDD Entry Point

None. This is a skill-prose and docs phase, with no Go logic. It is covered by review-phase's structural pre-check (step 2b), which runs because this section says None.

## Components

- `skills/arch-plan/SKILL.md`, `skills/impl-plan/SKILL.md`, `skills/phase-split/SKILL.md`, `skills/quick-task/SKILL.md`
- `skills/start-impl/SKILL.md`, `skills/review-impl/SKILL.md` (kind list only)
- `CLAUDE.md`, `skills/SHARED/CONFIG.md`

## Verification

```bash
grep -n "allowed-tools" skills/{arch-plan,impl-plan,phase-split,quick-task}/SKILL.md          # all include Write
grep -n "marvin issue create" skills/{arch-plan,impl-plan,phase-split,quick-task}/SKILL.md    # every line has --template
grep -n "marvin template validate impl-phase" skills/phase-split/SKILL.md                     # present
grep -n "gh issue edit" skills/phase-split/SKILL.md                                           # no matches
grep -rn "has no \`Write\` tool" skills/{arch-plan,impl-plan,phase-split,quick-task}/          # no matches
grep -n "immediately before creating that issue" skills/phase-split/SKILL.md                  # no matches
grep -rn "one of \`arch\`, \`impl\`, \`phase\`\." skills/                                      # no matches (task added)
grep -n "3" CLAUDE.md | grep -i "exit\|clierr"                                                # exit code 3 documented
```

## Success Criteria

- [ ] All four drafting skills list `Write` in `allowed-tools`.
- [ ] Every `marvin issue create` line in the four skills passes `--template`, and none uses `--body-file` with a heredoc body.
- [ ] The drafting rules state the `|`, quoting/escaping, one-key, no-tabs and no-`## ` rules.
- [ ] The review step shows rendered markdown via `render --draft`.
- [ ] The exit-3 retry rule (max 3, then surface) is present, and exit 1 or 2 is never retried.
- [ ] phase-split validates every phase draft before its first create, no longer says "immediately before creating that issue", reports already-created issues on a failed create, and its fix-up uses `marvin issue edit --template impl-phase`.
- [ ] CLAUDE.md documents exit code 3 in the contract line and the `clierr/` line. CONFIG.md documents the draft format, the required schema fields, the migration note and the validation precedence.
- [ ] All four `issue tree` kind lists include `task`.
- [ ] Every Verification grep returns the expected result.

---

## Implementation

This phase moved the four drafting skills (`arch-plan`, `impl-plan`, `phase-split`, `quick-task`) onto the checked-create flow built in earlier PLAN-00112 phases, in skill prose and docs only (no Go changes; `tool/` is identical to the trunk). Each skill now adds `Write` to its `allowed-tools`, drafts a YAML file starting from `marvin template render`'s `--skeleton` and `--guidance` output, shows the rendered markdown (`render --draft`, pasted into the reply), and creates the issue with `marvin issue create --template <type> --draft`. The body heredocs and the "has no `Write` tool" wording are gone. Drafts live under `<project-root>/.claude/cache/<plan or task>/` (`plan-XXXXX/arch-draft.yml`, `impl-draft.yml`, `phase-N-draft.yml`, `task-XXXXX/task-draft.yml`, plus `impl-<suffix>` variants for multi-impl tracks), keyed by the run's own identifier, read before overwrite and not deleted. Exit 3 is fixed and retried at most 3 times per round of user changes, with re-approval only when the fix changes content or title. Exit 1 and 2 are surfaced and never retried. Labels are built as one comma-joined string from only the parts that exist.

`phase-split` validates every phase draft with `marvin template validate impl-phase --draft` before the first create. If a create stops the run, it reports the phases already created and the steps that did not run. Title and body mismatches are repaired with `marvin issue edit <n> --template impl-phase --draft`. It shows rendered bodies only when a draft has warnings or the user asks. `quick-task`'s title no longer doubles `TASK`, and its Mode B cleanup runs `marvin findings clear <task>` to clear the task's draft cache. `task` was added to the four `issue tree` kind lists.

`CLAUDE.md` and `skills/SHARED/CONFIG.md` now document the draft format with a filled example that passes `marvin template validate`, and exit code 3. They also cover where the schema line and warnings go for each command, the required `title_prefix` and `named` schema fields with an override migration note, the allowed title prefixes, validation precedence, the fixed set of four types and the declared-type rule. Stale `CLAUDE.md` lines (`issue create`/`edit`, `template validate`, `validate`'s stdout, and the `issue/` and `cli/` package descriptions) were updated. The work was reviewed through two comment reviews (B1, N1-N22 and drift C1-C3; then N1-N17 and one follow-up). Each finding was fixed in its own commit, and the Go-side gaps were filed as #152.

### Test Plan / Verification

- [x] `grep -n "allowed-tools" skills/{arch-plan,impl-plan,phase-split,quick-task}/SKILL.md`: all four list `Write`
- [x] `grep -n "marvin issue create" …`: four lines, every one has `--template`
- [x] `grep -n "marvin template validate impl-phase" skills/phase-split/SKILL.md`: present
- [x] `grep -n "gh issue edit" skills/phase-split/SKILL.md`: no matches
- [x] The "has no `Write` tool" and "immediately before creating that issue" greps: no matches
- [x] The old `issue tree` kind-list grep: no matches (`task` added)
- [x] `grep -n "3" CLAUDE.md | grep -i "exit\|clierr"`: line 29 (`0 / 1 / 2 / 3`) and line 83 (the contract)
- [x] `grep -rn "/tmp/"` on the four skills: no matches (drafts are under `.claude/cache`)
- [x] `go test -count=1 ./...`, `go vet ./...` and an empty `gofmt -l tool`; no file under `tool/` changed
- [x] `git diff --check` clean; `tool/` identical to the trunk
- [x] Probes against a binary built from the worktree with a logging fake `gh` and a scratch config: a draft written with `Write` to `.claude/cache/plan-00007/impl-draft.yml` (parent directory created by `Write`) validates, renders, creates (exit 0, stdout number then URL, `--label` split into separate flags with no empty entry) and edits; a draft with an empty required section exits 3 with no `gh` call; `--body` with `--draft` exits 1; no config exits 2; the filled example in CONFIG.md passes `validate`

### Decisions

- **Draft files live under the project's `.claude/cache/<key>/`** (plan-keyed, task-keyed for `quick-task`), are read before overwrite, and are not deleted after create — `/tmp/<skill>-draft.yml` was shared by every run on the machine, so an overlapping run could swap in another plan's title and body. `wrap-phase` already clears the plan's cache. User decision, 2026-10-04; a config override for the location is a possible future setting and is deliberately not in the skills.
- **After an exit 3 at create, re-approval is required only if the fix changes what the draft says**, not how it is written — content or title changes need the user's approval, while quoting, escaping, block style and heading depth retry directly. `phase-split` has no review step and keeps the plain 3-attempt loop. User decision.
- **`phase-split` shows rendered bodies only when a draft produced warnings or the user asks**, not by default — it never showed bodies, and the user approves the phase list in step 2. User decision, 2026-10-04.
- **Draft retries are capped at 3 fix-and-render attempts per round of user changes** — exit 3 is retried, exit 1 and 2 are surfaced and never retried. The original "within the retry limit in the create step" wording was ambiguous.
- **Draft YAML is fixed by rewriting with `Write`, not `Edit`, and `tdd_entry_point` is always kept**, written as `None.` plus the reason when there is none — `--guidance` says the key may be omitted, but an omitted key passes silently and skips `review-phase`'s skill-prose pre-check.

### Scope Changes

- **added:** CONFIG.md docs beyond the issue's list — a filled example draft, an exit-code table, a per-command flag summary and the title-prefix rule — docs-only, each checked against a binary built from the branch. The example is extracted from the file and passes `marvin template validate`.
- **added:** `quick-task` cache cleanup — Mode B runs `marvin findings clear <task>` — because the new draft location introduced a task cache directory that nothing cleared. Verified to exit 0, including when the directory is already gone.
- **added:** a multi-impl suffix convention (`impl-<suffix>-draft.yml`, `[PLAN-XXXXX-<suffix>]` and `[PLAN-XXXXX-<suffix>-N]` titles) — review found that multi-impl tracks needed distinct draft files and titles, and a suffix title validates.
- **added:** fixes to stale `CLAUDE.md` lines outside the issue's list (the `issue/` and `cli/` package descriptions, marvin's operations, and `validate`'s stdout in the output contract) — they were inaccurate after phases 5 and 6.

### Deferred / Watch Items

- **Go fixes found in review** — `--guidance` completeness (including exempting code fences from the `\#` rule), the loader's single-quote advice, a loader check that `title_prefix` matches the type, trailing-comma label handling, an unknown subcommand exiting 0, and exit-2 wording for a malformed config — track in #152.
- **Explicit template-file flag** — track in #149.
- **Input-handling gaps** (`--body-file -`, bare `Atoi` on issue numbers, goldmark only when `Check` is clean) — track in #150.
- **Unverified: whether Claude Code prompts for `Write` under `.claude/`** in the permission modes users run — watch item with no ticket; if a prompt appears, move the draft location or make it a config setting.
- **Pre-existing skill-prose gaps not in this diff** — `impl-plan` omits the source issue's type label that `LABELS.md` carries; CONFIG.md still says marvin handles "all board and issue read operations"; `phase-split` handles only exit 2 from `link-parent`; no skill says what to put in the Author, Status and Date metadata values — track in #153.

### Corrections

- **`quick-task` title format** (N1): the title doubled the `TASK` prefix → it no longer does.
- **Exit-code handling for skeleton, guidance and render** (N5): exit 1 was not handled, and exit 2 was handled where it cannot occur → exit 1 is handled, and exit 2 is removed where it is impossible.
- **Draft-fix mechanism and ordering** (N8, N10, N8 of the re-review): the prose fixed drafts with `Edit`, put the `Write` instruction mid-step, and step 3b did not name the file to correct → drafts are fixed by rewriting with `Write`, the instruction sits at the end of the drafting step, and step 3b names the draft file.
- **Phase with no TDD entry point** (N9, N7 of the re-review): `--guidance` says the key may be omitted, but an omitted key passes silently and skips review-phase's pre-check → keep the key and write `None.` plus the reason.
- **Cross-references and phrasing** (N11, N12, N17 of the re-review): reference paths were inconsistent, "each naming a draft line" was imprecise, and a "Rules." fragment stood in for a heading → references use `../SHARED/CONFIG.md`, the wording is precise, and the fragment is a proper heading.
- **`phase-split` failure reporting and edit output** (N14, N15): a failed create gave no report of what was already created, and `issue edit`'s success output was undocumented → `phase-split` reports created phases and the steps that did not run, and the docs state that `issue edit` prints nothing on success.
- **Metadata, names and `--guidance` coverage** (N16, N17, N6 of the re-review): prose called `--guidance` the complete rule list, but it omits the loader's key-once, no-tab, no-tag/anchor/alias and metadata-format rules, and component `name:` values were unquoted → the skills and CONFIG.md say what `--guidance` covers and list the loader rules inline, state the metadata value formats, double-quote `name:` values, and forbid both `#` and `##` heading lines with the leading-`#` escape documented.
- **Create flags and CONFIG.md reference material** (N18, N19, N20, N21, N22, stale lines): the `create --title` rule and `--body-file` mode, where the schema line and warnings print, skeleton omissions, title-prefix rules and schema locations were undocumented, and several `CLAUDE.md` lines were stale → documented, with a filled example added and the stale lines updated.
- **Draft location** (post-review user decision): `/tmp/<skill>-draft.yml` was shared across runs, so another plan's title and body could be swapped in → drafts moved to `<project-root>/.claude/cache/<key>/`, keyed by plan or task, in every `Write`, render, validate, create and edit line.
- **Re-approval after an exit-3 fix at create**: the prose created the fixed draft without showing the user → re-approval is required only when the fix changes content or title.
- **Retry wording and the step-3b fix-up**: "within the retry limit in the create step" was ambiguous, and step 3b referred back to create's exit handling → the cap is 3 attempts per round of user changes, and the exit codes (3 retried, 1 and 2 never) are restated inline.
- **Labels placeholder**: the `<domain-labels>,<type-label>` placeholders made a trailing comma likely, which marvin passes to `gh` as an empty `--label` → the skills build one `<labels>` string from only the parts that exist, with an example and no spaces around commas.
- **CONFIG.md `title_prefix` and migration notes** (re-review N1, N2, N14, N16): a wrong-type `title_prefix` loads but fails every title with exit 3, not exit 1; the migration note pointed at `--guidance`, which exits 1 inside the broken project; the retry summary lacked the per-round cap; and the `--skeleton`/`--guidance` output was misdescribed → documented exit 3, pointed the note at the schemas directory or a run outside the project, added the per-round cap and the re-approval rule, and stated that `--skeleton` prints no schema line while `--guidance` prints `schema: <type>` and never an override origin.
- **Escape and plan-number rules** (re-review N3, N4): the `\#` escape was stated as universal but does not apply inside a code fence, and the `arch-plan` Plan Number wrongly kept the `-ARCH` suffix → the fence exception is documented (verified against the binary) and the Plan Number drops `-ARCH`.
- **Metadata validation rules** (re-review N5): the prose did not say that every metadata value must be non-empty, or that a plan named in a `#<n>` reference must be the title's → both are stated, and `quick-task` is noted as having no plan cross-check.
- **Path and key definitions** (re-review N9, N10): `<project-root>` and `arch-plan`'s `<plan>` were undefined, and multi-impl tracks had no distinct draft names → `<project-root>` is the main checkout's root, `<plan>` is `plan-` plus five digits, and multi-impl tracks use `impl-<suffix>-draft.yml`.
- **`phase-split` visibility and review steps** (re-review N12, N13): users were not told bodies were available, validate warnings were not shown, and the review steps had several approval points → `phase-split` says bodies are available and shows validate warnings, and each review step has one approval point.
- **`CLAUDE.md` `issue/` and `cli/` descriptions** (re-review N15): the lines implied `issue/` ran the create/edit check → they now say the check runs in `cli/` and `issue/` has plain wrappers.
- **`quick-task` cache cleanup**: the `.claude/cache/task-XXXXX/` directory introduced by the new draft location was never cleared → Mode B cleanup runs `marvin findings clear <task>`.
