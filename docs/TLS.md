# TLS.md — Tutorial mengaktifkan HTTPS (siap jalan begitu domain dibeli)

> Status: **DISIAPKAN, belum dieksekusi** (menunggu pemilik membeli domain).
> Prasyarat submit Google Play — lihat `Affluena-MOBILE/docs/PLAYSTORE.md` item 10.
> Sampai tutorial ini dijalankan, semuanya tetap beroperasi normal lewat HTTP + IP.

## Gambaran

Satu VPS (nginx → web statis + reverse-proxy API :8080). Sertifikat dari Let's Encrypt
via certbot (sudah terinstal sejak deploy — `python3-certbot-nginx`). Seluruh kerja server
dibungkus `scripts/setup-tls.sh`; sisanya konfigurasi kecil di .env + GitHub Variables +
satu release mobile.

**Strategi transisi (penting):** APK 1.4.0 yang sudah terpasang memakai URL `http://…`
yang di-bake saat build. Karena itu script default **membiarkan HTTP tetap hidup**
(`--no-redirect`) sampai release 1.5.0 (ber-HTTPS) terpasang; setelah itu jalankan ulang
dengan `--redirect` untuk memaksa HTTP→HTTPS.

## Langkah pemilik (±15 menit + propagasi DNS)

1. **Beli domain** (registrar mana pun; contoh di bawah memakai `affluena.example.com` —
   ganti dengan domainmu).
2. **Set DNS A record** → `43.133.147.101` (IP VPS). Cek propagasi:
   `dig +short affluena.example.com` harus menjawab IP itu.
3. **SSH ke VPS**, lalu:
   ```bash
   cd /opt/affluena/Affluena-API && git pull --ff-only origin master
   bash scripts/setup-tls.sh affluena.example.com
   ```
   Script akan: cek DNS mengarah ke server ini → set `server_name` nginx → terbitkan
   sertifikat → verifikasi auto-renew (certbot.timer) → health-check
   `https://…/healthz`. Gagal di langkah mana pun = berhenti dengan pesan jelas.

## Setelah script sukses (urut)

4. **Update env API** (di VPS): edit `/opt/affluena/deploy/.env` →
   `APP_BASE_URL=https://affluena.example.com` (link email reset) dan
   `CORS_ALLOWED_ORIGINS=https://affluena.example.com`, lalu
   `cd /opt/affluena/deploy && docker compose -f docker-compose.prod.yml up -d api`.
5. **GitHub Variables** (dipakai saat BUILD, bukan runtime): di repo **Affluena-WEB**
   dan **Affluena-Mobile** → Settings → Secrets and variables → Actions → Variables →
   `AFFLUENA_API_BASE_URL = https://affluena.example.com/api/v1`. Deploy web berikutnya
   otomatis memakai URL baru (trigger dengan merge apa pun / re-run workflow).
6. **Mobile release 1.5.0** — minta Claude menjalankan "paket release 1.5.0" di
   `Affluena-MOBILE/docs/PLAYSTORE.md`: base URL https default, hapus izin cleartext
   VPS dari `network_security_config.xml`, buang `local_auth`/`USE_BIOMETRIC`,
   keystore release, bump versi → workflow Shorebird `mode=release` → install APK baru.
7. **Kunci HTTP** setelah 1.5.0 terpasang di perangkatmu:
   ```bash
   bash scripts/setup-tls.sh affluena.example.com --redirect
   ```
8. **Update halaman legal**: URL kebijakan privasi di draft listing Play
   (`PLAYSTORE.md`) menjadi `https://affluena.example.com/privacy`.

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
