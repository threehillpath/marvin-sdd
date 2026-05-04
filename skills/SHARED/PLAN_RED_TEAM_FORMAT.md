# Plan Red-Team Findings Format

The red-team sub-agent (spawned by `red-team-plan`) returns a single JSON object in this shape. The orchestrator parses it to render the report and post a critique comment on the impl plan issue. A future auto-revise loop will consume the same JSON.

## Schema

```json
{
  "summary": "<one-paragraph overall verdict>",
  "verdict": "approve" | "revise" | "discuss",
  "blocking": [ <Finding> ],
  "concerns": [ <Finding> ]
}
```

### Finding

```json
{
  "id": "B1",
  "category": "hidden-assumption" | "missing-dependency" | "phase-ordering" | "tdd-gap" | "scope-ambiguity" | "success-criteria" | "interface-risk" | "integration-risk",
  "section": "Component 2 — Member Handler",
  "summary": "<one sentence — fits in an issue comment header>",
  "details": "<longer explanation: why this is a problem, what the consequence is>",
  "suggested_revision": "<concrete change to the plan — section/bullet/sentence level if possible>",
  "evidence": "<quoted text from the plan, code-digest reference, or named gap>"
}
```

## Field rules

- **`id`** — unique within the response. Convention: `B1`, `B2`, … for blocking; `C1`, `C2`, … for concerns. Used by the orchestrator and (future) auto-revise loop to address findings individually.
- **`category`** — exactly one of the values listed in `PLAN_RED_TEAM_RUBRIC.md`.
- **`section`** — the heading or bullet path within the plan that the finding anchors to (e.g. `Scope`, `Component 3 — Repo`, `Success Criteria #4`). For findings that are about something *missing* across the plan, use `Plan-wide`.
- **`summary`** — one sentence, ≤ 120 chars. Will appear as the first line of the rendered finding.
- **`details`** — full explanation. Markdown allowed. Be terse but include enough that the author can verify without re-reading the rubric.
- **`suggested_revision`** — see "Suggested-revision discipline" in `PLAN_RED_TEAM_RUBRIC.md`. Concrete and minimal.
- **`evidence`** — what makes the reviewer confident this is real. A quoted line from the plan, a code-digest reference, or a specific absence. Helps the user verify quickly and gives an auto-revise loop something to anchor against.

## Verdict semantics

- **`approve`** — both arrays empty. The plan is ready for `phase-split`. The orchestrator posts a brief approval comment.
- **`revise`** — at least one entry in `blocking`. The plan should be revised before `phase-split`. The orchestrator posts the findings as a comment and recommends revision.
- **`discuss`** — `blocking` is empty but `concerns` is non-empty. The plan can proceed, but the author may want to address concerns first. The orchestrator posts the concerns and asks the user how to proceed.

The reviewer chooses the verdict based on the contents of the arrays — they must be consistent. The orchestrator validates and rejects mismatches.

## Example

```json
{
  "summary": "Plan is well-scoped but two components share a contract that phase-split would have to merge, and one success criterion would pass with a no-op implementation.",
  "verdict": "revise",
  "blocking": [
    {
      "id": "B1",
      "category": "phase-ordering",
      "section": "Components 2 and 4",
      "summary": "Handler in Component 2 and repo in Component 4 share a contract that must land atomically",
      "details": "Component 2 specifies `CreateMember(input) -> MemberID` calling `repo.Insert`. Component 4 specifies `repo.Insert(row) -> ID`. Splitting these into separate phases means Phase 2 cannot compile until Phase 4 lands. Either merge the components, or call out an explicit ordering constraint so phase-split groups them.",
      "suggested_revision": "Either merge Components 2 and 4 into a single 'Member persistence' component, or add a 'Prerequisites' line to Component 2 naming Component 4.",
      "evidence": "Component 2 §Specifications: 'CreateMember calls repo.Insert'. Component 4 introduces repo.Insert with no forward reference."
    }
  ],
  "concerns": [
    {
      "id": "C1",
      "category": "success-criteria",
      "section": "Success Criteria #3",
      "summary": "'Membership flow is robust' is not falsifiable",
      "details": "The criterion uses 'robust' without naming a measurable outcome. A no-op CreateMember that always returns a fixed ID would not violate it. Replace with an outcome that fails if the behavior regresses.",
      "suggested_revision": "Replace with: 'CreateMember returns a unique non-empty ID for valid input and returns ErrInvalid for missing email.'",
      "evidence": "Plan §Success Criteria: '- [ ] Membership flow is robust'"
    }
  ]
}
```

## Output discipline

- Return **only** the JSON object. No prose before or after, no markdown code fence.
- If both arrays are empty, set `verdict: "approve"` and `summary` to one sentence describing what was reviewed.
- Do not invent findings to fill the response. An empty review is a valid review when you have actively looked for each rubric category.
- Do not flag the arch plan. The reviewer's scope is the impl plan.
