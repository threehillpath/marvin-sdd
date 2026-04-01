# Plan Naming Conventions

All plan-related issues follow a shared numbering sequence. XXX is zero-padded to 3 digits (e.g., `014`).

| Issue type | Title format | Example |
|---|---|---|
| Architecture plan | `[PLAN-XXX-ARCH] Title` | `[PLAN-014-ARCH] Member Invitations` |
| Implementation plan | `[PLAN-XXX] Title` | `[PLAN-014] Member Invitations` |
| Phase | `[PLAN-XXX-N] Title` | `[PLAN-014-1] Invitation Domain Model` |

## Plan number = source issue number

The plan number is the GitHub issue number being addressed. No scanning needed.

- Source issue #111 → PLAN-111, zero-padded to 3 digits: `111`
- Source issue #42 → PLAN-042

## Relationship

One arch plan → one impl plan → one or more phase issues, all sharing the same XXX (the source issue number).

For the edge case where one arch plan warrants multiple independent impl plans, see `arch-plan/SUPPLEMENTS/NAMING.md`.

## Branch naming

Branch names mirror the plan number, lowercase with hyphens:

| Branch type | Pattern | Example |
|---|---|---|
| Implementation | `feature/plan-XXX` | `feature/plan-111` |
| Phase | `feature/plan-XXX-N` | `feature/plan-111-1` |
| Bug fix | `bug/bug-XXX` | `bug/bug-143` |

Implementation branches are created from `main` by `/start-impl`. Phase branches are created from the implementation branch by `/implement-phase`. Phase PRs target the implementation branch; the implementation PR targets `main`.
