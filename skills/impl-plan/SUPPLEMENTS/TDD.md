# TDD Entry Points in Implementation Plans

Each component section in an impl plan must include a TDD entry point — the first test to write before any implementation for that component.

## Scope

TDD applies to backend logic (domain, repos, workers, handlers) **and** to all testable frontend logic — reducers, utility functions, data transformation, validation helpers, formatters, parsers, and other non-rendering code that runs in the browser.

### The exemption applies only to rendered controls

The exemption covers **only** the JSX/template markup, styling, layout, and rendering side-effects produced by a component. Anything else inside a component file must be extracted to a non-component module and tested. This includes:

- **Event handlers** (save, submit, click, change) that transform, filter, derive, or validate values before calling an API, dispatch, or store mutation.
- **Derived state** — values computed from props, hook outputs, or store reads before being rendered.
- **Validation, formatting, parsing, sorting, mapping** done inline.
- **Conditional rendering driven by non-trivial expressions** — extract the predicate to a tested helper, then render based on its result.

The litmus test: **if the code can be tested with the DOM removed, it is logic — extract and test it.** Only what genuinely cannot run without a DOM (the rendering itself) is exempt.

This rule is strict because UI components are where untested logic accumulates fastest — handlers grow over time, derivations get tangled, and bugs hide behind the markup.

## Principles

- One test, then one change. Never batch-write tests ahead of implementation.
- The entry point test should assert the outermost observable behavior of the component — what a caller or user would see, not internal state.
- Tests should be in the smallest testable units that still validate the behavior meaningfully.
- Prefer integration tests over unit tests when the component's value is in how it connects to other parts.

## What to specify in the plan

The TDD entry point in each component section should identify:
1. **What behavior** is being asserted (one sentence)
2. **Where the test lives** (file path pattern)
3. **What makes it pass** (the observable outcome — return value, DB state, HTTP response, etc.)

Do not write the test code itself in the plan. Describe intent.

## Example

```
### TDD Entry Point
Assert that CreateMember returns a member ID when given valid input and that the
member row exists in the database afterward.
Test lives in: `domain/membership/member_test.go`
Passes when: function returns non-nil ID with no error, and a SELECT confirms the row.
```

## Phase-level TDD

When the implementation plan is split into phases (via `/phase-split`), each phase should begin with its TDD entry point test before any implementation work starts. The phase issue will carry this forward.
