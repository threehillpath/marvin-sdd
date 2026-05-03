# Review Findings Format

The review sub-agent (spawned by `review-phase` and `review-impl`) returns a single JSON object in this shape. The orchestrator parses it to render the report and post a GitHub PR review. A future auto-fix loop will consume the same JSON.

## Schema

```json
{
  "summary": "<one-paragraph overall verdict>",
  "verdict": "approve" | "request-changes" | "comment",
  "blocking": [ <Finding> ],
  "nits":     [ <Finding> ]
}
```

### Finding

```json
{
  "id": "B1",
  "category": "tdd" | "self-review" | "spec-drift" | "correctness" | "security" | "integration",
  "file": "relative/path/to/file.ext",
  "line": 42,
  "end_line": 47,
  "summary": "<one sentence — fits in a PR review comment header>",
  "details": "<longer explanation: why this is a problem, what the consequence is>",
  "suggested_fix": "<concrete change — file/line/text level if possible>",
  "evidence": "<commit SHA, diff hunk reference, or quoted line(s) the finding is anchored to>"
}
```

## Field rules

- **`id`** — unique within the response. Convention: `B1`, `B2`, … for blocking; `N1`, `N2`, … for nits. Used by the orchestrator and (future) auto-fix loop to address findings individually.
- **`category`** — exactly one of the values listed in `REVIEW_RUBRIC.md`. `integration` is only valid in `review-impl` output.
- **`file`** — repo-root-relative path. If the finding spans multiple files, pick the primary location; mention the others in `details`.
- **`line`** — 1-indexed line in the file as it appears in the PR head. For multi-line findings, set `end_line`; otherwise omit `end_line` or set it equal to `line`. For findings that are about something *missing* (e.g. a missing export or test file), point at the most relevant existing line and explain in `details` — or use the line where the missing piece should be inserted.
- **`summary`** — one sentence, ≤ 100 chars. Will appear as the first line of the GitHub review comment.
- **`details`** — full explanation. Markdown allowed. No line-length limit, but be terse.
- **`suggested_fix`** — see "Suggested-fix discipline" in `REVIEW_RUBRIC.md`. Concrete and minimal.
- **`evidence`** — what makes the reviewer confident this is real. A commit SHA, a quoted line, or a brief diff reference. Helps the user verify quickly and gives an auto-fix loop something to anchor against.

## Verdict semantics

- **`approve`** — both arrays empty. The orchestrator will post a `gh pr review --approve` (after user confirmation).
- **`request-changes`** — at least one entry in `blocking`. Posts `gh pr review --request-changes` with all findings as inline comments.
- **`comment`** — `blocking` is empty but `nits` is non-empty. Posts `gh pr review --comment` with nits as inline comments.

The reviewer chooses the verdict based on the contents of the arrays — they must be consistent. The orchestrator validates and rejects mismatches.

## Example

```json
{
  "summary": "Phase delivers the member-creation handler and tests, but the new module is not exported from the domain barrel and one validation predicate is untested.",
  "verdict": "request-changes",
  "blocking": [
    {
      "id": "B1",
      "category": "self-review",
      "file": "domain/membership/index.ts",
      "line": 12,
      "summary": "CreateMember is not re-exported from the domain barrel",
      "details": "`domain/membership/member.ts` exports `CreateMember`, but `domain/membership/index.ts` does not re-export it. Callers importing from `domain/membership` will not see the new function. This will fail at runtime in any consumer that uses the barrel import.",
      "suggested_fix": "Add `export { CreateMember } from './member';` to `domain/membership/index.ts` line 12.",
      "evidence": "git show HEAD:domain/membership/index.ts shows no re-export line; commit a3f9c12 added the function but did not touch the barrel."
    }
  ],
  "nits": [
    {
      "id": "N1",
      "category": "tdd",
      "file": "domain/membership/validation.ts",
      "line": 8,
      "summary": "isValidEmail predicate has no direct test",
      "details": "The predicate is exercised indirectly through the CreateMember test, but a direct test would catch regressions in the predicate without depending on the handler.",
      "suggested_fix": "Add `domain/membership/validation_test.ts` with cases for valid, invalid, and edge-case emails.",
      "evidence": "validation.ts line 8; no validation_test.ts in the diff."
    }
  ]
}
```

## Output discipline

- Return **only** the JSON object. No prose before or after, no markdown code fence.
- If both arrays are empty, set `verdict: "approve"` and `summary` to one sentence describing what was reviewed.
- Do not invent findings to fill the response. An empty review is a valid review.
- Do not include findings about pre-existing code outside the diff — review the change, not the codebase.
