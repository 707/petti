#!/bin/sh
set -eu

REPO="${PETTI_REPO:-707/petti}"
TAP_REPO="${PETTI_TAP_REPO:-707/homebrew-petti}"
BINARY="${PETTI_BINARY:-petti}"

version_tag() {
  case "$1" in
    v*) printf "%s\n" "$1" ;;
    *) printf "v%s\n" "$1" ;;
  esac
}

release_asset_name() {
  version="$1"
  os_name="$2"
  arch_name="$3"
  clean_version=$(printf "%s" "$version" | sed 's/^v//')
  printf "%s_%s_%s_%s.tar.gz\n" "$BINARY" "$clean_version" "$os_name" "$arch_name"
}

checksum_for() {
  checksums_file="$1"
  asset_name="$2"
  awk -v asset="$asset_name" '$2 == asset { print $1; exit }' "$checksums_file"
}

download_checksums() {
  tag="$1"
  destination="$2"
  curl -fsSL -o "$destination" "https://github.com/$REPO/releases/download/$tag/checksums.txt"
}

write_formula() {
  version="$1"
  checksums_file="$2"
  formula_path="$3"

  clean_version=$(printf "%s" "$version" | sed 's/^v//')
  darwin_amd64_asset=$(release_asset_name "$clean_version" "darwin" "amd64")
  darwin_arm64_asset=$(release_asset_name "$clean_version" "darwin" "arm64")
  linux_amd64_asset=$(release_asset_name "$clean_version" "linux" "amd64")
  linux_arm64_asset=$(release_asset_name "$clean_version" "linux" "arm64")

  darwin_amd64_sha=$(checksum_for "$checksums_file" "$darwin_amd64_asset")
  darwin_arm64_sha=$(checksum_for "$checksums_file" "$darwin_arm64_asset")
  linux_amd64_sha=$(checksum_for "$checksums_file" "$linux_amd64_asset")
  linux_arm64_sha=$(checksum_for "$checksums_file" "$linux_arm64_asset")

  if [ -z "$darwin_amd64_sha" ] || [ -z "$darwin_arm64_sha" ] || [ -z "$linux_amd64_sha" ] || [ -z "$linux_arm64_sha" ]; then
    printf >&2 "missing release checksum(s) for %s\n" "$clean_version"
    return 1
  fi

  cat >"$formula_path" <<EOF
class Petti < Formula
  desc "Terminal UI for browsing installed packages across package managers"
  homepage "https://github.com/$REPO"
  version "$clean_version"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/$REPO/releases/download/v#{version}/${darwin_arm64_asset}"
      sha256 "${darwin_arm64_sha}"
    else
      url "https://github.com/$REPO/releases/download/v#{version}/${darwin_amd64_asset}"
      sha256 "${darwin_amd64_sha}"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/$REPO/releases/download/v#{version}/${linux_arm64_asset}"
      sha256 "${linux_arm64_sha}"
    else
      url "https://github.com/$REPO/releases/download/v#{version}/${linux_amd64_asset}"
      sha256 "${linux_amd64_sha}"
    end
  end

  def install
    bin.install "${BINARY}"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/${BINARY} --version")
  end
end
EOF
}

main() {
  version=""
  formula_path=""
  checksums_file=""

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --version)
        version="$2"
        shift 2
        ;;
      --formula)
        formula_path="$2"
        shift 2
        ;;
      --checksums-file)
        checksums_file="$2"
        shift 2
        ;;
      *)
        printf >&2 "unknown option: %s\n" "$1"
        exit 1
        ;;
    esac
  done

  if [ -z "$version" ] || [ -z "$formula_path" ]; then
    printf >&2 "usage: %s --version <version> --formula <path> [--checksums-file <path>]\n" "$0"
    exit 1
  fi

  tag=$(version_tag "$version")

  if [ -z "$checksums_file" ]; then
    temp_dir=$(mktemp -d)
    checksums_file="$temp_dir/checksums.txt"
    download_checksums "$tag" "$checksums_file"
  fi

  mkdir -p "$(dirname "$formula_path")"
  write_formula "$tag" "$checksums_file" "$formula_path"

  if [ "${temp_dir:-}" != "" ]; then
    rm -rf "$temp_dir"
  fi
}

if [ "${PETTI_TAP_LIB:-0}" != "1" ]; then
  main "$@"
fi
