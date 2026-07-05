#!/usr/bin/env bash
#
# Postgres backup for the Affluena VPS (and any docker-hosted instance).
#
# Usage:
#     bash scripts/backup-db.sh [label]        # label: daily (default) | pre-deploy | manual…
#
# Dumps the affluena_api database from the running Postgres container with
# `pg_dump -Fc` (compressed custom format, restorable with pg_restore) into
# $BACKUP_DIR, verifies the archive is readable, snapshots the deploy config
# (compose + .env + nginx — the secrets you'd need to rebuild the host), then
# applies retention. Called nightly from cron (installed by
# install-backup-cron.sh) and as the pre-migration safety net by the deploy
# workflow (label "pre-deploy").
#
# Override any default with env vars:
#     DEPLOY_ROOT=/opt/affluena BACKUP_DIR=… PG_SERVICE=postgres bash scripts/backup-db.sh
#
# Optional off-site copy: if `rclone` is installed and BACKUP_RCLONE_REMOTE is
# set (e.g. "b2:affluena-backups"), every new dump is copied there too.
set -euo pipefail

LABEL="${1:-daily}"
DEPLOY_ROOT="${DEPLOY_ROOT:-/opt/affluena}"
BACKUP_DIR="${BACKUP_DIR:-$DEPLOY_ROOT/backups}"
CONFIG_DIR="${CONFIG_DIR:-$DEPLOY_ROOT/deploy}"
PG_SERVICE="${PG_SERVICE:-postgres}"
DB_NAME="${DB_NAME:-affluena_api}"
DB_USER="${DB_USER:-affluena_api}"
KEEP_DAILY_DAYS="${KEEP_DAILY_DAYS:-14}"
KEEP_PRE_DEPLOY="${KEEP_PRE_DEPLOY:-10}"
MIN_FREE_MB="${MIN_FREE_MB:-200}"

# Use sudo only when the current user can't talk to the docker daemon
# (same convention as seed-prod.sh / deploy.yml).
SUDO=""
if ! docker ps >/dev/null 2>&1; then
  SUDO="sudo -n"
fi

log() { printf '[%s] backup: %s\n' "$(date '+%F %T')" "$1"; }

# Backups contain financial data + (config tar) secrets — never group/world readable.
umask 077
mkdir -p "$BACKUP_DIR"

# Refuse to fill the disk: a full disk would take Postgres itself down.
free_mb="$(df -Pm "$BACKUP_DIR" | awk 'NR==2 {print $4}')"
if [ "${free_mb:-0}" -lt "$MIN_FREE_MB" ]; then
  log "!! only ${free_mb}MB free on $BACKUP_DIR (need ${MIN_FREE_MB}MB) — aborting"
  exit 1
fi

# Find the Postgres container by its compose SERVICE label first (exact match,
# immune to substring-name collisions like a "postgres-exporter" sidecar), then
# fall back to the name filter for non-compose setups.
PG_CONTAINER="$($SUDO docker ps --filter "label=com.docker.compose.service=${PG_SERVICE}" --format '{{.Names}}' | head -n1)"
if [ -z "$PG_CONTAINER" ]; then
  PG_CONTAINER="$($SUDO docker ps --filter "name=${PG_SERVICE}" --format '{{.Names}}' | head -n1)"
fi
if [ -z "$PG_CONTAINER" ]; then
  log "!! no running container matching service/name '${PG_SERVICE}' — is the stack up?"
  exit 1
fi

stamp="$(date +%Y%m%d-%H%M%S)"
dump="$BACKUP_DIR/${DB_NAME}-${LABEL}-${stamp}.dump"

log "dumping $DB_NAME from $PG_CONTAINER ($LABEL) -> $dump"
# Write to a .part file first so a failed/interrupted dump can never be
# mistaken for a good backup. The trap covers the set -e abort paths (a failed
# docker exec/pg_dump) so a dead .part never lingers; it is disarmed after the
# rename. Stale .part files from SIGKILL/power-loss (where traps can't run) are
# swept age-gated in the retention section below.
trap 'rm -f -- "$dump.part"' EXIT
$SUDO docker exec "$PG_CONTAINER" pg_dump -U "$DB_USER" -Fc "$DB_NAME" > "$dump.part"

if [ ! -s "$dump.part" ]; then
  log "!! dump is empty"
  exit 1
fi
# Integrity gate: the archive TOC must parse — catches header/TOC corruption
# and outright garbage. (Full restorability, incl. the data section, is proven
# separately by the weekly verify-backup.sh restore.)
if ! $SUDO docker exec -i "$PG_CONTAINER" pg_restore --list < "$dump.part" >/dev/null; then
  log "!! dump failed pg_restore --list integrity check"
  exit 1
fi
mv "$dump.part" "$dump"
trap - EXIT
chmod 600 "$dump"
log "dump OK: $(du -h "$dump" | cut -f1) $dump"

# Snapshot the deploy config (docker-compose.prod.yml, .env with
# POSTGRES_PASSWORD/JWT_SECRET, nginx). Losing these with an intact DB volume
# still locks you out — they belong in the same safety net. Non-fatal: a
# missing config dir must never block the DB backup.
if [ -d "$CONFIG_DIR" ]; then
  cfg="$BACKUP_DIR/deploy-config-${LABEL}-${stamp}.tar.gz"
  if tar -czf "$cfg" -C "$(dirname "$CONFIG_DIR")" "$(basename "$CONFIG_DIR")" 2>/dev/null; then
    chmod 600 "$cfg"
    log "config snapshot OK: $cfg"
  else
    rm -f "$cfg"
    log "!! config snapshot failed (non-fatal)"
  fi
fi

# Retention. Daily dumps age out by mtime; pre-deploy dumps keep the newest N
# (they arrive in bursts, so age-based pruning would be wrong for them).
find "$BACKUP_DIR" -maxdepth 1 -name "${DB_NAME}-daily-*.dump" -mtime +"$KEEP_DAILY_DAYS" -delete
find "$BACKUP_DIR" -maxdepth 1 -name "deploy-config-daily-*.tar.gz" -mtime +"$KEEP_DAILY_DAYS" -delete
# Sweep .part leftovers from runs killed before their trap could fire
# (SIGKILL/power loss). Age-gated so a concurrently running dump's live .part
# is never touched.
find "$BACKUP_DIR" -maxdepth 1 -name "*.part" -mmin +60 -delete

# Keep the newest $1 of the files in $2… (glob already expanded by the caller
# under nullglob — an empty match is a no-op, NOT an error, so the very first
# run on a fresh host can't fail here). The embedded YYYYmmdd-HHMMSS stamp
# makes lexical order == chronological order; no `ls` parsing, no pipelines
# that trip `set -o pipefail` on a missing glob.
prune_keep_newest() {
  local keep="$1"
  shift
  [ "$#" -le "$keep" ] && return 0
  local sorted
  sorted="$(printf '%s\n' "$@" | sort)"
  local drop=$(($# - keep))
  printf '%s\n' "$sorted" | head -n "$drop" | while IFS= read -r old; do
    rm -f -- "$old"
    log "retention: pruned $old"
  done
}
shopt -s nullglob
prune_keep_newest "$KEEP_PRE_DEPLOY" "$BACKUP_DIR/${DB_NAME}"-pre-deploy-*.dump
prune_keep_newest "$KEEP_PRE_DEPLOY" "$BACKUP_DIR"/deploy-config-pre-deploy-*.tar.gz
shopt -u nullglob

# Optional off-site copy (strongly recommended: local backups die with the disk).
if [ -n "${BACKUP_RCLONE_REMOTE:-}" ]; then
  if command -v rclone >/dev/null 2>&1; then
    if rclone copy "$dump" "$BACKUP_RCLONE_REMOTE" 2>/dev/null; then
      log "off-site copy OK -> $BACKUP_RCLONE_REMOTE"
    else
      log "!! off-site copy failed (local backup kept)"
    fi
  else
    log "!! BACKUP_RCLONE_REMOTE set but rclone is not installed"
  fi
fi

log "done ($LABEL). backups on disk: $(find "$BACKUP_DIR" -maxdepth 1 -name "${DB_NAME}-*.dump" | wc -l | tr -d ' ')"
