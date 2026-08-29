#!/usr/bin/env bash
# Build a production linux/amd64 release tarball for GitHub Releases.
# Usage: VERSION=v1.2.3 bash sh/package-release.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${VERSION:-dev}"
# Strip a leading "v" for the filename while keeping the release tag as-is upstream.
VERSION_SLUG="${VERSION#v}"
OUT_DIR="${OUT_DIR:-$ROOT/dist}"
OS="${GOOS:-linux}"
ARCH="${GOARCH:-amd64}"
ASSET="leanote-${OS}-${ARCH}-${VERSION_SLUG}.bin.tar.gz"
TMP_TGZ="$OUT_DIR/.leanote-package.tar.gz"

mkdir -p "$OUT_DIR"
cd "$ROOT"

if ! command -v revel >/dev/null 2>&1; then
  echo "revel not found in PATH; install with: go install github.com/revel/cmd/revel@v1.0.3" >&2
  exit 1
fi

echo "[package] building ${OS}/${ARCH} package (version=${VERSION_SLUG})..."
rm -f "$TMP_TGZ" "$OUT_DIR/$ASSET"

# Native or cross-compile via GOOS/GOARCH (CI runs on ubuntu-latest = linux/amd64).
export GOOS="$OS"
export GOARCH="$ARCH"
export CGO_ENABLED="${CGO_ENABLED:-0}"

revel package --run-mode=prod --target-path="$TMP_TGZ" -a "$ROOT"
mv "$TMP_TGZ" "$OUT_DIR/$ASSET"

echo "[package] wrote $OUT_DIR/$ASSET"
ls -lh "$OUT_DIR/$ASSET"
