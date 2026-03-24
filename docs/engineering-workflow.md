# Engineering Workflow

This document is the persistent implementation contract for `pkgview`.

## Mandatory development process

- Use strict red-green-refactor TDD for every behavior change.
- Never count a slice as complete unless the relevant test run shows red first, then green after the minimum implementation.
- Keep the test target narrow while iterating, then run the broader package-level suite before moving on.

## Mandatory quality gates

- 100% passing tests at all times before closing a slice.
- 100% statement coverage for every package changed in the active slice.
- If a package cannot realistically hold 100% coverage because of an external boundary, isolate that boundary behind a small adapter and fully cover the internal logic.

## Escalation rules

- Flag major behavior, architecture, or scope changes to the user before implementation continues.
- Do not silently reinterpret the PRD or plan when the change would alter user-facing behavior or core structure.

## Session continuity

- Update `docs/session-memory.md` when priorities or rules change.
- Update `docs/tdd-log.md` after each meaningful slice with the exact behavior added and any remaining gaps.
- Keep commits aligned to one TDD slice where possible so Git history mirrors the red-green progression.
