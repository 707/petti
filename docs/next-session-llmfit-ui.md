# Next Session: llmfit-Style Full Layout

## Goal

Rebuild the `full` layout so it feels intentionally dense and structured like `llmfit`, while staying in Go on Bubble Tea/Lip Gloss.

This is not a framework migration task. `llmfit` itself is not Bubble Tea-based; it uses Rust `ratatui`/`crossterm`. The takeaway is UX structure, not library choice.

## Reference takeaways

- `llmfit` uses a persistent top summary bar.
- It gives filters first-class space in their own row, not just inline text.
- It treats the table as a dense data grid with hard width budgeting.
- It keeps a bottom mode/help/status bar visible at all times.
- It uses explicit modes to change what keys mean.

## Required product decisions for the next session

- Keep `full` as the default startup layout.
- Keep `compact` only behind `--layout compact`.
- Do not restore runtime layout switching.
- Keep the app read-only and offline.
- Keep strict red-green-refactor TDD and 100% coverage on touched packages.

## Target full-layout structure

### Row 1: System/status bar

- Left: `petti`, version, active mode
- Center: package totals and per-source counts
- Right: collector availability/error summary

### Row 2: Filter boxes

- Box 1: search input
- Box 2: source filter
- Box 3: sort key
- Box 4: optional quick flag such as `missing version`
- Use boxed segments rather than free text so the layout reads like a dashboard

### Row 3+: Main content

- Left/wide pane: custom dense package list
- Right pane: selected package detail panel
- Stop relying on the stock `bubbles/table` as the main visual primitive if it fights the layout
- Prefer a custom-rendered header row plus a `viewport`-backed list if that gives tighter control

### Bottom row: Mode/help bar

- Always-visible key legend
- Explicit mode label (`NORMAL`, `FILTER`, `COLUMN`, `EXPORT`)
- Show temporary status messages here, not floating in the body

## Implementation direction

1. Replace the current full-layout body renderer with explicit width budgeting.
2. Move from “table plus strings” toward a custom grid renderer if needed.
3. Render filter controls as visible boxes in the header area.
4. Keep compact mode untouched except for compatibility fixes.
5. Preserve current command semantics where still useful:
   - `/` search
   - `V` column mode
   - `e` export
   - `r` refresh
   - `?` help
   - `q` quit

## TDD slice order

1. Snapshot-style test for the top summary bar and filter row.
2. Width-budget test for the full-layout body at realistic terminal widths.
3. Detail-pane rendering tests for selected package metadata.
4. Mode/help bar rendering tests.
5. If needed, custom grid renderer tests replacing `bubbles/table` usage in full mode.

## Acceptance bar

- Full layout must visually fit within the provided window width.
- Full layout must visibly separate:
  - summary/status
  - filters
  - package grid
  - detail pane
  - mode/help bar
- Compact mode must still work via `--layout compact`.
- `make test`, `make cover-check`, and `make build` must all pass.
