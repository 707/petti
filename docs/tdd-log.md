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

## Remaining slices

1. Visual/UI polish and richer status rendering.
2. Optional real-binary integration tests for package managers on CI.
