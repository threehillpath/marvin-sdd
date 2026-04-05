# Label Conventions

## Type labels (applied by skill, always)

| Issue type | Label |
|---|---|
| Architecture plan | `plan:arch` |
| Implementation plan | `plan:impl` |
| Phase | `plan:phase` |

These labels are created by the `configure-plan-plugin` skill during setup. If a label is missing, create it before applying:
```bash
gh label create "plan:arch" --repo <repo> --description "Architecture plan" --color "0075ca"
gh label create "plan:impl" --repo <repo> --description "Implementation plan" --color "0075ca"
gh label create "plan:phase" --repo <repo> --description "Phase / implementation unit" --color "0075ca"
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
