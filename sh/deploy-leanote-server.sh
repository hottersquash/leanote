#!/usr/bin/env bash
set -euo pipefail

LEANOTE_ROOT=/home/byan/leanote
PKG=/tmp/leanote-linux-amd64.tar.gz
TMP=/tmp/leanote-deploy
START=/home/byan/start.sh

rm -rf "$TMP"
mkdir -p "$TMP"
tar -xzf "$PKG" -C "$TMP"

install -m 755 "$TMP/leanote" "$LEANOTE_ROOT/bin/leanote-linux-amd64"
echo "[deploy] updated binary"

for dir in public messages app/views mongodb_backup; do
  if [[ -d "$TMP/$dir" ]]; then
    rm -rf "$LEANOTE_ROOT/$dir"
    cp -a "$TMP/$dir" "$LEANOTE_ROOT/$dir"
    echo "[deploy] updated $dir/"
  fi
done

if [[ -d "$TMP/src" ]]; then
  rm -rf "$LEANOTE_ROOT/bin/src/github.com/leanote"
  mkdir -p "$LEANOTE_ROOT/bin/src/github.com/leanote"
  cp -a "$TMP/src/github.com/leanote/leanote" "$LEANOTE_ROOT/bin/src/github.com/leanote/" 2>/dev/null || true
  echo "[deploy] merged packaged src views"
fi

link_dir="$LEANOTE_ROOT/bin/src/github.com/leanote"
mkdir -p "$link_dir"
rm -rf "$link_dir/leanote"
ln -sfn "$LEANOTE_ROOT" "$link_dir/leanote"
echo "[deploy] linked app root -> $link_dir/leanote"

rm -rf "$TMP" "$PKG"
"$START" restart
"$START" status
curl -sf http://127.0.0.1:9002/ >/dev/null && echo "[deploy] HTTP 9002 OK" || echo "[deploy] HTTP 9002 FAIL"
ls -la "$LEANOTE_ROOT/bin/leanote-linux-amd64"
test -f "$LEANOTE_ROOT/app/views/member/blog/background.html" && echo "[deploy] background.html present"
