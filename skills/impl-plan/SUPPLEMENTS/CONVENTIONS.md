# Implementation Plan Conventions

Plans are specifications, not implementations. Describe *what* to build and *why* — the implementer writes the actual code.

## Include

- **Metadata**: objective, author, status, dependencies, date
- **Scope**: what's in and explicitly what's out (with references)
- **Component specs**: files to create/modify, key type names, function signatures, behavioral requirements
- **Design decisions and rationale**: why this approach, trade-offs considered, security implications
- **Verification steps**: concrete commands with expected output
- **Success criteria**: measurable checklist
- **TDD entry point per component**: first test to write (see TDD.md)

## Exclude

- Full function implementations (no complete method bodies)
- Line-by-line code that will be copied verbatim
- Boilerplate any competent implementer would add (imports, standard error handling)

## Acceptable short-form code

- Snippets under ~10 lines that clarify intent or show a specific pattern
- Function signatures (contract without body)
- Config examples (env vars, constants)
- Behavioral pseudocode
- Verification commands with expected output
