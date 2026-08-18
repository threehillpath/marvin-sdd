---
name: finish-impl
description: Open a PR from the implementation branch to main and move the impl plan to In Review
argument-hint: <impl-plan-issue-number>
allowed-tools: Bash, Read, Write, Agent
model: sonnet
---

Close out a completed implementation: confirm all phases are merged, open a PR from the implementation branch to main, and move the impl plan issue to In Review.

**Before starting**: Read `.claude/plan-workflow-config.yml` for project configuration (repo, owner). Read `../SHARED/GLOSSARY.md` for branch and status conventions.

## Arguments

- `$0` — Impl plan issue number

## Steps

### 1. Fetch context

```bash
gh issue view $0 --repo <repo> --json number,title,body,comments
```

Extract PLAN-XXXXX from the title:

```bash
marvin parse title "<issue title>"
```

Read the `plan_number:` line from the output (e.g. `plan-00042`) and use it as `<plan>` in subsequent steps.

Extract the phase list from the "Phases created:" comment in the issue's comments:

```bash
echo "<phases-created-comment-body>" | marvin parse phase-list
```

This emits `found: true|false` followed by `issues: <comma-joined-numbers>` (or `issues: (none)`) — read the `issues:` line for the phase issue numbers.

### 2. Verify branch state

```bash
git branch --show-current
```

Must be on the story's trunk branch (call the current branch `<main_branch>` for the rest of this skill — its real name is already known from this command's output, never reconstructed). If not, warn and stop — instruct the user to check out the implementation branch first.

Pull the latest from the remote — phase PRs were merged on GitHub and the local branch may be behind:

```bash
git pull origin <main_branch>
```

```bash
git status
```

If there are uncommitted changes **other than untracked or modified files under `docs/stories/<plan>/`** (leftovers from an interrupted run of this skill, which the steps below regenerate), stop and ask. Otherwise the impl branch should be clean — all work arrives via merged phase PRs.

### 3. Summarize what is being shipped

```bash
git log main..HEAD --oneline
```

Present the summary:

```
Implementation branch: <main_branch>
Target: main

Commits since main:
  <list from git log>

Phases:
  [PLAN-XXXXX-1] <title>
  [PLAN-XXXXX-2] <title>
  ...

Will also commit and push `docs/stories/<plan>/` to `<main_branch>` before opening the PR.

This will open a PR: <main_branch> → main. Proceed?
```

Ask for confirmation before creating the PR.

### 4. Assemble and commit the story record

```bash
marvin issue tree $0 --json
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first." If it exits 1, stop and surface the error — a failed hierarchy walk is not the same as an empty one; the graceful-degradation handling below applies only to a successful call that returns an empty or partial result.

Parse the returned JSON array of nodes (each with `kind`, `number`, `title`, `state`, `status`). Identify the one `arch` node (if any), the `impl` node (matches `$0`), and zero or more `phase` nodes. For each `phase` node, determine its ordinal via:

```bash
marvin parse title "<node title>"
```

Read the `phase:` line from the output and sort the phase nodes by this **parsed ordinal** — not `issue tree`'s own issue-number sort order, which only coincides with phase order within a single `phase-split` run. `marvin issue tree` is authoritative for the phase list here — step 1's `marvin parse phase-list` result is used only for that step's own summary display and is not consulted in this step.

If the `phase` nodes list from `issue tree` is empty, fall back to:

```bash
marvin issue list --label "plan:phase" --title-prefix "[PLAN-XXXXX-" --state all
```

(trailing hyphen, no closing bracket — substitute the real plan number, e.g. `"[PLAN-00042-"`). For a multi-impl track (`[PLAN-XXXXX-A]`, `[PLAN-XXXXX-B]` — see `../SHARED/GLOSSARY.md`), apply the same rule one level deeper and use `--title-prefix "[PLAN-XXXXX-<suffix>-"` (e.g. `"[PLAN-00042-A-"`), so the fallback does not also match the sibling track's phases. Treat the rows this fallback returns exactly as `phase` nodes for the rest of this step — run `marvin parse title` on each title and sort by the parsed ordinal the same way as above. If that also returns nothing, note the absence in `phases.md`'s header rather than emitting an empty table. If no `arch` node is present, skip `arch-plan.md` entirely and note its absence in `phases.md`'s header — do not fail either case.

The story slug is the plan identifier `<plan>` captured in step 1. Directory: `docs/stories/<plan>/`.

Fetch each node's body:

```bash
gh issue view <node-number> --repo <repo> --json number,title,body,url
```

Before writing each target path, check whether it already exists:

```bash
ls docs/stories/<plan>/ 2>/dev/null
```

and **Read it first if it does** — the Write tool refuses to overwrite a path it has not read this session:

- `docs/stories/<plan>/arch-plan.md`: header (`# <title>` / `Source: <url>`) followed by the arch plan body verbatim. Omitted entirely if no arch node was found.
- `docs/stories/<plan>/impl-plan.md`: header (`# <title>` / `Source: <url>`) followed by the impl plan body verbatim.
- `docs/stories/<plan>/phases.md`: an index table (`| Phase | Issue | Title |`) followed by each phase's full body under a `## [PLAN-XXXXX-N] <Title>` heading, in the parsed-ordinal order established above.

Before staging, verify the files this step just wrote actually exist — a failed `git add` on a missing pathspec exits non-zero and stages **nothing at all** (not even the paths that do exist), which `git diff --cached --quiet` would then read as "nothing to commit," indistinguishable from a legitimate no-op:

```bash
test -f docs/stories/<plan>/impl-plan.md && test -f docs/stories/<plan>/phases.md || { echo "story files missing — aborting" >&2; exit 1; }
```

(only check `arch-plan.md` too if an arch node was found). Stage only the files this step writes — never the whole directory:

```bash
git add docs/stories/<plan>/arch-plan.md docs/stories/<plan>/impl-plan.md docs/stories/<plan>/phases.md
```

(omit `arch-plan.md` from the `git add` if no arch node was found). Commit, gating on `git diff --cached --quiet` (not the commit's exit code) to distinguish a genuine no-op from a real failure; report what's about to be published before pushing; push unconditionally, outside the no-op check:

```bash
if git diff --cached --quiet; then
  echo "docs/stories/<plan>/ already up to date — no-op"
else
  git commit -m "docs: add docs/stories/<plan>/ arch/impl/phase records for [PLAN-XXXXX] <title>"
fi
git log origin/<main_branch>..HEAD --oneline
git push origin <main_branch>
```

Any non-zero exit from `git add`, `git commit`, `git log`, or `git push` stops here and surfaces the error — do not proceed to PR creation with the docs uncommitted or unpushed. A failed `git add` must never fall through to the no-op branch, whose message is identical to a legitimate re-run. If `git log origin/<main_branch>..HEAD` shows commits this run did not just make, report them to the user rather than pushing silently.

### 5. Synthesize and commit the retrospective

Filter the impl plan issue's comments — already fetched in step 1 (`gh issue view $0 --json …,comments`), no new `gh` call needed:

- **Wrap-up comments**: bodies starting with `## Phase wrap-up: [PLAN-XXXXX-`. When two or more matched comments share the same `[PLAN-XXXXX-N]` ident (a phase re-wrapped), keep only the most recent by comment timestamp.
- **Red-team critique**: the body starting with `## Plan Red-Team — verdict:`. Keep only the most recent if `red-team-plan` was re-run.

If the number of matched wrap-up comments is fewer than the number of phase nodes resolved in step 4 (or, if step 4's fallback was used, fewer than that fallback returned), report the shortfall to the user before synthesizing — don't proceed silently on a partial set. A missing red-team comment is expected (it's optional) and is not reported as a shortfall.

Check whether `docs/stories/<plan>/retrospective.md` already exists:

```bash
ls docs/stories/<plan>/ 2>/dev/null
```

If it does, **skip regeneration and report it as already-present** rather than re-synthesizing — the sub-agent's prose output is not deterministic between runs, so re-running would churn the durable record with different wording for no reason. **To force a rewrite, delete `docs/stories/<plan>/retrospective.md` and re-run `finish-impl`** — this is the documented escape hatch; there is no flag or prompt for it.

Otherwise, spawn an **Agent** with:

- `subagent_type: "general-purpose"`
- `model: "sonnet"`
- No worktree isolation

Read `SUPPLEMENTS/RETROSPECTIVE.md` and inline its rubric, along with the filtered comment bodies, directly in the sub-agent's prompt. The sub-agent returns the `retrospective.md` markdown directly — no code fence, no surrounding prose. Write it with the **Write** tool (no preceding Read needed on this path — the existence check above already established the file is absent).

```bash
git add docs/stories/<plan>/retrospective.md
if git diff --cached --quiet; then
  echo "retrospective.md already up to date — no-op"
else
  git commit -m "docs: add retrospective for [PLAN-XXXXX] <title>"
fi
git log origin/<main_branch>..HEAD --oneline
git push origin <main_branch>
```

Any non-zero exit from `git commit` or `git push` stops here and surfaces the error. If `git log origin/<main_branch>..HEAD` shows commits this run did not just make, report them to the user rather than pushing silently. The `git push` runs **unconditionally**, outside the no-op `if`/`else` — a push with nothing new to send is itself a harmless no-op, and this is what lets a prior run's failed push get retried on the next run even when nothing new gets staged or committed that time.

### 6. Create PR

Read `../SHARED/PR_TEMPLATE.md` and use the **Implementation PR** template. Include `Closes #$0` to auto-close the impl plan issue on merge.

```bash
gh pr create --repo <repo> \
  --title "[PLAN-XXXXX] <Impl Plan Title>" \
  --base main \
  --body "<PR body>"
```

### 7. Move impl plan to In Review

```bash
marvin board move $0 in-review
```

If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

### 8. Clear the findings cache

Now that the impl PR is open, clear the plan's findings cache — any accumulated review, drift, and red-team findings are superseded by the impl-level review that `/review-impl` will produce:

```bash
marvin findings clear <plan>
```

Where `<plan>` is the plan identifier from step 1 (e.g. `plan-00042`). This removes `.claude/cache/<plan>/` entirely. If the directory is already absent, this is a no-op.

### 9. Confirm

Report: PR URL, `docs/stories/<plan>/` commit (or no-op) and push status, `retrospective.md` commit (or no-op/skip) and push status, impl plan issue #$0 moved to In Review, findings cache cleared.

**Next step**: `/review-impl $0` to review the impl PR and post findings directly on it.
