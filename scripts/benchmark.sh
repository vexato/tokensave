#!/usr/bin/env sh
set -eu

usage() {
  printf '%s\n' 'Usage: scripts/benchmark.sh [--output FILE] [--json-output FILE]'
}

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
markdown_output=$repo_root/docs/benchmarks.md
json_output=$repo_root/docs/benchmarks.json

while [ "$#" -gt 0 ]; do
  case "$1" in
    --output)
      [ "$#" -ge 2 ] || { usage >&2; exit 2; }
      markdown_output=$2
      shift 2
      ;;
    --json-output)
      [ "$#" -ge 2 ] || { usage >&2; exit 2; }
      json_output=$2
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      usage >&2
      exit 2
      ;;
  esac
done

for command in go mktemp; do
  command -v "$command" >/dev/null 2>&1 || {
    printf '%s is required.\n' "$command" >&2
    exit 1
  }
done

if [ -n "${PYTHON:-}" ]; then
  python_bin=$PYTHON
elif command -v python3 >/dev/null 2>&1; then
  python_bin=python3
elif command -v python >/dev/null 2>&1; then
  python_bin=python
else
  printf '%s\n' 'Python 3 is required.' >&2
  exit 1
fi

tmpdir=$(mktemp -d "${TMPDIR:-/tmp}/tokensave-benchmark.XXXXXX")
cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT INT TERM

mkdir -p "$tmpdir/go-cache" "$tmpdir/go-tmp" "$tmpdir/state"
GOCACHE=$tmpdir/go-cache
GOTMPDIR=$tmpdir/go-tmp
GOTOOLCHAIN=local
GOPROXY=off
TOKENSAVE_HOME=$tmpdir/state
export GOCACHE GOTMPDIR GOTOOLCHAIN GOPROXY TOKENSAVE_HOME

if [ -n "${TOKENSAVE_BIN:-}" ]; then
  tokensave_bin=$TOKENSAVE_BIN
  [ -x "$tokensave_bin" ] || {
    printf 'TOKENSAVE_BIN is not executable: %s\n' "$tokensave_bin" >&2
    exit 1
  }
else
  case "$(go env GOOS)" in
    windows) executable_suffix=.exe ;;
    *) executable_suffix= ;;
  esac
  tokensave_bin=$tmpdir/tokensave$executable_suffix
  (cd "$repo_root" && go build -buildvcs=false -o "$tokensave_bin" ./cmd/tokensave)
fi

"$python_bin" "$repo_root/scripts/benchmark.py" \
  --tokensave-bin "$tokensave_bin" \
  --temp-dir "$tmpdir" \
  --markdown "$markdown_output" \
  --json "$json_output"
