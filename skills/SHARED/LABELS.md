# Label Conventions

## Type labels (applied by skill, always)

| Issue type | Label |
|---|---|
| Architecture plan | `plan:arch` |
| Implementation plan | `plan:impl` |
| Phase | `plan:phase` |
| Task (single-cycle, no phase hierarchy) | `plan:task` |

These labels are ensured automatically at issue creation time by each plan-creating skill via `marvin label ensure --builtins`. Manual creation should not be necessary; if a label is missing, run:
```bash
marvin label ensure --builtins
```

## Status label (applied by skill, always)

All newly created plan issues get: `status:upcoming`

## Domain labels (inferred from context, confirmed before creating)

Infer from what the plan touches. Apply all that match. Define the right set for your project — examples:

| If the plan involves... | Label |
|---|---|
| Server-side logic, APIs, data layer | `domain:backend` |
| UI components, client-side routing | `domain:frontend` |
| Auth, identity, access control | `domain:identity` |
| CI/CD, infrastructure, migrations | `domain:infra` |

Present inferred domain labels to the user during review ("I'll apply: `domain:backend`, `domain:frontend` — correct?") and allow correction before creating the issue.

## Source issue labels (carry forward when relevant)

If the source issue has `bug` or `enhancement`, apply the same label to the arch/impl plan issue so the type of work is traceable.
