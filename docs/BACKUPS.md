# Database Backups & Restore

Automated Postgres backups for the Affluena VPS. Three scripts in `scripts/`,
wired so that **merging to `master` keeps the whole scheme installed and
current** — no manual VPS steps.

## What runs, when

| What | When | How it's triggered |
| --- | --- | --- |
| `scripts/backup-db.sh daily` | Nightly **19:30 UTC** (02:30 WIB) | User crontab, installed by `install-backup-cron.sh` |
| `scripts/verify-backup.sh` | Weekly **Sun 20:00 UTC** (Mon 03:00 WIB) | Same crontab |
| `scripts/backup-db.sh pre-deploy` | **Every deploy**, before `docker compose up` boots the new image (and its startup migrations) | `.github/workflows/deploy.yml` |
| `scripts/install-backup-cron.sh` | Every deploy (idempotent refresh) | `.github/workflows/deploy.yml` |

A **failed pre-deploy backup fails the deploy on purpose** (fail-closed):
deploying a money app without a restore point is worse than a delayed deploy.
The gate keys on the **data volume**, not on a running container — if the
`affluena_api_postgres_data` volume exists, a snapshot is mandatory, so a
stopped/crashed Postgres at deploy time fails the deploy rather than silently
skipping the backup. It is skipped only on a true first deploy (no volume yet).
Emergency bypass: SSH in and run the compose commands from the workflow by hand.

The **cron refresh is deliberately non-fatal**: it runs after the new API is
already live and healthy, so a crontab hiccup emits a workflow warning instead
of reporting a shipped deploy as failed.

## What a backup contains

- `affluena_api-<label>-<stamp>.dump` — `pg_dump -Fc` (compressed custom
  format) of the whole database, integrity-gated with `pg_restore --list`
  (TOC parse — catches header corruption; full restorability is proven by the
  weekly restore-verify) before it replaces the `.part` temp file. A failed
  run cleans its own `.part` via trap; `.part` leftovers from a hard kill are
  swept age-gated (>60 min) by the next run's retention pass.
- `deploy-config-<label>-<stamp>.tar.gz` — snapshot of `/opt/affluena/deploy`
  (compose file, `.env` with `POSTGRES_PASSWORD`/`JWT_SECRET`, nginx config).
  Losing these with an intact DB volume still locks you out.

Everything lands in `/opt/affluena/backups` with `0700`/`0600` permissions
(financial data + secrets). Logs: `backup.log` / `verify.log` in the same dir.

## Retention

- `daily` dumps + config tars: deleted after **14 days** (`KEEP_DAILY_DAYS`).
- `pre-deploy` dumps + config tars: newest **10** kept (`KEEP_PRE_DEPLOY`) —
  they arrive in bursts, so age-based pruning would be wrong.
- A **200 MB free-disk floor** (`MIN_FREE_MB`) aborts a backup rather than fill
  the disk and take Postgres down with it.

At the current DB size a dump is ~100 KB, so a full retention window is a few
MB — negligible on the VPS disk.

## Restore verification (weekly + on demand)

`verify-backup.sh` restores the newest dump into a **throwaway** Postgres
container (same image as the live one; the live container and volume are never
touched), then asserts `schema_migrations` has rows and the core tables
(`users`, `wallets`, `transactions`, `category_budgets`) exist, printing row
counts. Exit 0 = the backup is restorable, not just present.

Hygiene: the throwaway's data dir is a **tmpfs** (the restored copy of the
production DB lives only in RAM, never on disk), the container is removed with
its volumes (`docker rm -fv`) on exit, and each run first reaps any stale
`affluena-restore-verify-*` container a hard kill may have left behind.
Readiness is probed over **TCP + a real `SELECT 1`** so the official image's
temporary socket-only init server can't fake a ready signal.

```bash
# on the VPS — verify the newest backup, or a specific one:
bash /opt/affluena/Affluena-API/scripts/verify-backup.sh
bash /opt/affluena/Affluena-API/scripts/verify-backup.sh /opt/affluena/backups/affluena_api-daily-20260705-023000.dump
```

## Restoring for real (disaster recovery)

```bash
cd /opt/affluena/deploy
# 1. Stop the API so nothing writes mid-restore:
docker compose -f docker-compose.prod.yml stop api
# 2. Drop + recreate the schema and restore the dump:
docker compose -f docker-compose.prod.yml exec -T postgres \
  psql -U affluena_api -d postgres -c 'DROP DATABASE affluena_api WITH (FORCE)'
docker compose -f docker-compose.prod.yml exec -T postgres \
  psql -U affluena_api -d postgres -c 'CREATE DATABASE affluena_api OWNER affluena_api'
docker compose -f docker-compose.prod.yml exec -T postgres \
  pg_restore -U affluena_api -d affluena_api --no-owner < /opt/affluena/backups/<dump-file>
# 3. Bring the API back (its startup migrations are no-ops on a restored DB):
docker compose -f docker-compose.prod.yml up -d api
curl -fsS http://127.0.0.1:8080/healthz
```

## Off-site copies (recommended next step)

Local backups die with the disk. `backup-db.sh` has an optional hook: install
`rclone`, configure a remote (Backblaze B2 / Cloudflare R2 / S3 / Google
Drive), then set `BACKUP_RCLONE_REMOTE` (e.g. `b2:affluena-backups`) in the
cron line or environment — every new dump is copied off-host automatically.
Until that's configured, backups protect against bad migrations and fat-finger
deletes, **not** against losing the VPS itself.

## Tuning

All knobs are env vars with defaults — see the header comments in each script:
`DEPLOY_ROOT`, `BACKUP_DIR`, `PG_SERVICE`, `KEEP_DAILY_DAYS`, `KEEP_PRE_DEPLOY`,
`MIN_FREE_MB`, `DAILY_SCHEDULE`, `VERIFY_SCHEDULE`, `BACKUP_RCLONE_REMOTE`.
Schedule/retention changes belong in a PR (the deploy refreshes the cron), not
in hand-edited crontabs.
