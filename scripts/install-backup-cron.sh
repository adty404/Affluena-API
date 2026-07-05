#!/usr/bin/env bash
#
# Idempotently install (or refresh) the Affluena backup cron entries in the
# CURRENT user's crontab:
#
#   - nightly  backup-db.sh daily      (default 19:30 UTC = 02:30 WIB)
#   - weekly   verify-backup.sh        (default Sun 20:00 UTC = Mon 03:00 WIB)
#
# Safe to re-run: lines tagged with the marker comment are replaced, everything
# else in the crontab is preserved. The deploy workflow runs this on every
# deploy so schedule changes ship with a merge — no manual VPS step.
#
# Override schedules/paths with env vars:
#     DAILY_SCHEDULE='0 3 * * *' bash scripts/install-backup-cron.sh
set -euo pipefail

DEPLOY_ROOT="${DEPLOY_ROOT:-/opt/affluena}"
REPO_DIR="${REPO_DIR:-$DEPLOY_ROOT/Affluena-API}"
LOG_DIR="${LOG_DIR:-$DEPLOY_ROOT/backups}"
DAILY_SCHEDULE="${DAILY_SCHEDULE:-30 19 * * *}"
VERIFY_SCHEDULE="${VERIFY_SCHEDULE:-0 20 * * 0}"
MARKER="# affluena-db-backup"

mkdir -p "$LOG_DIR"

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

# Keep every non-marker line the user may have; drop (then re-add) ours.
crontab -l 2>/dev/null | grep -vF "$MARKER" > "$tmp" || true

{
  printf '%s DEPLOY_ROOT=%s bash %s/scripts/backup-db.sh daily >> %s/backup.log 2>&1 %s\n' \
    "$DAILY_SCHEDULE" "$DEPLOY_ROOT" "$REPO_DIR" "$LOG_DIR" "$MARKER"
  printf '%s DEPLOY_ROOT=%s bash %s/scripts/verify-backup.sh >> %s/verify.log 2>&1 %s\n' \
    "$VERIFY_SCHEDULE" "$DEPLOY_ROOT" "$REPO_DIR" "$LOG_DIR" "$MARKER"
} >> "$tmp"

crontab "$tmp"

echo "backup cron installed for user $(id -un):"
crontab -l | grep -F "$MARKER"
