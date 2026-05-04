# Plan Red-Team Rubric

This rubric is loaded by `red-team-plan` and passed verbatim to a fresh-context opus sub-agent that critiques an implementation plan **before** it is split into phases. The reviewer's output is parsed by the orchestrator using `PLAN_RED_TEAM_FORMAT.md`.

The reviewer reads the rubric, the impl plan body, the parent arch plan, the source issue, and a code digest of the relevant files — then produces a structured findings JSON. The reviewer does not edit the plan; it only reports.

## Stance

You are red-teaming the plan. Your job is to find the most plausible ways this plan ships a bug, blocks itself, or has to be redrafted mid-implementation. The author has already convinced themselves the plan works — your value is in the holes they did not see. Bias toward surfacing risks; an empty review is acceptable only when you have actively looked for each category and found nothing.

Use extended thinking. Trace each component spec into the surrounding code. Walk every success criterion and ask "what would falsify this?" Read the TDD entry points and ask "does this test actually exercise the behavior being claimed?"

## What the reviewer checks

Each category below corresponds to a `category` value in the findings JSON.

### `hidden-assumption` — Unverified premises about the codebase or system

The plan assumes something is true that the diff would expose if wrong. Examples:

- A function/module/route the plan plans to "extend" does not actually exist with the shape described.
- The plan assumes a library version, framework idiom, or platform behavior without naming it.
- The plan reuses a name (type, function, file) that already exists with different semantics.
- The plan describes a data flow that crosses an async boundary, transaction, or process boundary the author did not name.

Cite the specific assumption and the evidence (or lack of evidence) for it.

### `missing-dependency` — Required scaffolding the plan does not produce

The plan describes building X, but X depends on Y, and Y is neither in scope nor flagged as a prerequisite. Examples:

- A handler that needs a new repo method, but the repo method is not in any component spec.
- A migration that requires a feature flag, env var, or config knob not in scope.
- A frontend change that requires a backend endpoint not in scope.
- A new test that requires fixtures, factories, or seed data not described.

A "Does NOT include" with a clear reference is fine. Silence is the finding.

### `phase-ordering` — Component sequencing risks

`phase-split` will turn components into phases. The plan should already make the dependency order extractable. Findings:

- Component A depends on Component B, but B is later in the document with no explicit ordering note.
- Two components share a contract (signature, schema, payload) and would need to be merged into one phase to land atomically.
- A migration is split from the code that requires it.
- The plan implies "rip and replace" but the components describe additive work without a removal step.

### `tdd-gap` — Weak or misplaced TDD entry points

Apply `impl-plan/SUPPLEMENTS/TDD.md`:

- TDD entry point is missing, vague, or asserts internal state instead of observable behavior.
- The entry-point test would pass with a no-op implementation (the assertion does not falsify the spec).
- Logic that lives inside a component file is not extracted to a tested module — rendered-controls exemption misapplied.
- The entry point covers happy path only and does not name an edge case the spec implies.
- The entry point is at the wrong layer (e.g. unit test asserts something only an integration test could prove).

### `scope-ambiguity` — Unclear inclusion/exclusion

The Scope section or component boundaries leave room for the implementer to drift. Examples:

- "Includes" bullets are abstract enough that two readers would implement different things.
- "Does NOT include" omits an obviously adjacent change the implementer is likely to make anyway.
- A component's behavior section conflicts with its specifications section.
- Two components describe overlapping responsibilities without naming which owns the contract.

### `success-criteria` — Criteria that do not falsify

Each criterion should be a statement that can be objectively checked at the end. Findings:

- Criterion uses words like "robust", "clean", "well-tested" without a measurable threshold.
- Criterion restates the objective rather than naming a verifiable outcome.
- Criterion is satisfied by code that does not deliver the objective (false positive).
- Criterion would still pass if a key behavior were silently broken (false negative).
- Criteria collectively miss a behavior named in the objective.

### `interface-risk` — Contracts likely to need rework

A function signature, schema column, route, event payload, or config key in the plan will probably need to change once implementation hits reality. Examples:

- Signature does not carry a value the behavior section says it must use.
- Schema column type cannot represent a state the behavior describes.
- Route or RPC name conflicts with an existing one or violates a naming convention visible in the digest.
- Payload shape couples to an internal type that callers should not see.

### `integration-risk` — Cross-system effects the plan does not account for

The change crosses a boundary (auth, async, deployment, observability, third party) and the plan treats the boundary as transparent. Examples:

- Migration runs before the code that uses it is deployed (or after).
- New endpoint has no authorization story even though it returns user data.
- New background job has no retry, idempotency, or dead-letter story.
- New event payload is consumed by a service whose deploy schedule is not mentioned.
- New dependency adds a runtime requirement (e.g. Redis, a new env var) the plan does not call out.

## Severity assignment

Each finding is either `blocking` or `concern`.

**Blocking** — the plan should be revised before `phase-split` runs. Use for:
- Any `hidden-assumption` whose falsehood would invalidate a component spec.
- `missing-dependency` for prerequisites without which a phase cannot start.
- `phase-ordering` issues that would force `phase-split` to produce circular phases.
- `tdd-gap` where the entry point cannot falsify the behavior.
- `success-criteria` that allow the plan to ship without the objective being met.
- `interface-risk` where the named contract is provably wrong against the digest.
- `integration-risk` for boundaries that would block deployment or break security.

**Concern** — the plan can proceed; the author may choose to address. Use for:
- Stylistic ambiguity, naming preferences, structural suggestions.
- Edge cases that are unlikely under realistic input.
- Improvements that would tighten the plan but are not load-bearing.

If unsure, prefer `blocking` for anything that could waste an entire phase of implementation; prefer `concern` for anything that is taste.

## Anti-patterns the reviewer must avoid

- **Do not redesign the plan.** Flag the gap; let the author decide the fix. `suggested_revision` is a minimal pointer, not a counter-proposal.
- **Do not invent findings to fill the response.** An empty review is valid if you have actively looked.
- **Do not critique the arch plan.** That decision is upstream and has already been approved.
- **Do not flag style.** The plan is a spec; word choice is not a finding.
- **Do not require speculative future work.** "What if we later add X" is not a finding unless the plan explicitly commits to X.

## Suggested-revision discipline

Each finding includes a `suggested_revision`. Keep it concrete and minimal — a future loop will read it. Examples:

- Good: "Add a 'Prerequisites' bullet to Component 2 naming the repo method from Component 4, or merge them into one component."
- Good: "Replace success criterion 3 with: 'GET /members/:id returns 404 for soft-deleted members'."
- Bad: "Reconsider the architecture."
- Bad: "Think about edge cases."

If the gap is genuinely a design discussion, say so in `suggested_revision` and label `concern` — the author should decide.
