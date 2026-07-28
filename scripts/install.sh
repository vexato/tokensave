#!/usr/bin/env sh
set -eu

repository="vexato/tokensave"
version="${TOKENSAVE_VERSION:-latest}"
install_dir="${TOKENSAVE_INSTALL_DIR:-$HOME/.local/bin}"

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
else
  download_url="https://github.com/$repository/releases/download/$version/$asset"
fi

tmpdir="$(mktemp -d)"
cleanup() { rm -rf "$tmpdir"; }
trap cleanup EXIT INT TERM

command -v curl >/dev/null 2>&1 || { echo "curl is required." >&2; exit 1; }
command -v tar >/dev/null 2>&1 || { echo "tar is required." >&2; exit 1; }

curl --fail --location --silent --show-error "$download_url" -o "$tmpdir/$asset"
tar -xzf "$tmpdir/$asset" -C "$tmpdir"
mkdir -p "$install_dir"
install -m 0755 "$tmpdir/tokensave" "$install_dir/tokensave"

printf 'TokenSave installed in %s\n' "$install_dir"
case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) printf 'Add it to PATH: export PATH="%s:$PATH"\n' "$install_dir" ;;
esac
