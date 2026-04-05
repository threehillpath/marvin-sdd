# Plan Naming Conventions

All plan-related issues follow a shared numbering sequence. XXXXX is zero-padded to 5 digits (e.g., `00014`).

| Issue type | Title format | Example |
|---|---|---|
| Architecture plan | `[PLAN-XXXXX-ARCH] Title` | `[PLAN-00014-ARCH] Member Invitations` |
| Implementation plan | `[PLAN-XXXXX] Title` | `[PLAN-00014] Member Invitations` |
| Phase | `[PLAN-XXXXX-N] Title` | `[PLAN-00014-1] Invitation Domain Model` |

## Plan number = source issue number

The plan number is the GitHub issue number being addressed. No scanning needed.

- Source issue #111 → PLAN-00111, zero-padded to 5 digits: `00111`
- Source issue #42 → PLAN-00042

## Relationship

One arch plan → one impl plan → one or more phase issues, all sharing the same XXXXX (the source issue number).

For the edge case where one arch plan warrants multiple independent impl plans, see `arch-plan/SUPPLEMENTS/NAMING.md`.

## Branch naming

Branch names mirror the plan number, lowercase with hyphens:

| Branch type | Pattern | Example |
|---|---|---|
| Implementation | `feature/plan-XXXXX` | `feature/plan-00111` |
| Phase | `feature/plan-XXXXX-N` | `feature/plan-00111-1` |
| Bug fix | `bug/bug-XXXXX` | `bug/bug-00143` |

Implementation branches are created from `main` by `/start-impl`. Phase branches are created from the implementation branch by `/implement-phase`. Phase PRs target the implementation branch; the implementation PR targets `main`.
