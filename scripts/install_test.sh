#!/bin/sh
set -eu

PETTI_INSTALL_LIB=1
. "$(dirname "$0")/install.sh"

assert_eq() {
  if [ "$1" != "$2" ]; then
    echo "expected '$2', got '$1'" >&2
    exit 1
  fi
}

assert_eq "$(normalize_os Darwin)" "darwin"
assert_eq "$(normalize_os Linux)" "linux"
assert_eq "$(normalize_arch x86_64)" "amd64"
assert_eq "$(normalize_arch arm64)" "arm64"
assert_eq "$(normalize_arch aarch64)" "arm64"
assert_eq "$(release_asset_name 0.6.3 darwin arm64)" "petti_0.6.3_darwin_arm64.tar.gz"
assert_eq "$(version_tag 0.6.3)" "v0.6.3"
assert_eq "$(version_tag v0.6.3)" "v0.6.3"
