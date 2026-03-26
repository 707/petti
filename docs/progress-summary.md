# petti Progress Summary

Last updated: 2026-03-26

## Current state

`petti` is now publicly usable as a released Go CLI/TUI with working release automation, Homebrew distribution, an install script, and public-facing README assets.

## Completed

- Final app name set to `petti`
- Codebase renamed from `pkgview` to `petti`
- Go module path updated to `github.com/707/petti`
- Entry point moved to `cmd/petti`
- GitHub Releases working through `v0.6.5`
- GoReleaser configured and fixed
- MIT license added
- Homebrew tap created and published:
  - app repo: `707/petti`
  - tap repo: `707/homebrew-petti`
- Homebrew formula added and validated with `brew audit --strict`
- README updated for public use with demo assets
- Install script added at `scripts/install.sh`
- npm wrapper package scaffolded for future `npx` support
- npm publish workflow added
- Release notes updated
- Homebrew checksum drift prevention added:
  - `scripts/update_tap_formula.sh`
  - `scripts/update_tap_formula_test.sh`
  - `Update Homebrew Tap` workflow

## Verified

- `make test`
- `make cover-check`
- `make build`
- `make build-release VERSION=0.6.5`
- `npm pack --dry-run`
- Homebrew formula install tested successfully

## Distribution status

### Live now

- GitHub Releases
- Homebrew install:
  - `brew install 707/petti/petti`
- Install script:
  - `curl -fsSL https://raw.githubusercontent.com/707/petti/main/scripts/install.sh | sh`

### Prepared but not live

- npm distribution
  - unscoped `petti` is blocked by npm because the name is too similar to an existing package
  - if npm distribution is revisited, the practical path is a scoped package such as `@<scope>/petti`

## Product and UX work completed

- Full-layout TUI polish
- Dynamic column sizing improvements
- Relative updated-date formatting
- Theme system with light/dark variants
- Runtime theme cycling with `t`
- Dependency status column
- Copy uninstall command action
- Package info modal
- Mouse row selection
- Refresh progress feedback
- Startup latency improvements for Homebrew dependency analysis

## Remaining worthwhile next steps

- Decide whether to publish a scoped npm package
- Add README install docs for npm only if that publish happens
- Test Homebrew install on an additional clean machine/environment
- Consider a small landing page later if discoverability becomes important

## Useful references

- Main README: `README.md`
- Release checklist: `docs/release-checklist.md`
- Session notes: `docs/session-memory.md`
- TDD log: `docs/tdd-log.md`
- Product spec: `petti-prd-spec.md`
