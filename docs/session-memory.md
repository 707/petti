# Session Memory

## Current state

- Repo bootstrapped as a Go module for a Bubble Tea implementation.
- Core v1 implementation now exists across collectors, UI state, Bubble Tea TUI, CLI orchestration, and binary entrypoint.
- `internal/model`, `internal/collectors`, `internal/app`, `internal/export`, `internal/tui`, `internal/cli`, and `cmd/pkgview` are all at 100% statement coverage with passing tests.
- Repo tooling now includes a `Makefile`, CI workflow, and coverage gate script.
- As of 2026-03-24, all further work must follow strict red-green-refactor TDD with 100% statement coverage on changed packages and 100% passing tests before a slice is considered complete.

## Next steps

1. Add polish slices for richer TUI presentation, status/error visibility, and export UX without breaking the current coverage bar.
2. Optionally add integration tests against real package-manager binaries on suitable CI runners.
3. Keep session memory and TDD log updated after each new behavior slice.

## Decisions

- Use Go rather than the Python/Textual implementation described in the PRD.
- Include Homebrew casks by default.
- Treat development continuity as checked-in docs plus Git history, not end-user persisted UI state.
- Strict engineering rule: show red first, then minimal green, then refactor.
- Strict quality rule: maintain 100% passing status and 100% code coverage for every package changed in the active slice.
- Escalation rule: if implementation requires a major behavior or architecture change from the agreed plan, stop and flag it to the user before proceeding.
