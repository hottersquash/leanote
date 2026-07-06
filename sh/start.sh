#!/usr/bin/env bash
# Unified start/stop for Leanote + MongoDB (split layout)
set -euo pipefail

MONGO_ROOT="${MONGO_ROOT:-/home/byan/mongo}"
LEANOTE_ROOT="${LEANOTE_ROOT:-/home/byan/leanote}"
MONGO_BIN="$MONGO_ROOT/bin"
DB_PATH="$MONGO_ROOT/db"
LOG_DIR="$MONGO_ROOT/logs"
RUN_DIR="$MONGO_ROOT/run"
MONGOD_PID="$RUN_DIR/mongod.pid"
LEANOTE_PID="$RUN_DIR/leanote.pid"
MONGOD_LOG="$LOG_DIR/mongod.log"
LEANOTE_LOG="$LOG_DIR/leanote.log"
MONGO_PORT="${MONGO_PORT:-27017}"
LEANOTE_PORT="${LEANOTE_PORT:-9002}"

export PATH="$MONGO_BIN:$PATH"

log() { echo "[$(date '+%H:%M:%S')] $*"; }

ensure_dirs() {
  mkdir -p "$DB_PATH" "$LOG_DIR" "$RUN_DIR" "$LEANOTE_ROOT/logs"
}

stop_mongo() {
  if [[ -f "$MONGOD_PID" ]]; then
    local pid
    pid="$(cat "$MONGOD_PID")"
    if kill -0 "$pid" 2>/dev/null; then
      log "Stopping MongoDB (pid $pid)..."
      kill "$pid" || true
      for _ in $(seq 1 10); do
        kill -0 "$pid" 2>/dev/null || break
        sleep 1
      done
      kill -9 "$pid" 2>/dev/null || true
    fi
    rm -f "$MONGOD_PID"
  fi
}

stop_leanote() {
  if [[ -f "$LEANOTE_PID" ]]; then
    local pid
    pid="$(cat "$LEANOTE_PID")"
    if kill -0 "$pid" 2>/dev/null; then
      log "Stopping Leanote (pid $pid)..."
      kill "$pid" || true
      sleep 2
      kill -9 "$pid" 2>/dev/null || true
    fi
    rm -f "$LEANOTE_PID"
  fi
  pkill -f "leanote-linux-amd64" 2>/dev/null || true
}

wait_mongo() {
  for _ in $(seq 1 30); do
    if ss -tln 2>/dev/null | grep -q "127.0.0.1:${MONGO_PORT} "; then
      return 0
    fi
    sleep 1
  done
  echo "MongoDB not listening on 127.0.0.1:${MONGO_PORT}" >&2
  return 1
}

wait_leanote() {
  for _ in $(seq 1 30); do
    if ss -tln 2>/dev/null | grep -q ":${LEANOTE_PORT} "; then
      return 0
    fi
    sleep 1
  done
  echo "Leanote not listening on :${LEANOTE_PORT}" >&2
  tail -30 "$LEANOTE_LOG" >&2 || true
  return 1
}

start_mongo() {
  ensure_dirs
  if [[ -f "$MONGOD_PID" ]] && kill -0 "$(cat "$MONGOD_PID")" 2>/dev/null; then
    log "MongoDB already running"
    return 0
  fi

  local auth_flag=()
  if [[ "${MONGO_AUTH:-1}" == "1" ]]; then
    auth_flag=(--auth)
  fi

  log "Starting MongoDB..."
  mongod \
    --dbpath "$DB_PATH" \
    --bind_ip 127.0.0.1 \
    --port "$MONGO_PORT" \
    "${auth_flag[@]}" \
    --logpath "$MONGOD_LOG" \
    --fork \
    --pidfilepath "$MONGOD_PID"
  wait_mongo
  log "MongoDB started"
}

start_leanote() {
  local bin="$LEANOTE_ROOT/bin/leanote-linux-amd64"
  if [[ ! -x "$bin" && ! -f "$bin" ]]; then
    echo "Leanote binary not found: $bin" >&2
    exit 1
  fi

  if [[ -f "$LEANOTE_PID" ]] && kill -0 "$(cat "$LEANOTE_PID")" 2>/dev/null; then
    log "Leanote already running"
    return 0
  fi

  log "Starting Leanote..."
  cd "$LEANOTE_ROOT/bin"
  export GOPATH="$LEANOTE_ROOT/bin"
  local link_dir="$LEANOTE_ROOT/bin/src/github.com/leanote"
  mkdir -p "$link_dir"
  rm -rf "$link_dir/leanote"
  ln -sfn "$LEANOTE_ROOT" "$link_dir/leanote"

  nohup "$bin" -importPath github.com/leanote/leanote \
    -srcPath "$LEANOTE_ROOT/bin/src" \
    -runMode prod \
    > "$LEANOTE_LOG" 2>&1 &
  echo $! > "$LEANOTE_PID"
  wait_leanote
  log "Leanote started on :${LEANOTE_PORT}"
}

cmd="${1:-start}"
case "$cmd" in
  start)
    start_mongo
    start_leanote
    ;;
  stop)
    stop_leanote
    stop_mongo
    ;;
  restart)
    stop_leanote
    stop_mongo
    sleep 2
    start_mongo
    start_leanote
    ;;
  status)
    ss -tln | grep -E "${MONGO_PORT}|${LEANOTE_PORT}" || true
    ;;
  *)
    echo "Usage: $0 {start|stop|restart|status}" >&2
    exit 1
    ;;
esac
