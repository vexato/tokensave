#!/usr/bin/env sh
set -eu

repository="vexato/tokensave"
version="${TOKENSAVE_VERSION:-latest}"
install_dir="${TOKENSAVE_INSTALL_DIR:-$HOME/.local/bin}"
skill_dir="${TOKENSAVE_SKILL_DIR:-$HOME/.agents/skills/tokensave}"
legacy_skill_dir="$HOME/.codex/skills/tokensave"
install_skill="${TOKENSAVE_INSTALL_SKILL:-1}"

os="$(uname -s)"
arch="$(uname -m)"
case "$os" in
  Linux) platform_os="linux" ;;
  Darwin) platform_os="darwin" ;;
  *) echo "TokenSave does not support $os through this installer." >&2; exit 1 ;;
esac
case "$arch" in
  x86_64|amd64) platform_arch="amd64" ;;
  arm64|aarch64) platform_arch="arm64" ;;
  *) echo "TokenSave does not support CPU architecture $arch." >&2; exit 1 ;;
esac

asset="tokensave_${platform_os}_${platform_arch}.tar.gz"
if [ "$version" = "latest" ]; then
  download_url="https://github.com/$repository/releases/latest/download/$asset"
  checksums_url="https://github.com/$repository/releases/latest/download/checksums.txt"
else
  download_url="https://github.com/$repository/releases/download/$version/$asset"
  checksums_url="https://github.com/$repository/releases/download/$version/checksums.txt"
fi

tmpdir="$(mktemp -d)"
cleanup() { rm -rf "$tmpdir"; }
trap cleanup EXIT INT TERM

command -v curl >/dev/null 2>&1 || { echo "curl is required." >&2; exit 1; }
command -v tar >/dev/null 2>&1 || { echo "tar is required." >&2; exit 1; }

case "$install_skill" in
  1|true|TRUE|yes|YES) install_skill=true ;;
  0|false|FALSE|no|NO) install_skill=false ;;
  *) echo "TOKENSAVE_INSTALL_SKILL must be 0 or 1." >&2; exit 1 ;;
esac

curl --fail --location --silent --show-error "$download_url" -o "$tmpdir/$asset"
curl --fail --location --silent --show-error "$checksums_url" -o "$tmpdir/checksums.txt"

expected_checksum="$(awk -v asset="$asset" '$2 == asset { print $1; exit }' "$tmpdir/checksums.txt")"
[ -n "$expected_checksum" ] || { echo "checksums.txt does not contain an entry for $asset." >&2; exit 1; }
[ "${#expected_checksum}" -eq 64 ] || {
  echo "checksums.txt contains an invalid SHA-256 entry for $asset." >&2
  exit 1
}
case "$expected_checksum" in
  *[!0123456789abcdefABCDEF]*)
    echo "checksums.txt contains an invalid SHA-256 entry for $asset." >&2
    exit 1
    ;;
esac

if command -v sha256sum >/dev/null 2>&1; then
  actual_checksum="$(sha256sum "$tmpdir/$asset" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  actual_checksum="$(shasum -a 256 "$tmpdir/$asset" | awk '{print $1}')"
else
  echo "sha256sum or shasum is required to verify the download." >&2
  exit 1
fi

[ "$actual_checksum" = "$expected_checksum" ] || {
  echo "SHA-256 verification failed for $asset." >&2
  exit 1
}
printf 'Verified SHA-256 for %s\n' "$asset"

tar -xzf "$tmpdir/$asset" -C "$tmpdir"
mkdir -p "$install_dir"
install -m 0755 "$tmpdir/tokensave" "$install_dir/tokensave"
if [ "$install_skill" = true ]; then
  [ -d "$tmpdir/skills/tokensave" ] || {
    echo "Downloaded archive does not contain skills/tokensave/." >&2
    exit 1
  }
  if [ -d "$legacy_skill_dir" ] && [ "$legacy_skill_dir" != "$skill_dir" ]; then
    printf 'Legacy Codex Skill found at %s. It was not removed; migrate or remove it manually after verifying %s.\n' "$legacy_skill_dir" "$skill_dir" >&2
  fi
  mkdir -p "$skill_dir"
  cp -R "$tmpdir/skills/tokensave/." "$skill_dir/"
  printf 'Codex Skill installed in %s\n' "$skill_dir"
  printf 'Verify installed skills: codex /skills\n'
fi

printf 'TokenSave installed in %s\n' "$install_dir"
case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) printf 'Add it to PATH: export PATH="%s:$PATH"\n' "$install_dir" ;;
esac
