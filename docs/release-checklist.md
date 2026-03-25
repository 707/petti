# Release Checklist

## Before tagging

1. Confirm the working tree is clean:
   ```sh
   git status --short
   ```
2. Run the full verification gate:
   ```sh
   make test
   make cover-check
   make build
   ```
3. Smoke-test the release build locally:
   ```sh
   make build-release VERSION=0.6.3
   ./pkgview --version
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
   git commit -m "release: prepare v0.6.3"
   ```

## Publish on GitHub

1. Push the branch:
   ```sh
   git push origin <branch-name>
   ```
2. Tag the release:
   ```sh
   git tag v0.6.3
   git push origin v0.6.3
   ```
3. Wait for the `Release` GitHub Actions workflow to publish the binaries and checksums.

## After the GitHub release

1. Verify the generated artifacts exist for:
   - `darwin-amd64`
   - `darwin-arm64`
   - `linux-amd64`
   - `linux-arm64`
2. Paste the release notes into the GitHub release if you want a cleaner changelog than the auto-generated one.
3. If you set up a Homebrew tap later, update the formula with the new archive URL and checksum.

## Recommended immediate follow-up

1. Add a demo GIF or screenshot to the README.
2. Add a Homebrew tap repository once the GitHub repo is stable.
3. Optionally add a small install script for macOS, Linux, and WSL2.
