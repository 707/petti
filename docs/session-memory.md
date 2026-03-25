# Session Memory

## Current state

- Repo bootstrapped as a Go module for a Bubble Tea implementation.
- Core v1 implementation now exists across collectors, UI state, Bubble Tea TUI, CLI orchestration, and binary entrypoint.
- `internal/model`, `internal/collectors`, `internal/app`, `internal/export`, `internal/tui`, `internal/cli`, and `cmd/petti` are all at 100% statement coverage with passing tests.
- Repo tooling now includes a `Makefile`, CI workflow, and coverage gate script.
- `v0.5` has been committed as the baseline snapshot.
- The next UI iteration has started:
  - default full-width layout
  - preserved compact layout
  - compact selected only by CLI flag
  - explicit TUI modes with `SELECT` as the header-action mode
  - detail pane collapsed by default in full layout
  - CLI `--layout full|compact`
  - top status bar, boxed filter strip, and bottom mode/help bar added to the full layout
  - full layout now uses a custom viewport-backed dense grid instead of `bubbles/table`
  - compact layout remains on `bubbles/table` as the compatibility renderer
  - package rows now include optional metadata columns for updated date, action required, last used, and description
  - full layout now uses popup-driven filters for source/action/updated state
  - dead branches removed from the full-grid and select-mode handlers to keep the coverage gate honest
- As of 2026-03-24, all further work must follow strict red-green-refactor TDD with 100% statement coverage on changed packages and 100% passing tests before a slice is considered complete.

## Next steps

1. Use [next-session-llmfit-ui.md](/Users/nad/Documents/Tests/codextest/docs/next-session-llmfit-ui.md) as the brief for the next visual polish pass on the new full-layout grid.
2. Tighten the full-layout density, header treatment, selected-row styling, and popup styling so the screen feels more deliberately `llmfit`-inspired.
3. Treat compact as a compatibility layout only, selected by `--layout compact`.
4. Consider baseline snapshot diffing and read-only jobs/history only after the full layout is visually strong.

## Decisions

- Use Go rather than the Python/Textual implementation described in the PRD.
- Include Homebrew casks by default.
- Treat development continuity as checked-in docs plus Git history, not end-user persisted UI state.
- `llmfit` is inspiration for layout and navigation patterns, not a library choice.
- The `llmfit`-style feel in `petti` should come from custom-rendered Bubble Tea chrome and a viewport-backed grid, not from migrating frameworks.
- Strict engineering rule: show red first, then minimal green, then refactor.
- Strict quality rule: maintain 100% passing status and 100% code coverage for every package changed in the active slice.
- Escalation rule: if implementation requires a major behavior or architecture change from the agreed plan, stop and flag it to the user before proceeding.
