# Plan Drift Rubric

This rubric is loaded by `plan-drift` and passed verbatim to a fresh-context sub-agent that compares the current phase branch's diff against the phase spec. The reviewer's output is parsed by the orchestrator using `PLAN_DRIFT_FORMAT.md`.

The drift check answers two questions:

1. **Coverage** — for each success criterion in the phase spec, does the current diff satisfy it?
2. **Containment** — does the diff stay within the scope the phase spec described, or does it add work that was not asked for?

The drift check overlaps with — but does not replace — the `spec-drift` category in `REVIEW_RUBRIC.md`. The full code review runs once at PR time. Drift is run on demand mid-phase to catch coverage gaps and scope creep before they accumulate.

## Stance

You are auditing whether the work-in-progress matches the spec. Be literal. The spec is the contract. If the diff does something the spec did not call for, that is drift even if the addition is "obviously a good idea." If the spec calls for something the diff has not yet done, that is an unmet criterion even if it would be trivial to add.

Drift documented in the PR body's Notes section is **not** a finding — that is the legitimate channel for course corrections. Drift without a Notes entry is the finding.

Use extended thinking to walk every success criterion against the diff. Read enough of the surrounding code to verify whether a criterion is actually satisfied (the test exists *and* exercises the behavior; the export exists *and* is reachable from callers).

## Part 1 — Criterion coverage

For every checkbox in the phase spec's Success Criteria section, classify status:

- **`met`** — the diff (or pre-existing code unchanged by this diff) clearly satisfies the criterion. Cite the file/line or test name as evidence.
- **`partial`** — some of the criterion is satisfied but a named sub-condition is missing (e.g. happy path tested but the spec calls out an error path that is not).
- **`unmet`** — no part of the diff satisfies the criterion.
- **`unverifiable`** — the criterion is too vague to audit from the diff alone (this is itself a flag — note it and continue).

A criterion is `met` only if a future regression would falsify the evidence you cite. Trace the test to the code, not just the test file's existence.

## Part 2 — Drift findings

Each category below corresponds to a `category` value in the findings JSON.

### `out-of-scope` — Changes the spec did not describe

The diff modifies files, adds modules, or changes behavior outside what the phase's component specs cover. Examples:

- A refactor of an unrelated module the phase did not name.
- A "while I was here" cleanup of code the phase touches but does not own.
- A new helper, type, or constant added in a location the spec did not list.
- A dependency added or upgraded that the spec did not call for.

Pre-existing convention follows (auto-imports, formatter output, lint-fix sweeps the project enforces) are not drift. Use judgment.

### `interface-divergence` — Contracts differ from the spec

The diff implements a function signature, schema column, route, event payload, or config key that does not match the spec, and the divergence is not documented in the PR body's Notes section. Examples:

- Spec said `CreateMember(input) -> MemberID`, diff returns `(*Member, error)`.
- Spec said route `/members/:id`, diff registered `/api/v1/members/:id`.
- Spec said schema column `email VARCHAR(254)`, diff added `email TEXT`.

A reasoned divergence with a Notes entry is not a finding. A silent divergence is.

### `undocumented-change` — Behavior the spec did not authorize

The diff implements a behavior the spec did not call for, even within the right files. Examples:

- Spec asked for create + read; diff also implements update.
- Spec asked for a single validation rule; diff adds three.
- Spec asked for one endpoint; diff registers two.

Like interface-divergence, a Notes entry converts this from a finding into a recorded decision.

### `omission` — Specified work that the diff has not done

The diff does not include code or tests the spec explicitly called for. This overlaps with `unmet` criteria but is broader — covers things in component specs even if no success criterion explicitly tracks them. Examples:

- Component spec lists a function the diff does not define.
- TDD entry point names a test that does not exist in the diff.
- Spec calls for a migration; diff has no migration file.

For pre-PR drift checks, omissions are expected (the work is in progress). Flag only omissions that the user should know about — not every unfinished item.

## Severity assignment

Each finding is either `blocking` or `concern`.

**Blocking** — the diff should be revised, scope-trimmed, or the PR's Notes section updated before opening (or merging) the PR. Use for:
- Any `interface-divergence` that is not documented in PR Notes — callers will rely on the wrong shape.
- `out-of-scope` changes that touch shared modules or shipped contracts (others will inherit the change unintentionally).
- `undocumented-change` that adds behavior with security, persistence, or external-API implications.
- `omission` of items that are required for the success criteria to be `met`.

**Concern** — worth surfacing; the implementer may choose to address. Use for:
- `out-of-scope` changes that are local, harmless, and arguably improve readability.
- `omission` of items that are tracked by an `unmet` criterion (already captured in Part 1).
- Stylistic divergences that are not contractual.

If unsure, prefer `blocking` for anything that ships a contract the spec did not authorize; prefer `concern` for anything local and reversible.

## What the reviewer does not do

- Does not run tests.
- Does not edit files.
- Does not duplicate the full review rubric — leave correctness, security, TDD-quality, and self-review checks to `review-phase`. The drift check is *only* coverage and containment.
- Does not flag pre-existing code outside the diff.
- Does not propose redesigns. `suggested_action` is one of: remove from this phase, move to a follow-up issue, document in PR Notes, revise to match spec.

## Suggested-action discipline

Each finding's `suggested_action` should be one of these literal phrases (or a short, concrete variant):

- **"Remove from this phase"** — for out-of-scope or undocumented additions that should not ship in this PR.
- **"Move to follow-up issue: <one-line description>"** — for additions that are valuable but should be tracked separately.
- **"Document in PR Notes section"** — for divergences that are intentional and the implementer wants to keep.
- **"Revise to match spec"** — for interface-divergences where the spec is authoritative.
- **"Implement before opening PR"** — for omissions that are required for the criteria to be met.

If the right action is genuinely a discussion, label `concern` and explain in `details`.
