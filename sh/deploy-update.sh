#!/usr/bin/env bash
# Incremental deploy on server: pull latest code, rebuild binary, restart services.
set -euo pipefail

LEANOTE_ROOT="${LEANOTE_ROOT:-/home/byan/leanote}"
START_SCRIPT="${START_SCRIPT:-/home/byan/start.sh}"
BRANCH="${BRANCH:-master}"
TMP_PKG="${TMP_PKG:-/tmp/leanote-deploy-$$.tar.gz}"

log() { echo "[$(date '+%H:%M:%S')] $*"; }

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing command: $1" >&2
    exit 1
  fi
}

if [[ ! -d "$LEANOTE_ROOT" ]]; then
  echo "Leanote root not found: $LEANOTE_ROOT" >&2
  exit 1
fi

require_cmd git
require_cmd revel

cd "$LEANOTE_ROOT"

if [[ -d .git ]]; then
  log "Pulling latest code ($BRANCH)..."
  git fetch origin
  git checkout "$BRANCH"
  git pull --ff-only origin "$BRANCH"
else
  log "No git repo at $LEANOTE_ROOT; skip pull"
fi

log "Building production package..."
rm -f "$TMP_PKG"
revel package --run-mode=prod --target-path="$TMP_PKG" -a "$LEANOTE_ROOT"

log "Extracting package..."
TMP_DIR="$(mktemp -d /tmp/leanote-extract-XXXXXX)"
tar -xzf "$TMP_PKG" -C "$TMP_DIR"
EXTRACTED=( "$TMP_DIR"/leanote* )
SRC="${EXTRACTED[0]}"

for dir in conf public messages app/views mongodb_backup; do
  if [[ -d "$SRC/$dir" ]]; then
    log "Updating $dir/ ..."
    rm -rf "$LEANOTE_ROOT/$dir"
    cp -a "$SRC/$dir" "$LEANOTE_ROOT/$dir"
  fi
done

if [[ -f "$SRC/leanote" ]]; then
  log "Updating binary..."
  mkdir -p "$LEANOTE_ROOT/bin"
  install -m 755 "$SRC/leanote" "$LEANOTE_ROOT/bin/leanote-linux-amd64"
elif [[ -f "$SRC/bin/leanote-linux-amd64" ]]; then
  log "Updating packaged binary..."
  mkdir -p "$LEANOTE_ROOT/bin"
  install -m 755 "$SRC/bin/leanote-linux-amd64" "$LEANOTE_ROOT/bin/leanote-linux-amd64"
fi

rm -rf "$TMP_DIR" "$TMP_PKG"

if [[ -x "$START_SCRIPT" ]]; then
  log "Restarting services..."
  "$START_SCRIPT" restart
else
  log "Start script not found: $START_SCRIPT"
  exit 1
fi

log "Deploy complete."
