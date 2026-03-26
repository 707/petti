#!/bin/sh
set -eu

PETTI_TAP_LIB=1
. "$(dirname "$0")/update_tap_formula.sh"

assert_contains() {
  file_path="$1"
  expected="$2"
  if ! grep -Fq "$expected" "$file_path"; then
    echo "missing expected content: $expected" >&2
    exit 1
  fi
}

temp_dir=$(mktemp -d)
checksums_file="$temp_dir/checksums.txt"
formula_file="$temp_dir/petti.rb"

cat >"$checksums_file" <<'EOF'
aaa111  petti_0.6.4_darwin_amd64.tar.gz
bbb222  petti_0.6.4_darwin_arm64.tar.gz
ccc333  petti_0.6.4_linux_amd64.tar.gz
ddd444  petti_0.6.4_linux_arm64.tar.gz
EOF

write_formula "v0.6.4" "$checksums_file" "$formula_file"

assert_contains "$formula_file" 'version "0.6.4"'
assert_contains "$formula_file" 'petti_0.6.4_darwin_arm64.tar.gz'
assert_contains "$formula_file" 'sha256 "bbb222"'
assert_contains "$formula_file" 'petti_0.6.4_linux_amd64.tar.gz'
assert_contains "$formula_file" 'sha256 "ccc333"'
assert_contains "$formula_file" 'shell_output("#{bin}/petti --version")'

rm -rf "$temp_dir"
