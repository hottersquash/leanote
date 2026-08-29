#!/usr/bin/env bash
# Build linux/amd64 release package inside WSL.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${VERSION:-v2.8.1}"

# Prefer Windows host proxy if port 7890 is open.
for candidate in "127.0.0.1" "$(ip route show default 2>/dev/null | awk '{print $3; exit}')" "$(grep -m1 nameserver /etc/resolv.conf | awk '{print $2}')"; do
  [[ -z "$candidate" ]] && continue
  if (echo >/dev/tcp/${candidate}/7890) >/dev/null 2>&1; then
    export http_proxy="http://${candidate}:7890"
    export https_proxy="http://${candidate}:7890"
    export HTTP_PROXY="$http_proxy"
    export HTTPS_PROXY="$https_proxy"
    echo "[wsl-package] proxy=${http_proxy}"
    break
  fi
done

export PATH="/usr/local/go/bin:${HOME}/go/bin:/usr/bin:/bin"
export GOPROXY="${GOPROXY:-https://goproxy.cn,https://proxy.golang.org,direct}"

if ! command -v go >/dev/null 2>&1; then
  echo "[wsl-package] go not found; install with: sudo apt-get install -y golang-go" >&2
  exit 1
fi
go version

if ! command -v revel >/dev/null 2>&1; then
  echo "[wsl-package] installing revel CLI ..."
  go install github.com/revel/cmd/revel@v1.0.3
fi
revel version || true

cd "$ROOT"
export VERSION
export CGO_ENABLED="${CGO_ENABLED:-0}"
export GOOS=linux
export GOARCH=amd64
bash sh/package-release.sh
