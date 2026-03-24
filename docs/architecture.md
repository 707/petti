# Architecture

`pkgview` is a read-only Go CLI/TUI that aggregates package inventory from Homebrew, npm, and pip.

## Components

- `internal/model`: shared package and collector status types
- `internal/collectors`: subprocess-backed discovery logic and concurrent aggregation
- `internal/app`: UI-agnostic state for filtering, sorting, selection, and summary counts
- `internal/export`: txt/json export functions
- `internal/tui`: Bubble Tea integration layer

## Collector contract

Each collector must:

- report availability without panicking when the binary is absent
- return structured status for ready/missing/timeout/error outcomes
- avoid mutating the system and only read from existing package managers

## UI contract

The TUI owns rendering, key handling, and message flow. Sorting/filtering logic should stay testable outside Bubble Tea where possible.
