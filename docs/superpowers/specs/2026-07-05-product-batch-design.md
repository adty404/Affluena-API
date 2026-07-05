# Design: Product batch — healthz, server search, onboarding defaults, mobile push, Beranda insights, web Kalender

Date: 2026-07-05 · Scope: Affluena-API, Affluena-MOBILE, Affluena-WEB (3 PRs)

## A. API

### A1. `/healthz` pings the DB
`GET /healthz` runs `SELECT 1` on the pgx pool with a ~2s timeout. Healthy →
`200 {"status":"ok"}`; DB unreachable → `503 {"status":"degraded","db":"<err class>"}`.
Consequence (intended): the deploy workflow's `curl -fsS /healthz` gate now
fails when Postgres is down, and nginx health checks stop reporting a dead app
as live. No separate liveness endpoint — one small binary, one gate.

### A2. `search` param on `GET /api/v1/transactions`
Optional `search=<q>` (trimmed, max 100 chars). Matches, user-scoped and
case-insensitive: transaction `note`, the transaction's category `name`, or its
wallet `name` (`ILIKE '%q%'` with `%`/`_` escaped; EXISTS subqueries — no result
duplication). Composes with every existing filter/sort/pagination. This gives
clients full-history search parity with the note/wallet/category semantics the
mobile Aktivitas client-side filter shipped with.

### A3. Onboarding defaults on register
`POST /auth/register` also creates, in the same transaction as the user row:
- 8 default categories (Bahasa Indonesia, icon+color from the shared client
  catalogs): Makanan & Minuman, Transportasi, Belanja, Tagihan & Utilitas,
  Hiburan, Kesehatan (expense) · Gaji, Penghasilan Lain (income).
- 1 starter wallet: "Dompet Utama" (cash, IDR, balance 0).
Fresh accounts land on a usable app instead of a blank one. Register response
shape unchanged. Seeder accounts unaffected (seeder deletes/recreates its own).

## B. MOBILE (Shorebird **release** — native plugin, NOT an OTA patch)

### B1. Device notifications (local, no FCM)
`flutter_local_notifications` (+ `timezone`). No Firebase: reminders are
scheduled ON-DEVICE from data the app already has, mirroring the server-side
notification rules (the API scheduler keeps powering in-app notifications).
- What: due reminders H-3 and H-1 at 09:00 local for upcoming installments /
  subscriptions / debts (from the dashboard summary's `upcoming_*`), gated on
  the user's `notification_rules` (a disabled rule schedules nothing).
- When (re)scheduled: app start + after any financial mutation → cancel-all,
  re-schedule batch (cap 50).
- Permission: Android 13+ `POST_NOTIFICATIONS` runtime request, asked once from
  the Pengaturan → Notifikasi flow (and on first launch when any rule is on).
- Ships as `version: 1.4.0+7` — the push-triggered patch job fail-fasts by
  design; the actual build is `workflow_dispatch → mode=release` producing an
  installable APK artifact. Users must install that APK once; after that,
  OTA patches resume as normal.

### B2. Beranda "Jatuh tempo terdekat"
New section: nearest 3 dues across subscriptions/installments/debts (summary
data, already returned and unused), sorted by date, each row tappable to its
domain screen. Hidden when empty.

### B3. Savings rate + net-worth trend
- Savings-rate tile (this month): `cashflow / income` (guard income=0 → "—").
- Net-worth trend: 12-point series derived client-side — anchor at current
  `net_worth_minor`, walk back month-by-month subtracting each month's net
  cashflow (from the cashflow-trend endpoint). Rendered as a sparkline card.

### B4. Aktivitas server-side search
The search field now debounces (~350ms) into `listTransactions(search:)` via a
`search` field on `ActivityQuery` — full history, same note/category/wallet
semantics as before (A2), no more 100-row client-side ceiling.

## C. WEB — Kalender
New `/calendar` route + "Kalender" nav entry (Wawasan section), mirroring the
mobile Kalender: month grid (id-ID, Monday start) over `useMonthTransactions`,
per-day income/expense markers + net amount, prev/next month + "Hari ini",
clicking a day opens its transaction list (existing row/Amount components,
EmptyState for empty days). Loading/error via shared states. Pure client-side
over existing endpoints.

## Testing
- API: integration tests — healthz degraded path (pool pointed at a dead DB),
  search matching note/category/wallet + escaping, register-creates-defaults.
  `make verify` green.
- MOBILE: unit/widget tests for the reminder scheduler (pure scheduling calc),
  Beranda sections, search debounce wiring; `analyze` + full suite green.
- WEB: vitest for calendar day-bucketing helper; build green.

## Out of scope
FCM/remote push, web savings-rate/net-worth, iOS notification setup (Android
sideload build only, per current distribution), localizing `/reports/*`.
