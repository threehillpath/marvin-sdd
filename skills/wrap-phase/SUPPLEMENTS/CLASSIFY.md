# PR History Classification Rubric

You are classifying the history of a merged phase PR. Your output is consumed by the `wrap-phase` skill, which will render it as a comment on the parent implementation plan issue and present it to the user for approval.

## Inputs

You will read:

1. The PR body (`gh pr view --json body`) — the original test plan and any Notes section.
2. The PR commit list (`gh pr view --json commits`) — chronological commits, including any pushed after the PR was opened.
3. PR-level review comments (`gh pr view --json reviews,comments`) — top-level discussion.
4. Inline review comments (`gh api repos/<repo>/pulls/<num>/comments`) — code-line-anchored comments.

Read all four before classifying. The `Notes` section of the PR body, if present, is the most authoritative record of corrections (per LOOP.md) — start there.

## Categories

### `decisions`

Substantive choices made during implementation that diverge from the original impl plan, where the diverging choice is the one that shipped. Examples: choosing one library over another, picking a different data structure, changing a function boundary.

Each entry: `{ "summary": "<one sentence>", "reasoning": "<why this choice was made>" }`

Exclude: trivial naming choices, formatting, style decisions made entirely by the author with no discussion.

### `scope_changes`

Things added to or removed from the phase mid-flight. Distinguished from `decisions` because they change *what* the phase delivers, not *how*.

Each entry: `{ "summary": "<one sentence>", "direction": "added" | "removed", "reason": "<why>" }`

### `deferred`

Concerns raised in review that were intentionally not resolved in this PR — the parties agreed to defer to a follow-up phase, a separate ticket, or "if it becomes a problem." Watch items go here.

Each entry: `{ "summary": "<one sentence>", "where_to_track": "<follow-up phase, ticket, or 'TBD'>" }`

### `corrections`

Commits pushed to the PR after it was opened that fix something the initial commit got wrong — whether self-caught or raised in review. The `Notes` section of the PR body should already have a structured entry per correction (per LOOP.md). Use those entries verbatim where they exist; for corrections not yet recorded in Notes, infer from the commit message and any associated review comments.

Each entry: `{ "what_changed": "<short>", "why_wrong": "<short>", "correction": "<short>" }`

## Output format

Return a **single JSON object**, no surrounding prose, with exactly these keys:

```json
{
  "decisions":      [ { "summary": "...", "reasoning": "..." } ],
  "scope_changes":  [ { "summary": "...", "direction": "added", "reason": "..." } ],
  "deferred":       [ { "summary": "...", "where_to_track": "..." } ],
  "corrections":    [ { "what_changed": "...", "why_wrong": "...", "correction": "..." } ]
}
```

If a category has no entries, return an empty array for it. Do not omit keys.

## Quality bar

- Be terse. Each `summary` should fit on one line.
- Cite the source: where a fact came from a specific review comment or commit, your reasoning should make that traceable in spirit (the orchestrator will not include URLs in the final output, but the user can find them).
- Do not editorialize. Report what happened; let the user judge.
- If the PR has no review comments and only one commit, all four arrays are likely empty — that is a valid result.
