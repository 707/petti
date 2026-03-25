# petti

<p align="center">
  <b>One terminal view for the packages you actually have installed.</b>
</p>

<p align="center">
  Browse Homebrew, casks, npm, and pip packages in one TUI with filtering, dependency signals, export, live theme switching, and read-only uninstall helpers.
</p>

<!-- Add demo.gif or screenshot here later -->

---

## Install

### Homebrew

```sh
brew install 707/petti/petti
```

If you prefer the explicit two-step form:

```sh
brew tap 707/homebrew-petti
brew install petti
```

### From source

```sh
git clone https://github.com/707/petti.git
cd petti
go build -o petti ./cmd/petti
```

### Local development binary

```sh
go build -o petti ./cmd/petti
./petti
```

### Run without building

```sh
go run ./cmd/petti
```

---

## Usage

### TUI (default)

```sh
./petti
```

The app detects packages from supported managers, shows them in a dense terminal grid, and lets you filter, inspect, sort, export, and review uninstall commands without changing your system.

### Useful startup flags

```sh
./petti --layout full
./petti --layout compact
./petti --theme default
./petti --theme default-light
./petti --theme ember-dark
./petti --theme frost-light
./petti --filter gh
./petti --version
./petti --export-txt packages.txt
./petti --export-json packages.json
```

---

## Keybindings

| Key | Action |
|---|---|
| `/` | Start search |
| `f` | Source filter popup |
| `a` | Action/status filter popup |
| `u` | Updated filter popup |
| `s` | Cycle sort |
| `V` | Column/select mode |
| `Enter` | Toggle detail pane |
| `i` | Open package info panel |
| `d` | Copy uninstall command |
| `t` | Cycle themes live |
| `e` | Export visible packages |
| `r` | Refresh package data |
| `?` | Toggle help |
| `q` | Quit |
| `j` / `k` or arrows | Move selection |
| `g` / `G` | Jump to top / bottom |
| `PgUp` / `PgDn` | Page up / down |

### Mouse support

- Click a row to select it.
- Click the selected row again to toggle package info.

---

## Themes

`petti` currently ships with 6 built-in runtime themes:

- `default-dark`
- `default-light`
- `ember-dark`
- `ember-light`
- `frost-dark`
- `frost-light`

The aliases `default`, `ember`, and `frost` start on the dark variant. Themes can be changed at launch with `--theme` or cycled live with `t`.

---

## Supported package managers

- Homebrew formulae
- Homebrew casks
- npm global packages
- pip / pip3 packages

### Dependency column

The `DEPENDANCY` column uses:

- `Y` when the package is a dependency of another installed package
- `N` when it is not currently a dependency of another installed package
- `-` when the app cannot determine that safely

---

## Platform support

Current intended target systems:

- macOS
- Linux
- WSL2

Notes:

- Homebrew support depends on `brew` being installed and available on `PATH`.
- On Linux and WSL2, npm and pip are the most direct package sources; Homebrew works if Linuxbrew is installed.
- Native Windows is not the first-class target yet.

---

## Export

You can export the currently visible package list to text or JSON:

```sh
./petti --export-txt packages.txt
./petti --export-json packages.json
```

From inside the TUI, press `e` and choose the format.

---

## Project structure

```text
cmd/
  petti/              # entrypoint
internal/
  app/                # app state, filtering, sorting
  cli/                # CLI argument handling and program startup
  collectors/         # Homebrew, cask, npm, pip collection and metadata
  export/             # txt/json export
  model/              # shared package models
  tui/                # Bubble Tea UI, rendering, themes, input handling
docs/
  tdd-log.md          # implementation log
  session-memory.md   # working notes
scripts/              # repo scripts
```

---

## Development

### Build

```sh
make build
```

### Test

```sh
make test
make cover-check
```

### Run locally

```sh
go run ./cmd/petti
```

---

## Distribution plan

Once the GitHub repo is live, the best next rollout is:

1. GitHub Releases with prebuilt binaries for macOS and Linux
2. A Homebrew tap for easy install on macOS and Linuxbrew
3. A small install script for macOS, Linux, and WSL2

That gives you the cleanest path for the current target platforms without overcommitting to native Windows packaging too early.

---

## Roadmap

- screenshot / GIF in README
- GitHub Releases
- Homebrew tap
- broader metadata coverage per package manager
- native Windows support if the package-manager story is clarified

---

## License

Add your preferred license file in the repo root before publishing.
