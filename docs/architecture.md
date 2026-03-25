# Architecture

`petti` is a read-only Go CLI/TUI that aggregates package inventory from Homebrew, npm, and pip.

## Components

- `internal/model`: shared package and collector status types
- `internal/collectors`: subprocess-backed discovery logic and concurrent aggregation
- `internal/app`: UI-agnostic state for filtering, sorting, selection, and summary counts
- `internal/export`: txt/json export functions
- `internal/tui`: Bubble Tea integration layer
- `internal/cli`: flag parsing, headless export, and TUI launch orchestration

## Collector contract

Each collector must:

- report availability without panicking when the binary is absent
- return structured status for ready/missing/timeout/error outcomes
- avoid mutating the system and only read from existing package managers
- enrich package rows with optional local metadata when it can be derived without mutating package-manager state

## UI contract

The TUI owns rendering, key handling, and message flow. Sorting/filtering logic should stay testable outside Bubble Tea where possible.
The next full-layout pass should prefer custom-rendered chrome and, if necessary, a custom grid/viewport over forcing all layout concerns through `bubbles/table`.

## Current UI model

- `full` layout is the default.
- `compact` layout preserves the original stacked renderer.
- `compact` is selected explicitly with `--layout compact`.
- `full` layout uses a custom viewport-backed dense grid for package rows.
- `compact` layout still uses `bubbles/table`.
- Full-layout rows can include package metadata such as description, updated date, action required, and last-used date when available.
- UI modes are explicit: `NORMAL`, `FILTER`, `SELECT`, `EXPORT`.
- Full-layout details are collapsed by default and open on demand.
- Full-layout filters are popup-driven from normal mode:
  - `f` -> source filter
  - `a` -> action filter
  - `u` -> updated-date filter
- `SELECT` mode currently maps:
  - `PKG` -> filter focus
  - `VER` -> version sort
  - `SRC` -> source popup
  - `UPDATED` -> updated-date sort
  - `ACTION` -> action popup
- The full-layout renderer should prefer removing dead UI branches over inventing impossible test setups; the 100% coverage rule is enforced against live behavior, not unreachable states.
