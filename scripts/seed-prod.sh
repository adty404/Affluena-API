#!/usr/bin/env bash
#
# Seed the demo data (including the "Berbagi Dompet" sharing demo) on the VPS.
#
# Usage (on the server):
#     bash scripts/seed-prod.sh
#
# It will:
#   1. git pull the latest API code (so the seeder matches the live schema),
#   2. read POSTGRES_PASSWORD from the deploy .env,
#   3. auto-detect the Postgres container's docker network,
#   4. run `make seed` in a throwaway golang container on that network.
#
# sudo is used automatically when you are not root (docker + the root-owned
# repo/.env need it). Override any path/name with env vars, e.g.:
#     REPO_DIR=/srv/api ENV_FILE=/srv/deploy/.env bash scripts/seed-prod.sh
#
# SAFETY: re-runnable (idempotent). The seeder deletes & recreates ONLY the demo
# accounts (demo@/pengamat@/calon@, plus the pre-rebrand pasangan@). Real user
# data is never touched.
set -euo pipefail

REPO_DIR="${REPO_DIR:-/opt/affluena/Affluena-API}"
ENV_FILE="${ENV_FILE:-/opt/affluena/deploy/.env}"
PG_SERVICE="${PG_SERVICE:-postgres}"     # compose service name == network alias == DB host
GO_IMAGE="${GO_IMAGE:-golang:1.26-alpine}"

# Use sudo unless already root.
SUDO=""
if [ "$(id -u)" -ne 0 ]; then
  SUDO="sudo"
fi

log() { printf '\n==> %s\n' "$1"; }

log "Pulling latest API code in $REPO_DIR"
$SUDO git -C "$REPO_DIR" pull --ff-only origin master

log "Reading POSTGRES_PASSWORD from $ENV_FILE"
POSTGRES_PASSWORD="$(
  $SUDO grep -E '^[[:space:]]*POSTGRES_PASSWORD[[:space:]]*=' "$ENV_FILE" \
    | tail -n1 | cut -d= -f2- \
    | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//' \
          -e 's/^"//' -e 's/"$//' -e "s/^'//" -e "s/'\$//"
)"
if [ -z "${POSTGRES_PASSWORD}" ]; then
  echo "!! POSTGRES_PASSWORD not found in $ENV_FILE" >&2
  exit 1
fi

log "Detecting the Postgres container's docker network"
PG_CONTAINER="$($SUDO docker ps --filter "name=${PG_SERVICE}" --format '{{.Names}}' | head -n1)"
if [ -z "${PG_CONTAINER}" ]; then
  echo "!! No running container matched name '${PG_SERVICE}'. Check: $SUDO docker ps" >&2
  exit 1
fi
NETWORK="$($SUDO docker inspect -f '{{range $k,$v := .NetworkSettings.Networks}}{{$k}} {{end}}' "${PG_CONTAINER}" | awk '{print $1}')"
if [ -z "${NETWORK}" ]; then
  echo "!! Could not determine the network for container '${PG_CONTAINER}'." >&2
  exit 1
fi
echo "    container=${PG_CONTAINER}  network=${NETWORK}  host=${PG_SERVICE}"

log "Running the seeder (make seed) via ${GO_IMAGE}"
$SUDO docker run --rm \
  --network "${NETWORK}" \
  -v "${REPO_DIR}:/src" \
  -w /src \
  -e DATABASE_URL="postgres://affluena_api:${POSTGRES_PASSWORD}@${PG_SERVICE}:5432/affluena_api?sslmode=disable" \
  "${GO_IMAGE}" \
  sh -c 'apk add --no-cache make >/dev/null && make seed'

log "Done. Demo logins: demo@ / pengamat@ / calon@affluena.com (password123)."
