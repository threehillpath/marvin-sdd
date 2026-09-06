---
name: wrap-phase
description: Capture decisions from a merged phase PR onto the impl plan, write a durable per-phase doc, close the phase issue, move it to Done, and clean up the worktree
argument-hint: <phase-issue-number> <impl-plan-issue-number>
allowed-tools: Bash, Read, Write, Agent
model: sonnet
---

Run after a phase PR has been merged. Reads the merged PR's history, classifies it into decisions / scope changes / deferred items / corrections, posts a structured wrap-up comment on the impl plan issue, writes a durable per-phase doc under `docs/stories/<plan>/` (spec + implementation summary + test results + the same classification), closes the phase issue, moves it to Done on the board, and removes the phase worktree.

**Before starting**: Read `.claude/plan-workflow-config.yml` for project configuration (repo, owner). Read `../SHARED/GLOSSARY.md` for naming, worktree, and status conventions.

## Arguments

- `$0` — Phase issue number (the `[PLAN-XXXXX-N]` issue)
- `$1` — Impl plan issue number (the parent `[PLAN-XXXXX]` issue)

## Steps

### 1. Locate the merged PR for this phase

```bash
gh issue view $0 --repo <repo> --json number,title,body,url
```

Extract the `[PLAN-XXXXX-N]` ident from the title, then locate the merged PR:

```bash
marvin parse title "<issue title>"
marvin pr find "[PLAN-XXXXX-N]" --state merged
```

Read the `plan_number:` line from `marvin parse` output (e.g. `plan-00042`) and use it as `<plan>` in subsequent steps. Also read the `plan:` line (e.g. `2`), `phase:` line (e.g. `3`), and `slug:` line (e.g. `role-assignment-ui`) — `plan`/`phase` are needed for `marvin names derive` below, and `slug` names the per-phase doc file rendered in step 5. The output from `marvin pr find` includes `found:`, `number:`, `url:`, and `state:` lines — capture `number:` and `url:` for later steps. If `found` is `false`, stop: "No merged PR found for phase #$0 — has it been merged yet?". If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

Also derive the trunk branch and worktree path now — both are needed later (branch verification in step 3, worktree removal in step 8) and this is one call:

```bash
marvin names derive <plan> --phase <phase>
```

Read the `main_branch:` line (call it `<main_branch>` for the rest of this skill) and the `worktree_path:` line.

### 2. Verify the phase issue is still open

```bash
gh issue view $0 --repo <repo> --json state,title
```

If the issue state is already `CLOSED`, ask the user whether to proceed (the PR's `Closes #` may have already auto-closed it) or stop. The wrap-up comment and phase doc are still useful even if the issue is closed.

### 3. Verify branch state

```bash
git branch --show-current
```

Must equal `<main_branch>` from step 1 — this skill writes the per-phase doc file directly to the trunk branch, the same branch `finish-impl` later requires. If not, stop and ask the user to check out `<main_branch>` first.

Pull the latest from the remote — the phase PR just merged there and the local branch may be behind:

```bash
git pull origin <main_branch>
```

### 4. Delegate PR-history analysis to a sub-agent

Read `SUPPLEMENTS/CLASSIFY.md` for the classification rubric and output format.

Spawn a **general-purpose** agent (model **sonnet**, no worktree isolation needed) with this task:

> Classify the history of merged PR #<pr-number> in repo `<repo>` per the rubric defined in `<absolute path to skills/wrap-phase/SUPPLEMENTS/CLASSIFY.md>`. Read that file first, then fetch the PR using these commands and analyze its body, review comments, inline comments, and commits:
>
> ```
> gh pr view <pr-number> --repo <repo> --json number,title,body,commits,reviews,comments
> gh api repos/<repo>/pulls/<pr-number>/comments  # inline review comments
> ```
>
> Return a single JSON object with keys `decisions`, `scope_changes`, `deferred`, `corrections`, `implementation_summary`, `test_plan`, per the rubric. Return only the JSON, no commentary.

The sub-agent does its own reading inside its own context window — do not fetch the PR contents in the orchestrator first.

### 5. Render both drafts

From the sub-agent's JSON:

- The **wrap-up comment**, using `SUPPLEMENTS/COMMENT_TEMPLATE.md` (the four classification categories only — `implementation_summary` and `test_plan` do not appear here). Skip any section whose array is empty.
- The **phase doc**, using `SUPPLEMENTS/PHASE_DOC_TEMPLATE.md`: the phase issue body fetched in step 1 (verbatim, from its first `##` heading onward), plus `## Implementation` (`implementation_summary`), `### Test Plan / Verification` (`test_plan`), and the same four classification categories reused from the comment draft above. The filename is `docs/stories/<plan>/phase-NN-<slug>.md`, where `NN` is the `phase:` ordinal from step 1 zero-padded to at least two digits (`01`, `12`, `100`) and `<slug>` is step 1's `slug:` output verbatim — never hand-slugify; this is what lets `finish-impl` link to the exact filename this skill wrote.

### 6. Present for confirmation

Show the user the wrap-up comment draft in full, and the phase doc's new `## Implementation` and `### Test Plan / Verification` sections (the four classification sections are identical to the comment draft already shown — no need to repeat them). Ask:

> "This will be posted as a comment on impl plan #<impl-plan-issue>, written to `docs/stories/<plan>/phase-NN-<slug>.md` (committed and pushed to `<main_branch>`), and the phase issue will be closed and moved to Done. Edit anything, or proceed?"

Iterate until the user approves. **Any edit to a classification category applies identically to both renders** — the comment and the phase doc's matching section must not drift from each other. Do not proceed until explicitly confirmed.

### 7. Post the wrap-up comment on the impl plan issue

```bash
gh issue comment $1 --repo <repo> --body "$(cat <<'EOF'
<approved comment body>
EOF
)"
```

### 8. Write, commit, and push the phase doc

```bash
test -f docs/stories/<plan>/phase-NN-<slug>.md
```

**Read the path first if it exists** — the Write tool refuses to overwrite a path it has not read this session. This happens on a re-wrap of the same phase; always regenerate the file (never skip-if-exists) since a re-wrap's output supersedes the previous version and git history already preserves what came before.

Write the approved phase doc content with the **Write** tool, then stage, commit, and push:

```bash
set -e
test -f docs/stories/<plan>/phase-NN-<slug>.md || { echo "phase doc missing — aborting" >&2; exit 1; }
git add docs/stories/<plan>/phase-NN-<slug>.md
if git diff --cached --quiet; then
  echo "phase doc already up to date — no-op"
else
  git commit -m "docs: add phase doc for [PLAN-XXXXX-N] <title>"
fi
git log origin/<main_branch>..HEAD --oneline
git push origin <main_branch>
```

Any non-zero exit from `git add`, `git commit`, `git log`, or `git push` stops here and surfaces the error — a failed `git add` must never fall through to the no-op branch, whose message is identical to a legitimate re-run. If `git log origin/<main_branch>..HEAD` shows commits this run did not just make, report them to the user rather than pushing silently. The `git push` runs unconditionally, outside the no-op `if`/`else`.

### 9. Close the phase issue (if still open) and move to Done

```bash
marvin board move $0 done
gh issue close $0 --repo <repo> --reason completed
```

`marvin board move done` sets the board status and closes the issue when `done` is configured. The explicit `gh issue close` is the fallback for boards where `done: n/a` (where the board move is a no-op). Running both is safe — `gh issue close` on an already-closed issue exits 0.

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

### 10. Remove the phase worktree and clear the findings cache

The phase worktree was created by `/implement-phase` and left in place for review. Its path is `<worktree_path>` from step 1 — remove it:

```bash
marvin worktree remove <worktree_path>
marvin worktree prune
```

If the path is not registered with git (e.g. manually removed earlier), `marvin worktree remove` is a no-op — skip silently. A manually-deleted-but-still-registered ("prunable") worktree is not a no-op: `marvin worktree remove` cleans it up via `git worktree remove --force`.

Do **not** delete the phase branch (`<type>/PLAN-XXXXX/phase-N`) — it remains on the remote as the merge source and is useful for archaeology.

Clear the plan's findings cache — review, drift, and red-team findings accumulated during this phase are now stale:

```bash
marvin findings clear <plan>
```

Where `<plan>` is the plan identifier from step 1 (e.g. `plan-00042`). This removes `.claude/cache/<plan>/` entirely. If the directory is already absent, this is a no-op.

### 11. Confirm

Report:
- Comment URL on impl plan issue
- `docs/stories/<plan>/phase-NN-<slug>.md` commit (or no-op) and push status
- Phase issue closed and moved to Done
- Worktree removed
- Findings cache cleared

If more phases remain: "Next: `/implement-phase <next-phase-issue-number>`"
If this was the last phase: "Next: `/review-impl <impl-plan-issue-number>` (comprehensive cross-phase review), then `/finish-impl <impl-plan-issue-number>`."
