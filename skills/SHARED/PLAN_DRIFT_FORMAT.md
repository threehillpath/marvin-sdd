# Plan Drift Findings Format

The drift sub-agent (spawned by `plan-drift`) returns a single JSON object in this shape. The orchestrator parses it to render the report and (optionally) post a comment on the phase issue or PR. A future auto-correct loop will consume the same JSON.

## Schema

```json
{
  "summary": "<one-paragraph overall verdict>",
  "verdict": "aligned" | "attention" | "revise",
  "criteria": [ <CriterionStatus> ],
  "blocking": [ <Finding> ],
  "concerns": [ <Finding> ]
}
```

### CriterionStatus

```json
{
  "id": "SC1",
  "text": "<verbatim text of the criterion from the phase spec>",
  "status": "met" | "partial" | "unmet" | "unverifiable",
  "evidence": "<file:line, test name, or named absence — what makes the status defensible>"
}
```

### Finding

```json
{
  "id": "B1",
  "category": "out-of-scope" | "interface-divergence" | "undocumented-change" | "omission",
  "file": "relative/path/to/file.ext",
  "line": 42,
  "end_line": 47,
  "summary": "<one sentence — fits in a comment header>",
  "details": "<longer explanation: why this is drift, what the consequence is>",
  "suggested_action": "<one of the literal phrases in PLAN_DRIFT_RUBRIC.md, or a concrete variant>",
  "evidence": "<commit SHA, diff hunk reference, or quoted line(s) anchoring the finding>"
}
```

## Field rules

- **`id`** — unique within the response. Conventions: `SC1`, `SC2`, … for criteria (matching the order in the phase spec); `B1`, `B2`, … for blocking findings; `C1`, `C2`, … for concerns.
- **`text`** (criterion) — copied verbatim from the spec, including any leading/trailing punctuation. The orchestrator uses this to match findings back to the source spec when re-running.
- **`status`** (criterion) — exactly one of the four values. `met` requires the evidence to falsify a regression (test exists *and* exercises the behavior).
- **`category`** (finding) — exactly one of the values listed in `PLAN_DRIFT_RUBRIC.md`.
- **`file`** / **`line`** — repo-root-relative path and 1-indexed line in the file as it appears in the head ref under audit. For findings that are about something *missing* (e.g. an unimplemented function), point at the most relevant existing line and explain in `details`. For diff-wide findings (e.g. dependency added), use the manifest file (e.g. `package.json`).
- **`summary`** — one sentence, ≤ 100 chars.
- **`details`** — full explanation. Markdown allowed. No line-length limit, but be terse.
- **`suggested_action`** — see "Suggested-action discipline" in `PLAN_DRIFT_RUBRIC.md`. Use one of the literal phrases when possible.
- **`evidence`** — what makes the reviewer confident this is real. A commit SHA, a quoted line, or a brief diff reference.

## Verdict semantics

The reviewer chooses the verdict based on coverage and containment together:

- **`aligned`** — every criterion is `met` *and* both finding arrays are empty. The phase is ready to open (or merge) its PR with respect to drift.
- **`attention`** — at least one criterion is `partial`/`unmet`/`unverifiable`, but `blocking` is empty. Implementation is in progress or has minor concerns; nothing requires correction yet.
- **`revise`** — `blocking` is non-empty. The diff has out-of-scope work, undocumented divergence, or required-omission findings that must be addressed (or recorded in PR Notes) before the phase ships.

The orchestrator validates that the verdict matches the contents.

## Example

```json
{
  "summary": "Two of three criteria are met; one is unmet because the error path is not implemented yet. Diff also adds an unrelated lint-fix sweep across three files outside the phase's stated component scope.",
  "verdict": "revise",
  "criteria": [
    {
      "id": "SC1",
      "text": "CreateMember returns a member ID for valid input",
      "status": "met",
      "evidence": "domain/membership/member_test.go:18 'TestCreateMember_Valid' asserts non-empty ID; domain/membership/member.go:24 returns the ID."
    },
    {
      "id": "SC2",
      "text": "CreateMember returns ErrInvalid for missing email",
      "status": "unmet",
      "evidence": "No test in the diff covers the missing-email case; member.go has no early-return for empty email."
    },
    {
      "id": "SC3",
      "text": "Member row exists after CreateMember succeeds",
      "status": "met",
      "evidence": "member_test.go:32 SELECTs the row after CreateMember and asserts non-nil."
    }
  ],
  "blocking": [
    {
      "id": "B1",
      "category": "out-of-scope",
      "file": "internal/util/strings.go",
      "line": 1,
      "summary": "Lint-fix sweep across three util files unrelated to membership phase",
      "details": "The phase scope is `domain/membership/*` and `internal/db/*`. The diff also reformats `internal/util/strings.go`, `internal/util/dates.go`, and `internal/util/errors.go`. These files are unrelated to the phase and the changes are mechanical; they should not ship in this PR.",
      "suggested_action": "Remove from this phase (revert the util changes) or move to follow-up issue: 'Apply lint-fix to internal/util'.",
      "evidence": "git diff --name-only feature/plan-00042 shows internal/util/* in the diff; phase spec scope does not list internal/util."
    }
  ],
  "concerns": [
    {
      "id": "C1",
      "category": "omission",
      "file": "domain/membership/member.go",
      "line": 24,
      "summary": "Missing-email error path tracked by SC2 is not yet implemented",
      "details": "Captured in criterion SC2 above. Listed here as a concern only because the phase is mid-implementation; not a blocker for the drift check.",
      "suggested_action": "Implement before opening PR.",
      "evidence": "member.go:24 returns id, nil unconditionally; no validation branch."
    }
  ]
}
```

## Output discipline

- Return **only** the JSON object. No prose before or after, no markdown code fence.
- Always populate `criteria` with every success criterion from the spec, even when status is `met`. The orchestrator renders the full table.
- If `blocking` and `concerns` are both empty and every criterion is `met`, set `verdict: "aligned"`.
- Do not invent findings to fill the response. An empty drift list is valid.
- Do not include findings about pre-existing code outside the diff under audit.
- Do not duplicate the broader code-review rubric (correctness, security, TDD quality). Drift is only coverage and containment.
