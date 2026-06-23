#!/usr/bin/env bash
# Server-side: install Leanote binary + MongoDB to split paths
set -euo pipefail

MONGO_ROOT="${MONGO_ROOT:-/home/byan/mongo}"
LEANOTE_ROOT="${LEANOTE_ROOT:-/home/byan/leanote}"
OLD_ROOT="/home/byan/leanote"
STAGING="${STAGING:-/tmp/leanote-binary-deploy}"
DB_USER="${DB_USER:-leanote}"
DB_PASS="${DB_PASS:?Set DB_PASS to the MongoDB leanote user password}"
LEANOTE_PORT="${LEANOTE_PORT:-9002}"
SITE_URL="${SITE_URL:-https://note.byan.fun}"
MONGO_AUTH="${MONGO_AUTH:-1}"

MIGRATE_TMP="/tmp/leanote-migrate-$$"
MIGRATE_DB=""
MIGRATE_CONF=""
MIGRATE_BACKUP=""

log() { echo "[$(date '+%H:%M:%S')] $*"; }

require_file() {
  if [[ ! -f "$1" ]]; then
    echo "Missing file: $1" >&2
    exit 1
  fi
}

cleanup() {
  rm -rf "$MIGRATE_TMP"
}
trap cleanup EXIT

stop_old_services() {
  if [[ -x "$OLD_ROOT/start.sh" ]]; then
    log "Stopping old services via $OLD_ROOT/start.sh ..."
    (cd "$OLD_ROOT" && ./start.sh stop) || true
  elif [[ -x /home/byan/start.sh ]]; then
    log "Stopping via /home/byan/start.sh ..."
    /home/byan/start.sh stop || true
  fi
  sleep 2
}

collect_old_data() {
  mkdir -p "$MIGRATE_TMP"

  if [[ -d "$OLD_ROOT/db" ]] && [[ -n "$(ls -A "$OLD_ROOT/db" 2>/dev/null)" ]]; then
    log "Backing up old MongoDB data from $OLD_ROOT/db ..."
    cp -a "$OLD_ROOT/db" "$MIGRATE_TMP/db"
    MIGRATE_DB="$MIGRATE_TMP/db"
  fi

  if [[ -f "$OLD_ROOT/leanote/conf/app.conf" ]]; then
    cp "$OLD_ROOT/leanote/conf/app.conf" "$MIGRATE_TMP/app.conf"
    MIGRATE_CONF="$MIGRATE_TMP/app.conf"
    log "Saved old app.conf (nested layout)"
  elif [[ -f "$OLD_ROOT/conf/app.conf" ]]; then
    cp "$OLD_ROOT/conf/app.conf" "$MIGRATE_TMP/app.conf"
    MIGRATE_CONF="$MIGRATE_TMP/app.conf"
    log "Saved old app.conf"
  fi

  if [[ -f "$OLD_ROOT/leanote.tag.gz" ]]; then
    cp "$OLD_ROOT/leanote.tag.gz" "$MIGRATE_TMP/leanote.tag.gz"
    MIGRATE_BACKUP="$MIGRATE_TMP/leanote.tag.gz"
    log "Saved backup leanote.tag.gz"
  fi
}

remove_old_layout() {
  log "Removing old layout at $OLD_ROOT ..."
  rm -rf "$OLD_ROOT"
}

install_mongo() {
  require_file "$STAGING"/mongodb-linux-x86_64-*.tgz
  local mongo_tgz
  mongo_tgz=( "$STAGING"/mongodb-linux-x86_64-*.tgz )
  mongo_tgz="${mongo_tgz[0]}"

  log "Installing MongoDB to $MONGO_ROOT ..."
  rm -rf "$MONGO_ROOT"
  mkdir -p "$MONGO_ROOT"
  tar -xzf "$mongo_tgz" -C "$MONGO_ROOT"

  local extracted
  extracted=( "$MONGO_ROOT"/mongodb-linux-x86_64-* )
  if [[ -d "${extracted[0]}" ]]; then
    mv "${extracted[0]}"/* "$MONGO_ROOT/"
    rmdir "${extracted[0]}"
  fi

  mkdir -p "$MONGO_ROOT/db" "$MONGO_ROOT/logs" "$MONGO_ROOT/run"
  log "MongoDB installed: $($MONGO_ROOT/bin/mongod --version | head -1)"
}

install_leanote() {
  require_file "$STAGING"/leanote-linux-amd64-*.bin.tar.gz
  local leanote_tgz
  leanote_tgz=( "$STAGING"/leanote-linux-amd64-*.bin.tar.gz )
  leanote_tgz="${leanote_tgz[0]}"

  log "Installing Leanote to $LEANOTE_ROOT ..."
  rm -rf "$LEANOTE_ROOT"
  mkdir -p "$(dirname "$LEANOTE_ROOT")"
  tar -xzf "$leanote_tgz" -C "$(dirname "$LEANOTE_ROOT")"

  if [[ ! -d "$LEANOTE_ROOT" ]]; then
    local candidate
    candidate=( "$(dirname "$LEANOTE_ROOT")"/leanote* )
    if [[ -d "${candidate[0]}" ]]; then
      mv "${candidate[0]}" "$LEANOTE_ROOT"
    else
      echo "Could not find extracted leanote directory" >&2
      exit 1
    fi
  fi
  log "Leanote installed at $LEANOTE_ROOT"
}

restore_db() {
  export PATH="$MONGO_ROOT/bin:$PATH"

  if [[ -n "$MIGRATE_DB" && -d "$MIGRATE_DB" ]]; then
    log "Restoring migrated database files ..."
    if command -v rsync >/dev/null 2>&1; then
      rsync -a "$MIGRATE_DB/" "$MONGO_ROOT/db/"
    else
      cp -a "$MIGRATE_DB/." "$MONGO_ROOT/db/"
    fi
    return 0
  fi

  if [[ -n "$MIGRATE_BACKUP" && -f "$MIGRATE_BACKUP" ]]; then
    log "Restoring from backup archive ..."
    local restore_dir="$MIGRATE_TMP/restore"
    mkdir -p "$restore_dir"
    tar -xzf "$MIGRATE_BACKUP" -C "$restore_dir"
    local dump_dir="$restore_dir"
    [[ -d "$restore_dir/leanote" ]] && dump_dir="$restore_dir/leanote"

    mongod --dbpath "$MONGO_ROOT/db" --bind_ip 127.0.0.1 --port 27017 \
      --logpath "$MONGO_ROOT/logs/mongod-restore.log" --fork \
      --pidfilepath "$MONGO_ROOT/run/mongod-restore.pid"
    sleep 3
    mongorestore -h 127.0.0.1 --port 27017 -d leanote --dir "$dump_dir" --drop
    kill "$(cat "$MONGO_ROOT/run/mongod-restore.pid")" 2>/dev/null || true
    rm -f "$MONGO_ROOT/run/mongod-restore.pid"
    sleep 2
    return 0
  fi

  if [[ -d "$LEANOTE_ROOT/mongodb_backup/leanote_install_data" ]]; then
    log "Importing initial Leanote data from binary package ..."
    mongod --dbpath "$MONGO_ROOT/db" --bind_ip 127.0.0.1 --port 27017 \
      --logpath "$MONGO_ROOT/logs/mongod-init.log" --fork \
      --pidfilepath "$MONGO_ROOT/run/mongod-init.pid"
    sleep 3
    mongorestore -h 127.0.0.1 --port 27017 -d leanote \
      --dir "$LEANOTE_ROOT/mongodb_backup/leanote_install_data"
    kill "$(cat "$MONGO_ROOT/run/mongod-init.pid")" 2>/dev/null || true
    rm -f "$MONGO_ROOT/run/mongod-init.pid"
    sleep 2
    return 0
  fi

  log "No existing data; starting with empty database"
}

mongo_eval() {
  local js="$1"
  if command -v mongosh >/dev/null 2>&1; then
    mongosh --quiet --eval "$js"
  else
    mongo --quiet --eval "$js"
  fi
}

setup_mongo_auth() {
  if [[ "$MONGO_AUTH" != "1" ]]; then
    return 0
  fi

  export PATH="$MONGO_ROOT/bin:$PATH"
  log "Configuring MongoDB auth user '$DB_USER' ..."
  mongod --dbpath "$MONGO_ROOT/db" --bind_ip 127.0.0.1 --port 27017 \
    --logpath "$MONGO_ROOT/logs/mongod-setup.log" --fork \
    --pidfilepath "$MONGO_ROOT/run/mongod-setup.pid"
  sleep 3

  mongo_eval "
    db = db.getSiblingDB('leanote');
    try { db.dropUser('$DB_USER'); } catch(e) {}
    db.createUser({
      user: '$DB_USER',
      pwd: '$DB_PASS',
      roles: [{ role: 'readWrite', db: 'leanote' }]
    });
    print('user created');
  "

  kill "$(cat "$MONGO_ROOT/run/mongod-setup.pid")" 2>/dev/null || true
  rm -f "$MONGO_ROOT/run/mongod-setup.pid"
  sleep 2
}

configure_leanote() {
  local conf="$LEANOTE_ROOT/conf/app.conf"
  require_file "$conf"

  if [[ -n "$MIGRATE_CONF" && -f "$MIGRATE_CONF" ]]; then
    log "Restoring previous app.conf ..."
    cp "$MIGRATE_CONF" "$conf"
  fi

  cp "$conf" "$conf.bak.$(date +%Y%m%d%H%M%S)"
  sed -i "s/^db.host=.*/db.host=127.0.0.1/" "$conf"
  sed -i "s/^db.port=.*/db.port=27017/" "$conf"
  sed -i "s/^db.username=.*/db.username=$DB_USER/" "$conf"
  sed -i "s/^db.password=.*/db.password=$DB_PASS/" "$conf"

  if grep -q '^http.port=' "$conf"; then
    sed -i "s/^http.port=.*/http.port=$LEANOTE_PORT/" "$conf"
  else
    echo "http.port=$LEANOTE_PORT" >> "$conf"
  fi
  sed -i '/^httpport=/d' "$conf"
  if grep -q '^site.url=' "$conf"; then
    sed -i "s|^site.url=.*|site.url=${SITE_URL}|" "$conf"
  else
    echo "site.url=${SITE_URL}" >> "$conf"
  fi

  log "app.conf:"
  grep -E '^db\.(host|port|dbname|username|password)=|^http\.port=|^site\.url=' "$conf" || true
}

install_start_script() {
  require_file "$STAGING/start.sh"
  install -m 755 "$STAGING/start.sh" /home/byan/start.sh
  log "Installed /home/byan/start.sh"
}

verify() {
  log "Verification:"
  /home/byan/start.sh status || true
  curl -sf "http://127.0.0.1:${LEANOTE_PORT}/" >/dev/null && log "Leanote HTTP OK on :${LEANOTE_PORT}" || true
}

main() {
  stop_old_services
  collect_old_data
  remove_old_layout
  install_mongo
  install_leanote
  restore_db
  setup_mongo_auth
  configure_leanote
  install_start_script
  log "Starting services..."
  /home/byan/start.sh restart
  verify
  log "Done."
  log "  Leanote:  http://127.0.0.1:${LEANOTE_PORT}"
  log "  MongoDB:  127.0.0.1:27017  (data: $MONGO_ROOT/db)"
  log "  Manage:   /home/byan/start.sh {start|stop|restart|status}"
}

main "$@"
