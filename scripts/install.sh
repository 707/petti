#!/bin/sh
set -eu

REPO="707/petti"
BINARY="petti"

normalize_os() {
  case "$1" in
    Darwin) printf "darwin\n" ;;
    Linux) printf "linux\n" ;;
    *) printf >&2 "unsupported OS: %s\n" "$1"; return 1 ;;
  esac
}

normalize_arch() {
  case "$1" in
    x86_64) printf "amd64\n" ;;
    arm64|aarch64) printf "arm64\n" ;;
    *) printf >&2 "unsupported architecture: %s\n" "$1"; return 1 ;;
  esac
}

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

latest_version() {
  curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" |
    awk -F'"' '/"tag_name":/ { print $4; exit }'
}

checksum_for() {
  checksums_file="$1"
  asset_name="$2"
  awk -v asset="$asset_name" '$2 == asset { print $1; exit }' "$checksums_file"
}

sha256_file() {
  file_path="$1"
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file_path" | awk '{print $1}'
    return
  fi
  sha256sum "$file_path" | awk '{print $1}'
}

install_dir() {
  if [ "${INSTALL_LOCAL:-0}" = "1" ]; then
    printf "%s/.local/bin\n" "$HOME"
    return
  fi
  printf "/usr/local/bin\n"
}

download_and_install() {
  version="$1"
  tag=$(version_tag "$version")
  clean_version=$(printf "%s" "$tag" | sed 's/^v//')
  os_name=$(normalize_os "$(uname -s)")
  arch_name=$(normalize_arch "$(uname -m)")
  asset_name=$(release_asset_name "$clean_version" "$os_name" "$arch_name")

  temp_dir=$(mktemp -d)
  archive_path="$temp_dir/$asset_name"
  checksums_path="$temp_dir/checksums.txt"
  extract_dir="$temp_dir/extract"
  mkdir -p "$extract_dir"

  curl -fsSL -o "$archive_path" "https://github.com/$REPO/releases/download/$tag/$asset_name"
  curl -fsSL -o "$checksums_path" "https://github.com/$REPO/releases/download/$tag/checksums.txt"

  expected=$(checksum_for "$checksums_path" "$asset_name")
  actual=$(sha256_file "$archive_path")
  if [ -z "$expected" ] || [ "$actual" != "$expected" ]; then
    printf >&2 "checksum verification failed for %s\n" "$asset_name"
    rm -rf "$temp_dir"
    exit 1
  fi

  tar -xzf "$archive_path" -C "$extract_dir"

  target_dir=$(install_dir)
  if [ "$target_dir" = "/usr/local/bin" ] && [ ! -w "$target_dir" ]; then
    if command -v sudo >/dev/null 2>&1; then
      sudo mkdir -p "$target_dir"
      sudo install -m 0755 "$extract_dir/$BINARY" "$target_dir/$BINARY"
    else
      printf >&2 "sudo is required to install to %s\n" "$target_dir"
      rm -rf "$temp_dir"
      exit 1
    fi
  else
    mkdir -p "$target_dir"
    install -m 0755 "$extract_dir/$BINARY" "$target_dir/$BINARY"
  fi

  rm -rf "$temp_dir"
  printf "installed %s %s to %s/%s\n" "$BINARY" "$clean_version" "$target_dir" "$BINARY"
}

main() {
  version=""
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --version)
        version="$2"
        shift 2
        ;;
      --local)
        INSTALL_LOCAL=1
        shift
        ;;
      *)
        printf >&2 "unknown option: %s\n" "$1"
        exit 1
        ;;
    esac
  done

  if [ -z "$version" ]; then
    version=$(latest_version)
  fi
  download_and_install "$version"
}

if [ "${PETTI_INSTALL_LIB:-0}" != "1" ]; then
  main "$@"
fi
