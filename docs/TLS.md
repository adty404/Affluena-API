# TLS.md — Tutorial mengaktifkan HTTPS (siap jalan begitu domain dibeli)

> Status: **DIEKSEKUSI 2026-10-08** — `https://affluena.adityabuild.codes` sudah live
> (Langkah 1–5 selesai: DNS, TLS, env API, GitHub Variables). Sisa: install release mobile
> ber-HTTPS (1.6.0+9) lalu kunci `--redirect` (Langkah 6–7).
> Prasyarat submit Google Play — lihat `Affluena-MOBILE/docs/PLAYSTORE.md` item 10.

## Gambaran

Satu VPS (nginx → web statis + reverse-proxy API :8080). Sertifikat dari Let's Encrypt
via certbot (sudah terinstal sejak deploy — `python3-certbot-nginx`). Seluruh kerja server
dibungkus `scripts/setup-tls.sh`; sisanya konfigurasi kecil di .env + GitHub Variables +
satu release mobile.

**Strategi transisi (penting):** APK lama (1.4.0 / 1.5.0+8) memakai URL `http://…` yang
di-bake saat build, jadi server tetap melayani HTTP. Script default **membiarkan HTTP
tetap hidup** (`--no-redirect`) sampai release ber-HTTPS terpasang; setelah itu jalankan
ulang dengan `--redirect` untuk memaksa HTTP→HTTPS.

## Langkah pemilik (±15 menit + propagasi DNS)

1. ✅ **Beli domain** — dipakai: `affluena.adityabuild.codes` (subdomain dari `adityabuild.codes`).
2. ✅ **Set DNS A record** → `43.133.147.101` (IP VPS). (Tambahan server: vhost `affluena`
   dijadikan `default_server` port 80 agar akses lewat IP tetap ke Affluena — APK lama.)
3. ✅ **SSH ke VPS**, lalu jalankan script (sudah dijalankan dengan domain asli):
   ```bash
   cd /opt/affluena/Affluena-API && git pull --ff-only origin master
   bash scripts/setup-tls.sh affluena.adityabuild.codes
   ```
   Script akan: cek DNS mengarah ke server ini → set `server_name` nginx → terbitkan
   sertifikat → verifikasi auto-renew (certbot.timer) → health-check
   `https://…/healthz`. Gagal di langkah mana pun = berhenti dengan pesan jelas.

## Setelah script sukses (urut)

4. ✅ **Update env API** (di VPS): edit file yang muncul di `env_file:` compose —
   `/opt/affluena/Affluena-API/.env.production` (BUKAN `deploy/.env`, itu hanya untuk
   interpolasi `${POSTGRES_PASSWORD}`) → `APP_BASE_URL=https://affluena.adityabuild.codes`
   (link email reset) dan `CORS_ALLOWED_ORIGINS=https://affluena.adityabuild.codes`, lalu:
   ```bash
   cd /opt/affluena/deploy && docker compose -f docker-compose.prod.yml up -d --force-recreate api
   ```
5. **GitHub Variables** (dipakai saat BUILD, bukan runtime): repo **Affluena-WEB** →
   `VITE_API_BASE_URL=https://affluena.adityabuild.codes` ✅ (web sudah di-rebuild);
   repo **Affluena-Mobile** → `AFFLUENA_API_BASE_URL=https://affluena.adityabuild.codes/api/v1`
   (dipakai build 1.6.0+9). Deploy berikutnya otomatis memakai URL baru.
6. ⏳ **Mobile release 1.6.0+9** — paketnya sudah disiapkan di repo `Affluena-Mobile`
   (lihat `docs/PLAYSTORE.md` → "Paket rilis 1.6.0+9"): base URL https default, izin
   cleartext VPS dibuang, keystore + signing release, bump versi. Setelah PR merge →
   workflow Shorebird `mode=release` → install APK baru.
7. ⏳ **Kunci HTTP** setelah 1.6.0+9 terpasang di perangkat:
   ```bash
   bash scripts/setup-tls.sh affluena.adityabuild.codes --redirect
   ```
8. **Update halaman legal**: URL kebijakan privasi di draft listing Play
   (`PLAYSTORE.md`) menjadi `https://affluena.adityabuild.codes/privacy`.

## Verifikasi akhir

- `curl -fsS https://affluena.example.com/healthz` → `{"status":"ok"}`
- Web login lewat `https://` normal; mobile 1.5.0 login normal.
- `sudo certbot renew --dry-run` hijau (perpanjangan otomatis; sertifikat LE berumur
  90 hari, timer memperpanjang sendiri).
- Formulir Play Data Safety: "data dienkripsi in transit" kini bisa dijawab **Ya**.

## Rollback

TLS bersifat aditif: `sudo sed -i -E 's/^(\s*server_name).*/\1 _;/' \
/etc/nginx/sites-available/affluena` + hapus blok 443 yang ditambah certbot (atau
restore backup `*.bak-tls`), `sudo nginx -t && sudo systemctl reload nginx`. Klien HTTP
lama langsung bekerja lagi.
