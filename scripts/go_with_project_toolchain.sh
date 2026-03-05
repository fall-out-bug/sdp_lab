#!/usr/bin/env bash
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
GO_MOD="$ROOT/go.mod"

if [[ ! -f "$GO_MOD" ]]; then
  echo "go.mod not found at $GO_MOD" >&2
  exit 1
fi

required_minor="$(awk '/^go [0-9]+\.[0-9]+/{print $2; exit}' "$GO_MOD")"
if [[ -z "$required_minor" ]]; then
  echo "unable to read required Go version from go.mod" >&2
  exit 1
fi

required_patch="${required_minor}.0"
os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch_raw="$(uname -m)"
case "$arch_raw" in
  x86_64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *)
    echo "unsupported architecture: $arch_raw" >&2
    exit 1
    ;;
esac

toolchain_root="$ROOT/.cache/go-toolchains"
toolchain_dir="$toolchain_root/go${required_patch}"
go_bin="$toolchain_dir/go/bin/go"

have_compatible_go() {
  if ! command -v go >/dev/null 2>&1; then
    return 1
  fi

  local current
  current="$(go env GOVERSION 2>/dev/null || true)"
  if [[ "$current" =~ ^go([0-9]+)\.([0-9]+)(\.[0-9]+)?$ ]]; then
    local major="${BASH_REMATCH[1]}"
    local minor="${BASH_REMATCH[2]}"
    local req_major req_minor
    req_major="${required_minor%%.*}"
    req_minor="${required_minor##*.}"
    if (( major > req_major )) || (( major == req_major && minor >= req_minor )); then
      return 0
    fi
  fi
  return 1
}

if have_compatible_go; then
  go_cmd="$(command -v go)"
else
  mkdir -p "$toolchain_root"
  if [[ ! -x "$go_bin" ]]; then
    archive="$toolchain_root/go${required_patch}.${os}-${arch}.tar.gz"
    url="https://go.dev/dl/go${required_patch}.${os}-${arch}.tar.gz"
    echo "Installing Go ${required_patch} toolchain into $toolchain_dir" >&2
    if command -v curl >/dev/null 2>&1; then
      curl -fsSL "$url" -o "$archive"
    elif command -v wget >/dev/null 2>&1; then
      wget -qO "$archive" "$url"
    else
      echo "curl or wget is required to download Go toolchain" >&2
      exit 1
    fi
    rm -rf "$toolchain_dir"
    mkdir -p "$toolchain_dir"
    tar -xzf "$archive" -C "$toolchain_dir"
  fi
  go_cmd="$go_bin"
fi

if [[ $# -eq 0 ]]; then
  "$go_cmd" version
  exit 0
fi

"$go_cmd" "$@"
