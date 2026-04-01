# TDD Entry Points in Implementation Plans

Each component section in an impl plan must include a TDD entry point — the first test to write before any implementation for that component.

## Scope

TDD applies to backend logic (domain, repos, workers, handlers) **and** to any testable frontend logic — reducers, utility functions, data transformation, validation helpers, and other non-rendering code that runs in the browser.

**The exemption is narrow**: only rendered UI components and other code whose primary output is rendered UI elements are exempt. Running in the browser does not make code UI. If it has inputs, outputs, and no DOM dependency, it should have a TDD entry point.

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
