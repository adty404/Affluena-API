# Affluena-API — orientation for a fresh session

Personal-finance backend. **Go 1.26 + Gin + PostgreSQL (pgx/pgxpool)**, module `affluena-api`.
Clean layering per domain: `handler → usecase → repository`, wired in `internal/server/router.go`.

> This file is the 2-minute orientation. For the full working rules / invariants / change
> checklist, read **`AGENTS.md`**. For endpoints read **`docs/API_CONTRACT.md`**; for the
> module map read **`docs/system_map.md`**.

## ⚠️ Read before you change anything (prod dangers)

- **Merging to `master` deploys to PRODUCTION.** `.github/workflows/deploy.yml` ("API CI/CD")
  runs on every push to `master` and ships to the VPS. There is no separate "release" step —
  **a merged PR is a prod deploy.** Get explicit confirmation before merging.
- **Migrations run automatically at startup** (`RUN_MIGRATIONS` defaults to `true`). They are
  **forward-only** (`migrations/000NNN_*.sql`, applied in order, tracked in `schema_migrations`),
  run inside a tx under an advisory lock by `internal/db/migrate.go`. **A failed migration calls
  `os.Exit(1)` — the container won't start, so a bad migration blocks prod from booting.**
  - Never edit an already-applied migration (the runner silently skips it). Always add a new file.
  - **Validate data-transforming migrations against POPULATED data**, not just a fresh DB.
    `make verify` migrates an EMPTY test DB, so ordering bugs that only fail on existing rows
    slip through (e.g. an `UPDATE … SET col='new'` that runs *before* the old CHECK is dropped).
    Spin up Postgres, run the OLD binary + `make seed` to populate, then apply the new migration.
- **HTTP status is keyword-mapped from the error message** (`internal/httpx`): text containing
  `not found`→404, `already exists`/`conflict`→409, etc. Rewording an error can silently change
  the response status. Prefer `httpx.NewPublicError(msg, status)` for an explicit status.

## Run / test (Docker, Go, and psql are all available in the cloud sandbox)

```bash
# Full stack (port 8080, auto-migrates, RUN_MIGRATIONS=true):
docker compose up --build

# Run the server directly against a Postgres you control:
JWT_SECRET='<≥32 random chars>' DATABASE_URL='postgres://…/affluena_api?sslmode=disable' go run ./cmd/api

# Verification gate (gofmt, unit + integration tests vs Postgres, vet, build, Postman JSON, health):
make verify           # uses AFFLUENA_API_TEST_DATABASE_URL (or the compose DB)

# Demo/seed data (idempotent; demo@/pengamat@/calon@affluena.com / password123):
make seed             # see docs/SEED_DATA.md ; on the VPS use scripts/seed-prod.sh
```

`JWT_SECRET` must be a real value (startup `config.Validate()` rejects the placeholder), and
`sslmode=disable` is refused in production unless `ALLOW_INSECURE_DB=true`.

## Where things live

- `cmd/api/main.go` → boot: load config → open pool → migrate → `server.Router` → listen `:8080`.
- `internal/server/router.go` → composition root; every route is registered here.
- `internal/<domain>/` → one folder per feature (`auth`, `wallet`, `transaction`, `budget`,
  `debt`, `tracker`, `recurring`, `goal`, `category`, `partner`, `dashboard`, `report`,
  `notification`, …), each as `domain.go` / `repository.go` / `usecase.go` / `handler.go` (+ tests).
- `internal/httpx` → shared HTTP helpers (error→status mapping, UUID params, JSON, rate limit).
- `migrations/` → ordered SQL. `internal/db` → connection + migration runner.

**Adding a domain:** create `internal/<x>/{domain,repository,usecase,handler}.go` (scope every
query by `user_id`), wire routes in `router.go`, add an integration test under `internal/server/`,
update `docs/API_CONTRACT.md` + the Postman collection, then `make verify`.

## Conventions & invariants (summary — details in AGENTS.md / API_CONTRACT.md)

- **Money = integer minor units** (`*_minor`, bigint). Never floats on the wire.
- **Auth**: HS256 access JWT (~15m) + rotating opaque refresh token. `PUT /auth/password` revokes
  other sessions and returns a fresh pair (clients must re-persist).
- **User isolation**: every resource is scoped by `user_id` at the DB + app layer.
- **Shared wallets**: `wallet_shares.role` = `member` (read+write) or `viewer` (read-only). Write
  paths must check access (canonical `wallet.AccessChecker`; some modules still inline the predicate).
- **Dates**: a Postgres `DATE`/timestamp column is serialized by Go `time.Time` as a **full RFC3339
  timestamp** (e.g. budget `month` → `"2026-06-01T00:00:00Z"`, NOT `"2026-06"`). Clients must parse
  defensively — assuming `YYYY-MM` will throw.
- **"Berbagi Dompet"** (share-all-my-wallets, read-only, max 5 viewers, one-way) keeps historical
  internal names: HTTP endpoints `/api/v1/partners`, Go package `internal/partner`, but DB table
  `wallet_share_links` (cols `owner_id`/`viewer_id`) and `wallet_shares.source = 'link'`. The
  user-facing app calls it "Berbagi Dompet". See `docs/API_CONTRACT.md`.
- `GET /api/v1/goals` returns a bare JSON array (the one list endpoint without `{…, pagination}`).
