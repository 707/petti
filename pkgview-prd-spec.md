# pkgview — PRD & Technical Specification

> A 2026 TUI for Consolidated Package Inventory on macOS

| Field | Detail |
|---|---|
| Version | 1.0 — Draft |
| Date | March 2026 |
| Status | Ready for Engineering Review |
| Platforms | macOS (primary), Linux (secondary) |

---

## Table of Contents

1. [Overview](#1-overview)
2. [Problem Statement](#2-problem-statement)
3. [Goals & Non-Goals](#3-goals--non-goals)
4. [User Stories](#4-user-stories)
5. [Functional Requirements](#5-functional-requirements)
6. [TUI Design Specification](#6-tui-design-specification)
7. [Technical Specification](#7-technical-specification)
8. [Data Model](#8-data-model)
9. [CLI Interface](#9-cli-interface)
10. [Error Handling](#10-error-handling)
11. [Testing Requirements](#11-testing-requirements)
12. [Open Questions](#12-open-questions)

---

## 1. Overview

`pkgview` is a zero-dependency, read-only terminal UI that aggregates and displays explicitly user-installed CLI packages across Homebrew, npm (global), and pip in a single, navigable interface. No new package managers. No background daemons. No internet required.

---

## 2. Problem Statement

Developers on macOS routinely install CLI tools across multiple package managers. Over time, there is no single place to see what you have explicitly installed vs. what was pulled in as a dependency. Existing solutions are either:

- **Too narrow** — only cover one package manager
- **Too heavy** — fleet management tools (Kolide, osquery) aimed at IT teams
- **Update-focused** — tools like `topgrade` run updates, not inventory
- **Manual** — shell one-liners that must be authored and maintained

There is no lightweight, purpose-built TUI for this exact workflow.

---

## 3. Goals & Non-Goals

### Goals

- Aggregate explicitly user-installed packages from Homebrew, npm (global), and pip into one view
- Provide a navigable, filterable 2026-style TUI with keyboard controls
- Show package name, version, and source manager per entry
- Export the consolidated list to a plain text or JSON file
- Work entirely from existing tools already on the user's system
- Run in under 2 seconds on a modern Mac

### Non-Goals

- **No updates** — this is read-only; it will not upgrade or manage packages
- **No dependency graph** — only top-level, user-requested packages (leaves only)
- **No GUI / Electron app** — terminal only
- **No network requests** — fully offline
- **No support for language-specific envs** (virtualenv, nvm, rbenv) in v1
- **No Windows support** in v1

---

## 4. User Stories

| ID | As a... | I want to... | So that... |
|---|---|---|---|
| US-01 | Developer | See all my explicitly installed packages in one place | I don't have to run 3 separate commands |
| US-02 | Developer | Filter the list by name | I can quickly check if something is installed |
| US-03 | Developer | See which package manager owns each package | I know how to uninstall it |
| US-04 | Developer | Export the list to a file | I can document or back up my environment |
| US-05 | Developer | See package versions | I know if things are out of date at a glance |
| US-06 | Developer on a new Mac | Quickly audit what is installed | I can decide what to keep after a migration |

---

## 5. Functional Requirements

### FR-01 — Package Discovery

| ID | Requirement | Priority |
|---|---|---|
| FR-01a | Query `brew leaves` for Homebrew top-level formulae | P0 — Must |
| FR-01b | Query `brew list --cask` for Homebrew casks | P1 — Should |
| FR-01c | Query `npm list -g --depth=0` for global npm packages, stripping npm itself | P0 — Must |
| FR-01d | Query `pip list --not-required` for top-level pip packages | P0 — Must |
| FR-01e | Gracefully skip any manager not found on PATH | P0 — Must |
| FR-01f | Capture package version alongside name | P1 — Should |

### FR-02 — TUI Display

| ID | Requirement | Priority |
|---|---|---|
| FR-02a | Render a full-screen TUI on launch | P0 — Must |
| FR-02b | Display packages in a scrollable, column-aligned list | P0 — Must |
| FR-02c | Show columns: Name, Version, Source | P0 — Must |
| FR-02d | Support live fuzzy-search / filter by typing | P0 — Must |
| FR-02e | Allow sorting by Name, Source, or Version | P1 — Should |
| FR-02f | Show a per-source summary count in a status bar | P1 — Should |
| FR-02g | Support mouse scroll in addition to keyboard | P2 — Nice |

### FR-03 — Export

| ID | Requirement | Priority |
|---|---|---|
| FR-03a | Export full list to `.txt` (one package per line) via keyboard shortcut | P1 — Should |
| FR-03b | Export full list to `.json` with name/version/source fields | P1 — Should |
| FR-03c | Respect active filter when exporting (export what you see) | P2 — Nice |

---

## 6. TUI Design Specification

### 6.1 Layout

```
+---------------------------------------------------------------------+
|  pkgview  v1.0                               [?] help  [q] quit     |
+---------------------------------------------------------------------+
|  Filter: > _                                                         |
+--------------------------+--------------+---------------------------+
|  Package                 |  Version     |  Source                   |
+--------------------------+--------------+---------------------------+
|  bat                     |  0.24.0      |  homebrew                 |
|  eza                     |  0.18.9      |  homebrew                 |
|  fd                      |  10.1.0      |  homebrew                 |
|  fzf                     |  0.54.0      |  homebrew                 |
|> gh                      |  2.47.0      |  homebrew                 |
|  git-delta               |  0.17.0      |  homebrew                 |
|  typescript              |  5.4.3       |  npm                      |
|  eslint                  |  8.57.0      |  npm                      |
|  black                   |  24.3.0      |  pip                      |
|  ruff                    |  0.3.4       |  pip                      |
+--------------------------+--------------+---------------------------+
|  42 packages  .  homebrew: 28  .  npm: 8  .  pip: 6                |
|  [up/dn] navigate  [/] filter  [s] sort  [e] export  [q] quit      |
+---------------------------------------------------------------------+
```

### 6.2 Colour Palette

| Element | Colour |
|---|---|
| Background | #0D1117 (near-black) |
| Header / footer bar | #161B22 |
| Accent / active row | #00D4AA (terminal teal) |
| Homebrew source badge | #F97316 (orange) |
| npm source badge | #CB3837 (npm red) |
| pip source badge | #3B82F6 (blue) |
| Muted / secondary text | #6B7280 |
| Border lines | #30363D |

### 6.3 Keyboard Controls

| Key | Action |
|---|---|
| Up / Down | Navigate rows |
| PgUp / PgDn | Scroll page |
| g / G | Jump to top / bottom |
| / or typing | Enter filter mode |
| Esc | Clear filter |
| s | Cycle sort: Name > Version > Source > Default |
| e | Export menu (txt / json) |
| r | Refresh / re-query all package managers |
| ? | Toggle help overlay |
| q / Ctrl+C | Quit |

### 6.4 Help Overlay

Pressing `?` displays a modal overlay with the full key reference and a one-line description of each package manager query being used under the hood.

---

## 7. Technical Specification

### 7.1 Technology Choice

`pkgview` is implemented as a single Python script using only the Python standard library plus one well-established TUI library:

| Component | Choice | Rationale |
|---|---|---|
| Language | Python 3.10+ | Available by default on macOS; no install needed for users |
| TUI framework | `textual` (PyPI) | Best-in-class 2026 Python TUI; CSS-like styling, reactive |
| Packaging | `pipx install pkgview` | Isolated install, exposes binary to PATH, one command |
| Distribution | PyPI | Standard; no homebrew tap needed in v1 |

### 7.2 Architecture

```
pkgview/
  collectors/
    homebrew.py     # runs brew leaves + brew list --cask
    npm.py          # runs npm list -g --depth=0
    pip.py          # runs pip list --not-required --format=json
  models.py         # Package dataclass (name, version, source)
  app.py            # Textual App, screens, widgets
  export.py         # txt / json export handlers
```

### 7.3 Collector Interface

Each collector follows the same interface:

```python
class Collector(Protocol):
    name: str                           # "homebrew" | "npm" | "pip"
    def available(self) -> bool: ...    # checks shutil.which()
    def collect(self) -> list[Package]: ...
```

Collectors are run concurrently using `asyncio.gather` to minimise startup time.

### 7.4 Subprocess Strategy

All package manager queries use `subprocess.run` with:

```python
subprocess.run(
    cmd,
    capture_output=True,
    text=True,
    timeout=15,      # hard timeout per collector
    check=False,     # handle returncode manually
)
```

### 7.5 Homebrew Collector Detail

```bash
# Formulae (top-level only, no deps)
brew leaves --installed-on-request

# Versions
brew list --versions $(brew leaves --installed-on-request)

# Casks (all casks are user-requested by definition)
brew list --cask --versions
```

### 7.6 npm Collector Detail

```bash
npm list -g --depth=0 --json 2>/dev/null
```

Parse the `dependencies` key; strip `npm` itself from results.

### 7.7 pip Collector Detail

```bash
pip list --not-required --format=json 2>/dev/null
```

Falls back to `pip3` if `pip` is not found. In v1, only the system/user global pip is queried.

---

## 8. Data Model

```python
from dataclasses import dataclass
from typing import Literal

Source = Literal["homebrew", "homebrew-cask", "npm", "pip"]

@dataclass(frozen=True, order=True)
class Package:
    name:    str
    version: str      # empty string if unavailable
    source:  Source
```

JSON export schema:

```json
[
  {
    "name":    "bat",
    "version": "0.24.0",
    "source":  "homebrew"
  }
]
```

---

## 9. CLI Interface

```
Usage: pkgview [OPTIONS]

Options:
  --export-txt PATH    Export list to .txt and exit (no TUI)
  --export-json PATH   Export list to .json and exit (no TUI)
  --filter TEXT        Pre-populate the filter on launch
  --no-color           Disable colour output (plain mode)
  --version            Show version and exit
  --help               Show this message and exit
```

The `--export-*` flags allow `pkgview` to be used non-interactively in scripts, preserving the original one-liner use case.

---

## 10. Error Handling

| Scenario | Behaviour |
|---|---|
| Package manager not on PATH | Source silently omitted; status bar shows "homebrew: not found" |
| Command times out (>15s) | Source row shows warning; other sources still display |
| Command returns non-zero exit | Source row shows error; stderr logged to ~/.pkgview.log |
| Terminal too small (<80x24) | Show warning: "Terminal too small — resize to at least 80x24" |
| Python < 3.10 | Graceful exit with version error message |

---

## 11. Testing Requirements

| Test | Type | Notes |
|---|---|---|
| Each collector returns valid Package list | Unit | Mock subprocess |
| Collector skips gracefully when binary missing | Unit | Mock shutil.which returning None |
| Collector handles subprocess timeout | Unit | Mock TimeoutExpired |
| npm strips npm from its own output | Unit | |
| pip --not-required flag filters deps | Unit | Fixture with known output |
| Brew leaves excludes known dependency | Integration | Requires Homebrew on CI |
| TUI renders without crash on mock data | Smoke | Use Textual test harness |
| Export txt matches displayed packages | Unit | |
| Export json is valid and matches schema | Unit | |

---

## 12. Open Questions

| # | Question | Owner | Status |
|---|---|---|---|
| 1 | Should `brew leaves --installed-on-request` be used instead of plain `brew leaves`? | Engineering | Open |
| 2 | Support multiple Python versions (pyenv, conda)? Adds complexity; defer to v2? | Product | Deferred to v2 |
| 3 | Should casks be on by default or behind a `--casks` flag? | Product | Open |
| 4 | Should a Homebrew formula be the primary distribution instead of pipx? | Engineering | Open |
| 5 | Refresh interval — should r do a full re-query or serve a cache? | Engineering | Open |

---

*pkgview — read your stack, don't manage it.*
