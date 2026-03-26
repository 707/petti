# Release Checklist

## Before tagging

1. Confirm the working tree is clean:
   ```sh
   git status --short
   ```
   Expected for a clean release cut:
   - no tracked file changes
   - untracked docs/images are either intentionally excluded or committed separately
2. Run the full verification gate:
   ```sh
   make test
   make cover-check
   make build
   ```
3. Smoke-test the release build locally:
   ```sh
   make build-release VERSION=0.6.5
   ./petti --version
   ```
4. Review [README.md](/Users/nad/Documents/Tests/codextest/README.md) for final repo URL, install instructions, and demo asset links.
5. Make sure the repo has a license file in the root.

## Create the release commit

1. Stage the intended files:
   ```sh
   git add .
   ```
2. Create the release-prep commit:
   ```sh
   git commit -m "release: prepare v0.6.5"
   ```

## Publish on GitHub

1. Push the branch:
   ```sh
   git push origin <branch-name>
   ```
2. Tag the release:
   ```sh
   git tag v0.6.5
   git push origin v0.6.5
   ```
3. Wait for the `Release` GitHub Actions workflow to publish the binaries and checksums.
4. Do not retag or replace a published release tag. Cut a new version instead.
5. If a release fails after tagging:
   - fix the issue on `main`
   - bump to the next version
   - cut a fresh tag
   - do not reuse the failed published version tag

## After the GitHub release

1. Verify the generated artifacts exist for:
   - `darwin-amd64`
   - `darwin-arm64`
   - `linux-amd64`
   - `linux-arm64`
2. Verify the `update-homebrew-tap` job inside the main `Release` workflow ran successfully if `HOMEBREW_TAP_TOKEN` is configured.
3. Replace the auto-generated GitHub release body with a short human-written summary:
   - headline improvements
   - install/upgrade command
   - note that Homebrew now follows the tagged release automatically when the tap job succeeds
4. Validate at least one install path from the published release:
   ```sh
   brew install 707/petti/petti
   petti --version
   ```
5. Validate upgrade behavior after at least one prior version exists:
   ```sh
   brew update
   brew upgrade petti
   ```

## Recommended immediate follow-up

1. Add a demo GIF or screenshot to the README.
2. Keep `HOMEBREW_TAP_TOKEN` configured so tap checksum updates remain automatic.
3. If Homebrew does not advance after a release, check the `update-homebrew-tap` job before changing the formula by hand.
4. Optionally publish a scoped npm package later if `npx` support becomes a priority.

## Required GitHub secrets

- `GITHUB_TOKEN`
  - provided automatically to the release workflow
- `HOMEBREW_TAP_TOKEN`
  - required for the `update-homebrew-tap` job in the main `Release` workflow
- `NPM_TOKEN`
  - only needed if scoped npm publishing is enabled later
