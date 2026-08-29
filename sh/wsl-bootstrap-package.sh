#!/usr/bin/env bash
set -euo pipefail

ROOT="/mnt/c/Users/abc/Projects/leanote"
SUDO_PASS="${SUDO_PASS:-}"
VERSION="${VERSION:-v2.8.1}"
GO_TGZ="$ROOT/dist/go1.21.13.linux-amd64.tar.gz"

echo "[wsl] installing Go 1.21.13 to /usr/local/go ..."
echo "$SUDO_PASS" | sudo -S rm -rf /usr/local/go
echo "$SUDO_PASS" | sudo -S tar -C /usr/local -xzf "$GO_TGZ"

export PATH="/usr/local/go/bin:${HOME}/go/bin:/usr/bin:/bin"
export GOPROXY="${GOPROXY:-https://goproxy.cn,https://proxy.golang.org,direct}"
go version

echo "[wsl] (re)installing revel CLI with Go 1.21 ..."
go install github.com/revel/cmd/revel@v1.0.3
revel version || true

sed -i 's/\r$//' "$ROOT/sh/package-release.sh"
export PATH="/usr/local/go/bin:${HOME}/go/bin:/usr/bin:/bin"
export VERSION
export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64
bash "$ROOT/sh/package-release.sh"
