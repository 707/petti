# TDD Log

## Operating rule

All remaining implementation must use strict red-green-refactor:

1. Add or tighten one failing test.
2. Run the smallest relevant test target and confirm red.
3. Implement the minimum code required for green.
4. Re-run tests and coverage for the touched package(s).
5. Refactor only with tests still green and coverage still at 100%.

If a required change is materially different from the current plan, pause and flag it to the user before implementing it.

## Completed slices

1. Added package/source/status domain types and tests.
2. Added collector runner abstraction, parser-focused collector tests, and concurrent aggregation.
3. Added UI-agnostic state tests for filter/sort/summary behavior.
4. Added txt/json export tests.
5. Added Bubble Tea TUI model tests and implementation for filter/help/sort/refresh/export flows.
6. Added CLI orchestration tests and implementation for headless export, TUI launch, and flag handling.
7. Added binary entrypoint tests and implementation.
8. Added repo tooling for build/test/coverage CI.
9. Added full-width default layout, preserved compact layout behind `--layout compact`, source tabs, detail pane, column mode, and CLI `--layout`.
10. Added llmfit-style full-layout chrome: top status bar, boxed filter strip, constrained full-width body, and bottom mode/help bar.
11. Replaced the full-layout `bubbles/table` path with a custom viewport-backed dense grid while keeping compact on `bubbles/table`.
12. Added optional package metadata enrichment and surfaced it in the full-layout grid, detail pane, and mode/help shortcuts.
13. Added popup-driven source/action/updated filters, `SELECT`-style header actions, metadata columns in the full grid, and coverage-closing tests/refactors for the new full-layout paths.

## Remaining slices

1. Visual polish on the full-layout dense grid, popup styling, header treatment, selected-row emphasis, and detail pane.
2. Optional real-binary integration tests for package managers on CI.
3. Optional read-only jobs/history and snapshot diffing.

## Completed slices (continued)

14. Grid column refactoring: removed `used` column, renamed `action`→`status` and moved after `desc`, src column no longer truncates, updated column shows relative time ("4d ago"), box-drawing gridlines (┌┐└┘│─).
15. Full-layout grid/theme/deletion-plan slice: moved `LAST UPDATED` to the right of `DESC` with a 4-char two-line header, made `VER` half-width, added `USED BY`, added 3 named themes via `--theme`, colored full-layout chrome through centralized theme helpers, and added `d` to copy uninstall commands to the clipboard with status feedback.
16. Full-layout chrome/dependency-safety/info slice: removed the stray separator above filter boxes, made `?` visibly toggle full-layout help, renamed the dependency column to `DEPENDANCY` with `Y/N/-` semantics, added manager-specific package inspection, added the centered `i` info modal with live metadata, enabled runtime mouse support for row selection and modal toggling, and kept `make test`, `make cover-check`, and `make build` green at 100% statement coverage.
