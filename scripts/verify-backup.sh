#!/usr/bin/env bash
#
# Restore-verify an Affluena Postgres backup — a backup you never restored is
# a hope, not a backup.
#
# Usage:
#     bash scripts/verify-backup.sh [dump-file]     # default: newest *.dump in $BACKUP_DIR
#
# Restores the dump into a THROWAWAY Postgres container (same image as the
# live one, never touching it or its volume), then sanity-checks the result:
# schema_migrations has rows and the core money tables exist. Exit 0 = the
# backup is restorable. Run weekly from cron (install-backup-cron.sh) or by
# hand before risky maintenance.
set -euo pipefail

DEPLOY_ROOT="${DEPLOY_ROOT:-/opt/affluena}"
BACKUP_DIR="${BACKUP_DIR:-$DEPLOY_ROOT/backups}"
PG_SERVICE="${PG_SERVICE:-postgres}"
DB_NAME="${DB_NAME:-affluena_api}"
DB_USER="${DB_USER:-affluena_api}"
FALLBACK_IMAGE="${FALLBACK_IMAGE:-postgres:17-alpine}"

SUDO=""
if ! docker ps >/dev/null 2>&1; then
  SUDO="sudo -n"
fi

log() { printf '[%s] verify: %s\n' "$(date '+%F %T')" "$1"; }

# `|| true` keeps an empty backup dir from tripping set -o pipefail inside the
# command substitution — the explicit diagnostic below must own that case.
DUMP="${1:-$(ls -1t "$BACKUP_DIR/${DB_NAME}"-*.dump 2>/dev/null | head -n1 || true)}"
if [ -z "$DUMP" ] || [ ! -f "$DUMP" ]; then
  log "!! no dump found (looked for $BACKUP_DIR/${DB_NAME}-*.dump)"
  exit 1
fi
log "verifying $DUMP"

# Reap throwaways left by runs killed before their trap fired (SIGKILL,
# reboot); each one holds a restored copy of the production DB.
stale="$($SUDO docker ps -aq --filter "name=affluena-restore-verify-" || true)"
if [ -n "$stale" ]; then
  # shellcheck disable=SC2086 — container IDs are single tokens, splitting is intended
  $SUDO docker rm -fv $stale >/dev/null 2>&1 || true
fi

# Restore with the same Postgres image the live container runs (a dump from a
# newer server may not restore on an older one) — resolved by compose-service
# label first (exact), then name filter; fall back when nothing runs.
IMAGE="$($SUDO docker ps --filter "label=com.docker.compose.service=${PG_SERVICE}" --format '{{.Image}}' | head -n1)"
if [ -z "$IMAGE" ]; then
  IMAGE="$($SUDO docker ps --filter "name=${PG_SERVICE}" --format '{{.Image}}' | head -n1)"
fi
IMAGE="${IMAGE:-$FALLBACK_IMAGE}"

NAME="affluena-restore-verify-$$"
# -v drops the anonymous PGDATA volume with the container — without it every
# weekly run would leak a full restored copy of the production financial DB
# into /var/lib/docker. The tmpfs mount goes further: the restored data only
# ever lives in RAM (the DB is small), never on disk.
cleanup() { $SUDO docker rm -fv "$NAME" >/dev/null 2>&1 || true; }
trap cleanup EXIT

log "starting throwaway $IMAGE as $NAME"
$SUDO docker run -d --name "$NAME" \
  --tmpfs /var/lib/postgresql/data \
  -e POSTGRES_DB="$DB_NAME" \
  -e POSTGRES_USER="$DB_USER" \
  -e POSTGRES_PASSWORD=restore-verify \
  "$IMAGE" >/dev/null

# Readiness MUST be probed over TCP: the official image's entrypoint first
# boots a TEMPORARY init server that answers only on the unix socket
# (listen_addresses=''), then shuts it down — a socket probe can pass during
# that phase and the restore then dies mid-flight. TCP only answers once the
# final server is up; the SELECT 1 confirms the target DB really accepts work.
ready=""
for _ in $(seq 1 60); do
  if $SUDO docker exec "$NAME" pg_isready -h 127.0.0.1 -U "$DB_USER" -d "$DB_NAME" >/dev/null 2>&1 \
    && $SUDO docker exec -e PGPASSWORD=restore-verify "$NAME" psql -h 127.0.0.1 -U "$DB_USER" -d "$DB_NAME" -tAc 'SELECT 1' >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 2
done
if [ -z "$ready" ]; then
  log "!! throwaway postgres never became ready"
  exit 1
fi

log "restoring dump (pg_restore --exit-on-error)"
$SUDO docker exec -i "$NAME" pg_restore -U "$DB_USER" -d "$DB_NAME" --no-owner --exit-on-error < "$DUMP"

psql_count() {
  $SUDO docker exec "$NAME" psql -U "$DB_USER" -d "$DB_NAME" -tAc "$1"
}

migrations="$(psql_count 'SELECT count(*) FROM schema_migrations')"
if [ "${migrations:-0}" -lt 1 ]; then
  log "!! restored DB has no schema_migrations rows — restore is unusable"
  exit 1
fi
log "schema_migrations: $migrations applied"

# Core money tables must exist (count queries fail loudly if a table is gone).
for table in users wallets transactions category_budgets; do
  rows="$(psql_count "SELECT count(*) FROM ${table}")"
  log "table ${table}: ${rows} row(s)"
done

log "PASS — $DUMP restores cleanly"
