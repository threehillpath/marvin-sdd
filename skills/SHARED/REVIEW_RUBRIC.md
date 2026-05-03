# Code Review Rubric

This rubric is loaded by `review-phase` and `review-impl` and passed verbatim to a sub-agent that performs the actual review. The sub-agent's output is parsed by the orchestrator using `REVIEW_FINDING_FORMAT.md`.

The reviewer reads the rubric, the spec being implemented, and the diff under review — then produces a structured findings JSON. The reviewer does **not** apply fixes itself; it only reports.

## What the reviewer checks

Each category below corresponds to a `category` value in the findings JSON.

### `tdd` — TDD compliance and the rendered-controls exemption

The full rule lives in `impl-plan/SUPPLEMENTS/TDD.md`. Apply it to the diff:

- Every new non-component module (handlers, derived state, validation, formatting, parsing, sorting, mapping, predicates) must have at least one test in the diff or in a co-located test file.
- Logic embedded inside a component file is a finding unless the test exists for it. The litmus test: **if the code can be tested with the DOM removed, it is logic — extract and test.**
- Tests added after implementation (commits show implementation before tests for that criterion) are a finding — TDD requires the failing test first. Commit order is the evidence.
- Tests that assert internal state instead of outermost observable behavior are a finding (low-severity nit unless the assertion is so coupled to internals that a refactor would falsely fail).

### `self-review` — Items from `implement-phase/SUPPLEMENTS/LOOP.md` §4

Re-run the self-review checks against the diff. The implementer was supposed to do these before staging; the reviewer verifies.

1. **Data-path tracing.** For every handler that transforms data before sending it to an API, dispatch, or store: confirm each value is the intended *post-mutation* value, not a stale intermediate. Off-by-one writes, pre-mutation reads, and "fixed in next render" patterns go here.
2. **Exports and registrations.** New modules, functions, types, components — confirm they are exported wherever peers are exported (barrel files, public-API modules, route registries, plugin manifests, DI containers). Missing registrations are blocking.
3. **Declaration order in stateful code.** Hooks before derived values that read them; state before handlers that close over them. Out-of-order declarations often pass type checks but fail at runtime.
4. **Unintended duplication.** Same data rendered twice, same listener registered twice, same migration step in two places — confirm intentional or flag.

### `spec-drift` — Divergence from the spec

Compare the diff to the phase issue's success criteria (for `review-phase`) or to the impl plan's component sections (for `review-impl`). Findings:

- A success criterion is not satisfied by the diff.
- The diff implements behavior the spec did not call for, with no Notes-section explanation.
- An interface (function signature, route, schema column, event payload) differs from what the spec specified, and the change is not documented in the PR body's Notes.

Drift recorded in the Notes section is **not** a finding — that is the legitimate channel. Drift without a Notes entry is.

### `correctness` — Bugs the tests would not catch

These require reading the code, not just running it. Examples:

- Race conditions, missed cancellation, missed cleanup (subscriptions, listeners, timers).
- Off-by-one, wrong loop bound, wrong comparator.
- Null/undefined handling that the test happens to avoid hitting.
- Error paths that swallow or rethrow incorrectly.
- Resource leaks (unclosed handles, unbounded caches, retained references).
- Type assertions or casts that hide a real type mismatch.

### `security` — Common vulnerabilities

- Injection (SQL, shell, template, command) where input is interpolated without escaping or parameterization.
- Secrets in code, env defaults, or test fixtures committed to the repo.
- Authorization checks missing on a new route or RPC.
- Unsafe deserialization, unsafe HTML rendering, unsafe redirect targets.
- Crypto misuse (weak algorithms, hardcoded keys, custom auth schemes).

### `integration` (review-impl only) — Cross-phase issues

Only applies to the impl-level review. Examples:

- Phase A added a column that Phase B's code does not use, suggesting a phase missed it.
- Two phases each added a similar helper instead of one phase using the other's.
- A deferred item recorded in a phase wrap-up was never picked up by a later phase.
- Migration ordering across phases would break if applied in the order the phases were merged.

## Severity assignment

Each finding is either `blocking` or `nit`.

**Blocking** — the PR should not merge as-is. Use for:
- Any `security` finding.
- `correctness` findings that change observable behavior incorrectly under a plausible input.
- `tdd` findings where required tests are missing for new logic.
- `self-review` findings of categories 2 (missing exports/registrations) and 3 (declaration order) — these break at runtime.
- `spec-drift` where a success criterion is unmet or an interface diverges without documentation.
- `integration` findings that produce incorrect behavior end-to-end.

**Nit** — the PR can merge; the author may choose to address. Use for:
- Style, naming, structure preferences.
- Test assertions that work but could be tighter.
- Internal-state coupling that is not actively wrong.
- Duplication that is small and easily refactored later.

If unsure, prefer `blocking` for anything that could ship a bug; prefer `nit` for anything that is taste.

## What the reviewer does not do

- Does not edit files.
- Does not run tests (the implementer was responsible for green; the review reads code).
- Does not post comments to GitHub (the orchestrator posts after the user approves).
- Does not redesign the spec — flags drift, does not propose alternatives unless asked in `suggested_fix`.

## Suggested-fix discipline

Each finding includes a `suggested_fix`. Keep it concrete and minimal — a future auto-fix loop will read it. Examples:

- Good: "Add `export { CreateMember } from './member'` to `domain/membership/index.ts`."
- Good: "Move the `useReducer` call above the `useEffect` that reads `state`."
- Bad: "Refactor for clarity."
- Bad: "Consider whether this is the right approach."

If the fix is genuinely a discussion (e.g. "two valid approaches; pick one"), say so in `suggested_fix` and label `nit` — auto-fix should skip, humans should decide.
