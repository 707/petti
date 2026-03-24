#!/bin/sh
set -eu

output="$(GOCACHE="${GOCACHE:-$(pwd)/.gocache}" GOMODCACHE="${GOMODCACHE:-$(pwd)/.gomodcache}" go test -cover ./...)"
printf '%s\n' "$output"

printf '%s\n' "$output" | awk '
/coverage: / {
  split($0, parts, "coverage: ")
  split(parts[2], rest, "%")
  if (rest[1] != "100.0") {
    print "coverage check failed for: " $0 > "/dev/stderr"
    exit 1
  }
}
'
