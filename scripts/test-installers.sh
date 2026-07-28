#!/usr/bin/env sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
installer=$repo_root/scripts/install.sh
bash -n "$installer"
grep -F 'skill_dir="${TOKENSAVE_SKILL_DIR:-$HOME/.agents/skills/tokensave}"' "$installer" >/dev/null
grep -F 'legacy_skill_dir="$HOME/.codex/skills/tokensave"' "$installer" >/dev/null
grep -F 'cp -R "$tmpdir/skills/tokensave/." "$skill_dir/"' "$installer" >/dev/null
grep -F 'SHA-256 verification failed' "$installer" >/dev/null
tmpdir=$(mktemp -d "${TMPDIR:-/tmp}/tokensave-installer-test.XXXXXX")
trap 'rm -rf "$tmpdir"' EXIT INT TERM
asset=tokensave_linux_amd64.tar.gz
printf '%s' fixture >"$tmpdir/$asset"
sha256sum "$tmpdir/$asset" | awk -v asset="$asset" '{ print $1 "  " asset }' >"$tmpdir/checksums.txt"
expected=$(awk -v asset="$asset" '$2 == asset { print $1; exit }' "$tmpdir/checksums.txt")
actual=$(sha256sum "$tmpdir/$asset" | awk '{print $1}')
[ "$expected" = "$actual" ]
printf '%s\n' 'Unix installer path and checksum checks passed.'
