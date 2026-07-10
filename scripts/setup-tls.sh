#!/usr/bin/env bash
#
# Enable HTTPS for Affluena on the VPS with Let's Encrypt — run ON THE VPS:
#
#     bash scripts/setup-tls.sh affluena.example.com
#
# Prerequisites (docs/TLS.md is the full tutorial):
#   1. You own the domain and its DNS A record points at THIS server's IP.
#   2. nginx + certbot are installed (deploy step 4 already did this).
#
# What it does:
#   - sanity-checks the domain resolves to this machine,
#   - sets `server_name` in /etc/nginx/sites-available/affluena,
#   - obtains + installs the certificate via `certbot --nginx`,
#   - verifies auto-renewal with a dry run,
#   - health-checks https://<domain>/healthz.
#
# ⚠️ TRANSITION DEFAULT: HTTP is kept alive (--no-redirect) so the already-
# installed mobile 1.4.0 builds (plain http) keep working. After the 1.5.0
# release (https base URL) is installed, re-run with --redirect to force
# HTTP→HTTPS:
#
#     bash scripts/setup-tls.sh affluena.example.com --redirect
set -euo pipefail

DOMAIN="${1:-}"
MODE="${2:---no-redirect}"
SITE="/etc/nginx/sites-available/affluena"

if [ -z "$DOMAIN" ]; then
  echo "usage: bash scripts/setup-tls.sh <domain> [--redirect|--no-redirect]" >&2
  exit 1
fi
case "$MODE" in --redirect|--no-redirect) ;; *)
  echo "!! second arg must be --redirect or --no-redirect (got: $MODE)" >&2; exit 1;;
esac

SUDO=""
if [ "$(id -u)" -ne 0 ]; then SUDO="sudo"; fi

log() { printf '\n==> %s\n' "$1"; }

log "Checking DNS: $DOMAIN must point at this server"
SERVER_IP="$(curl -fsS -4 https://ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')"
DOMAIN_IP="$(getent ahostsv4 "$DOMAIN" 2>/dev/null | awk '{print $1; exit}' || true)"
if [ -z "$DOMAIN_IP" ]; then
  echo "!! $DOMAIN does not resolve yet. Set the DNS A record to $SERVER_IP and wait for propagation (check: dig +short $DOMAIN)." >&2
  exit 1
fi
if [ "$DOMAIN_IP" != "$SERVER_IP" ]; then
  echo "!! $DOMAIN resolves to $DOMAIN_IP but this server is $SERVER_IP." >&2
  echo "   Fix the A record first — certbot's HTTP challenge will fail otherwise." >&2
  exit 1
fi
echo "   OK: $DOMAIN -> $DOMAIN_IP (this server)"

log "Setting server_name in $SITE"
[ -f "$SITE" ] || { echo "!! $SITE not found — finish deploy step 10 first." >&2; exit 1; }
$SUDO sed -i.bak-tls -E "s/^(\s*server_name)\s+.*;/\1 $DOMAIN;/" "$SITE"
$SUDO nginx -t
$SUDO systemctl reload nginx

log "Obtaining certificate via certbot (mode: $MODE)"
$SUDO certbot --nginx -d "$DOMAIN" --non-interactive --agree-tos \
  --register-unsafely-without-email "$MODE"

log "Verifying auto-renewal (dry run)"
$SUDO certbot renew --dry-run >/dev/null && echo "   OK: auto-renewal works (systemd timer certbot.timer)"

log "Health checks"
curl -fsS "https://$DOMAIN/healthz" && echo "   OK: https://$DOMAIN/healthz"
if [ "$MODE" = "--no-redirect" ]; then
  curl -fsS "http://$DOMAIN/healthz" >/dev/null && echo "   OK: plain HTTP still up (transition mode for mobile 1.4.0)"
fi

cat <<NEXT

==> TLS aktif untuk https://$DOMAIN
    LANGKAH LANJUTAN (lihat docs/TLS.md bagian "Setelah script sukses"):
    1. deploy .env: APP_BASE_URL=https://$DOMAIN + CORS_ALLOWED_ORIGINS=https://$DOMAIN,
       lalu: cd /opt/affluena/deploy && docker compose -f docker-compose.prod.yml up -d api
    2. GitHub → Affluena-WEB & Affluena-Mobile → Settings → Variables:
       AFFLUENA_API_BASE_URL=https://$DOMAIN/api/v1 (dipakai saat build)
    3. Minta Claude jalankan paket release mobile 1.5.0 (https + tanpa cleartext).
    4. Setelah 1.5.0 terpasang: bash scripts/setup-tls.sh $DOMAIN --redirect
NEXT
