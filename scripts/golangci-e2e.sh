#!/usr/bin/env bash
# Builds a custom golangci-lint with the jsonv2audit module plugin from this
# checkout and runs it on cmd/jsonv2-audit-vet/testdata/mod, checking the
# issue count with no settings, with skip, and that an unknown rule fails.
#
# Usage: scripts/golangci-e2e.sh [path to golangci-lint]
# Needs network on the first run (golangci-lint custom clones its source).
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
gcl=${1:-golangci-lint}
e2e="$root/golangci/testdata/e2e"
cd "$root"

"$gcl" custom  # reads .custom-gcl.yml in the repo root
bin="$root/bin/gcl-jsonv2audit"
[ -x "$bin" ] || bin="$bin.exe"

count() {
  # Prints the number of jsonv2audit issues; golangci-lint exits 1 on issues.
  (cd cmd/jsonv2-audit-vet/testdata/mod &&
    GOWORK=off "$bin" run -c "$e2e/$1" --output.text.path stdout ./... || true) |
    grep -c '(jsonv2audit)$' || true
}

fail=0
check() {
  if [ "$2" = "$3" ]; then echo "ok   $1: $2"; else echo "FAIL $1: got $2, want $3"; fail=1; fi
}

check "no settings" "$(count all.yml)" 2
check "skip nil-slice" "$(count skip.yml)" 1

out=$(cd cmd/jsonv2-audit-vet/testdata/mod && GOWORK=off "$bin" run -c "$e2e/bad.yml" ./... 2>&1 || true)
if grep -q 'unknown rule "nil-slcie"' <<<"$out"; then
  echo "ok   unknown rule fails"
else
  echo "FAIL unknown rule: $out"
  fail=1
fi
exit $fail
