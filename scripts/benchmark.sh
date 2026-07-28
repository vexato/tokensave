#!/usr/bin/env sh
set -eu

usage() { printf '%s\n' 'Usage: scripts/benchmark.sh [--output FILE]'; }

output=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --output) [ "$#" -ge 2 ] || { usage >&2; exit 2; }; output="$2"; shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *) usage >&2; exit 2 ;;
  esac
done

for command in awk go mktemp sh wc; do
  command -v "$command" >/dev/null 2>&1 || { printf '%s is required.\n' "$command" >&2; exit 1; }
done

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
tmpdir=$(mktemp -d "${TMPDIR:-/tmp}/tokensave-benchmark.XXXXXX")
cleanup() { rm -rf "$tmpdir"; }
trap cleanup EXIT INT TERM

if [ -n "${TOKENSAVE_BIN:-}" ]; then
  tokensave_bin=$TOKENSAVE_BIN
  [ -x "$tokensave_bin" ] || { printf 'TOKENSAVE_BIN is not executable: %s\n' "$tokensave_bin" >&2; exit 1; }
else
  tokensave_bin=$tmpdir/tokensave
  (cd "$repo_root" && go build -o "$tokensave_bin" ./cmd/tokensave)
fi

TOKENSAVE_HOME=$tmpdir/runs
export TOKENSAVE_HOME
report=$tmpdir/benchmark.md
cat >"$report" <<'EOF'
# TokenSave benchmark

This local comparison uses deterministic shell output. Results vary with the wrapped command, selected parser, configuration, operating system, and TokenSave version; line and byte reduction are not token-reduction measurements.

| Scenario | Raw lines | Summary lines | Line reduction | Raw bytes | Summary bytes | Byte reduction | Exit code preserved |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
EOF

run_scenario() {
  name=$1
  exit_code=$2
  raw=$tmpdir/$name.raw
  summary=$tmpdir/$name.summary
  set +e
  sh -c 'i=1; while [ "$i" -le 120 ]; do printf "fixture line %03d: deterministic output\n" "$i"; i=$((i + 1)); done; exit "$1"' sh "$exit_code" >"$raw" 2>&1
  raw_exit=$?
  "$tokensave_bin" sh -c 'i=1; while [ "$i" -le 120 ]; do printf "fixture line %03d: deterministic output\n" "$i"; i=$((i + 1)); done; exit "$1"' sh "$exit_code" >"$summary" 2>&1
  tokensave_exit=$?
  set -e
  [ "$raw_exit" -eq "$exit_code" ] || { printf 'raw %s scenario returned %s, expected %s\n' "$name" "$raw_exit" "$exit_code" >&2; exit 1; }
  [ "$tokensave_exit" -eq "$raw_exit" ] || { printf 'TokenSave did not preserve %s exit code: got %s, expected %s\n' "$name" "$tokensave_exit" "$raw_exit" >&2; exit 1; }
  raw_lines=$(wc -l <"$raw" | tr -d ' ')
  summary_lines=$(wc -l <"$summary" | tr -d ' ')
  raw_bytes=$(wc -c <"$raw" | tr -d ' ')
  summary_bytes=$(wc -c <"$summary" | tr -d ' ')
  line_reduction=$(awk -v raw="$raw_lines" -v summary="$summary_lines" 'BEGIN { printf "%.1f%%", (1 - summary / raw) * 100 }')
  byte_reduction=$(awk -v raw="$raw_bytes" -v summary="$summary_bytes" 'BEGIN { printf "%.1f%%", (1 - summary / raw) * 100 }')
  printf '| %s | %s | %s | %s | %s | %s | %s | Yes |\n' "$name" "$raw_lines" "$summary_lines" "$line_reduction" "$raw_bytes" "$summary_bytes" "$byte_reduction" >>"$report"
}

run_scenario success 0
run_scenario failure 7

if [ -n "$output" ]; then
  output_dir=$(dirname -- "$output")
  mkdir -p "$output_dir"
  cp "$report" "$output"
fi
cat "$report"
