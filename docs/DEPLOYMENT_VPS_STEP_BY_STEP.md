# Panduan Deploy VPS dan CI/CD Affluena

Panduan ini dibuat untuk pemula. Ikuti urutan langkahnya pelan-pelan.

Target deployment:

- Satu VPS Ubuntu.
- PostgreSQL dan API berjalan lewat Docker Compose.
- API hanya dibuka di `127.0.0.1:8080`.
- Nginx menjadi pintu publik untuk web dan API.
- Web React/Vite dipublish sebagai file static ke `/srv/affluena/web`.
- CI/CD memakai GitHub Actions dari repo `Affluena-API` dan `Affluena-WEB`.

Ganti semua placeholder berikut dengan data milikmu:

| Placeholder | Contoh | Keterangan |
| --- | --- | --- |
| `VPS_PUBLIC_IP` | `43.133.147.101` | IP publik VPS. |
| `VPS_USER` | `ubuntu` | User SSH VPS. |
| `your-domain.com` | `affluena.com` | Domain, opsional. |

Label command:

- `[LOCAL]` berarti jalankan di laptop/komputer lokal.
- `[VPS]` berarti jalankan setelah SSH masuk ke VPS.
- `[GITHUB]` berarti lakukan di UI GitHub.

## 0. Checklist Sebelum Mulai

Pastikan sudah punya:

- VPS Ubuntu dengan akses SSH.
- Repo GitHub `Affluena-API`.
- Repo GitHub `Affluena-WEB`.
- Folder deploy berisi `docker-compose.prod.yml`, `.env`, dan config Nginx.
- Port VPS provider/security group membuka `22`, `80`, dan `443`.

Jangan pernah kirim private key ke chat. Jika private key pernah terlanjur dikirim ke tempat publik atau chat, anggap key itu bocor dan buat key baru.

## 0.1 Pilih Pakai IP atau Domain

Kamu bisa deploy dengan IP dulu. Domain tidak wajib untuk awal.

Jika belum punya domain, pakai IP:

| Kebutuhan | Isi dengan |
| --- | --- |
| URL browser | `http://VPS_PUBLIC_IP` |
| Health check publik | `http://VPS_PUBLIC_IP/healthz` |
| API base URL untuk Web | `http://VPS_PUBLIC_IP` |
| `CORS_ALLOWED_ORIGINS` API | `http://VPS_PUBLIC_IP` |
| Nginx `server_name` | `_` |
| Certbot/HTTPS | Skip dulu |

Jika sudah punya domain, pakai domain:

| Kebutuhan | Isi dengan |
| --- | --- |
| URL browser | `https://your-domain.com` |
| Health check publik | `https://your-domain.com/healthz` |
| API base URL untuk Web | `https://your-domain.com` |
| `CORS_ALLOWED_ORIGINS` API | `https://your-domain.com` |
| Nginx `server_name` | `your-domain.com` |
| Certbot/HTTPS | Jalankan setelah DNS mengarah ke VPS |

Untuk CI/CD Web, nilai ini disimpan di GitHub repository variable:

```text
VITE_API_BASE_URL=http://VPS_PUBLIC_IP
```

atau jika sudah pakai domain:

```text
VITE_API_BASE_URL=https://your-domain.com
```

Catatan penting: `VITE_API_BASE_URL` dibaca saat `npm run build`. Jika build dilakukan oleh GitHub Actions, yang dipakai adalah GitHub repository variable. Jika build manual di VPS, yang dipakai adalah `.env.production` di VPS.

## 1. Buka Firewall Provider dan UFW

Di dashboard provider VPS, buka inbound rules:

```text
TCP 22  dari 0.0.0.0/0
TCP 80  dari 0.0.0.0/0
TCP 443 dari 0.0.0.0/0
```

Tes SSH dari laptop:

```bash
ssh -p 22 ubuntu@VPS_PUBLIC_IP
```

Jika muncul `Permission denied (publickey)`, port sudah terbuka, tetapi SSH key/user belum cocok.
Jika `Operation timed out`, cek security group provider atau firewall.

Setelah masuk VPS, aktifkan UFW:

```bash
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw --force enable
sudo ufw status
```

## 2. Prepare VPS

[VPS]

```bash
sudo apt update && sudo apt upgrade -y
sudo apt install -y git curl ufw ca-certificates nano rsync
```

Tambahkan swap karena VPS 2 GB RAM cukup ketat:

```bash
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

Catatan: jangan pakai `sudo echo ... >> /etc/fstab`, karena redirect `>>` tetap dijalankan oleh user biasa. Pakai `sudo tee`.

## 3. Install Docker

[VPS]

```bash
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER
```

Logout dari SSH, lalu login lagi. Setelah login ulang:

```bash
docker --version
docker compose version
docker ps
```

Jika `docker ps` masih permission denied:

```bash
groups
```

Pastikan user ada di group `docker`. Kalau belum, ulangi `sudo usermod -aG docker $USER`, lalu logout-login lagi.

## 4. Install Nginx, Certbot, dan Node.js 22

[VPS]

```bash
sudo apt install -y nginx certbot python3-certbot-nginx
sudo systemctl enable nginx
sudo systemctl start nginx
```

Install Node.js 22 untuk build manual web:

```bash
sudo apt remove -y nodejs npm
sudo apt autoremove -y
sudo apt install -y ca-certificates curl
curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -
sudo apt install -y nodejs
node -v
npm -v
```

Output `node -v` harus `v22.x`. Jika masih `v18.x`, `npm run build` Vite akan gagal.

## 5. Buat Folder Project di VPS

[VPS]

```bash
sudo mkdir -p /opt/affluena
sudo chown -R $USER:$USER /opt/affluena
cd /opt/affluena
```

Clone repo:

```bash
git clone https://github.com/adty404/Affluena-API.git
git clone https://github.com/adty404/Affluena-WEB.git
mkdir -p deploy
```

Jika repo sudah ada:

```bash
cd /opt/affluena/Affluena-API && git pull --ff-only origin master
cd /opt/affluena/Affluena-WEB && git pull --ff-only origin master
```

## 6. Buat File Deploy

[VPS]

```bash
cd /opt/affluena/deploy
nano docker-compose.prod.yml
```

Isi:

```yaml
services:
  postgres:
    image: postgres:17-alpine
    restart: unless-stopped
    environment:
      POSTGRES_DB: affluena_api
      POSTGRES_USER: affluena_api
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - affluena_api_postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U affluena_api -d affluena_api"]
      interval: 10s
      timeout: 5s
      retries: 10

  api:
    build:
      context: ../Affluena-API
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
    env_file:
      - ../Affluena-API/.env.production
    environment:
      DATABASE_URL: postgres://affluena_api:${POSTGRES_PASSWORD}@postgres:5432/affluena_api?sslmode=disable
      RUN_MIGRATIONS: "true"
    ports:
      - "127.0.0.1:8080:8080"

volumes:
  affluena_api_postgres_data:
```

Buat `.env` untuk Docker Compose:

```bash
nano .env
```

Isi:

```env
POSTGRES_PASSWORD=GANTI_DENGAN_PASSWORD_HEX
```

Buat password database dengan karakter aman URL:

```bash
openssl rand -hex 24
```

Jangan pakai output base64 untuk `POSTGRES_PASSWORD` jika password itu masuk ke `DATABASE_URL`, karena karakter seperti `/`, `+`, atau `=` bisa membuat URL rusak.

## 7. Configure API Production Env

[VPS]

```bash
cd /opt/affluena/Affluena-API
cp .env.production.example .env.production
nano .env.production
```

Generate JWT secret:

```bash
openssl rand -base64 48
```

Minimal isi penting:

```env
APP_ENV=production
HTTP_ADDR=:8080
JWT_SECRET=ISI_DARI_OPENSSL_RAND_BASE64_48
RUN_MIGRATIONS=true
CORS_ALLOWED_ORIGINS=http://VPS_PUBLIC_IP
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=
SMTP_PASS=
SMTP_FROM=noreply@your-domain.com
```

Jika sudah pakai domain:

```env
CORS_ALLOWED_ORIGINS=https://your-domain.com
```

Jangan commit `.env.production`.

## 8. Start API dan PostgreSQL

[VPS]

```bash
cd /opt/affluena/deploy
docker compose -f docker-compose.prod.yml up -d --build
docker compose -f docker-compose.prod.yml ps
```

Cek log API:

```bash
docker compose -f docker-compose.prod.yml logs --tail=100 api
```

Cek API lokal di VPS:

```bash
curl -i http://127.0.0.1:8080/healthz
```

Expected: HTTP `200`.

## 9. Configure dan Build Web Manual

[VPS]

```bash
cd /opt/affluena/Affluena-WEB
cp .env.production.example .env.production
nano .env.production
```

Jika masih pakai IP:

```env
VITE_API_BASE_URL=http://VPS_PUBLIC_IP
```

Jika sudah pakai domain:

```env
VITE_API_BASE_URL=https://your-domain.com
```

Build:

```bash
npm ci
npm run build
```

Publish:

```bash
sudo mkdir -p /srv/affluena/web
sudo chown -R $USER:$USER /srv/affluena
rsync -a --delete dist/ /srv/affluena/web/
```

## 10. Configure Nginx

[VPS]

```bash
sudo nano /etc/nginx/sites-available/affluena
```

Untuk IP-only, isi `server_name _;`.

```nginx
server {
    listen 80;
    listen [::]:80;
    server_name _;

    root /srv/affluena/web;
    index index.html;

    client_max_body_size 10m;

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location = /healthz {
        proxy_pass http://127.0.0.1:8080/healthz;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

Enable site:

```bash
sudo ln -sfn /etc/nginx/sites-available/affluena /etc/nginx/sites-enabled/affluena
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl reload nginx
```

Cek dari VPS:

```bash
curl -I http://127.0.0.1
curl -i http://127.0.0.1/healthz
```

Cek dari laptop:

```bash
curl -I http://VPS_PUBLIC_IP
curl -i http://VPS_PUBLIC_IP/healthz
```

## 11. HTTPS dengan Domain

Lewati step ini kalau belum punya domain.

Di DNS provider, buat `A record`:

```text
your-domain.com -> VPS_PUBLIC_IP
```

Cek DNS:

```bash
dig +short your-domain.com
```

Edit Nginx:

```bash
sudo nano /etc/nginx/sites-available/affluena
```

Ganti:

```nginx
server_name your-domain.com;
```

Reload:

```bash
sudo nginx -t
sudo systemctl reload nginx
```

Issue SSL:

```bash
sudo certbot --nginx -d your-domain.com
```

Cek renewal:

```bash
sudo certbot renew --dry-run
```

## 12. Seed Data di Server

[VPS]

```bash
cd /opt/affluena/Affluena-API
set -a
. /opt/affluena/deploy/.env
set +a
```

Jalankan seed lewat container Go:

```bash
docker run --rm \
  --network deploy_default \
  -v /opt/affluena/Affluena-API:/src \
  -w /src \
  -e DATABASE_URL="postgres://affluena_api:${POSTGRES_PASSWORD}@postgres:5432/affluena_api?sslmode=disable" \
  golang:1.26-alpine \
  sh -c 'export PATH=/usr/local/go/bin:$PATH; apk add --no-cache make && make seed'
```

Jika network bukan `deploy_default`, cek:

```bash
docker network ls
```

## 13. Manual Deploy Update

Gunakan ini jika ada update baru dan belum memakai CI/CD.

### API Only

[VPS]

```bash
cd /opt/affluena/Affluena-API
git status --short
git pull --ff-only origin master

cd /opt/affluena/deploy
docker compose -f docker-compose.prod.yml up -d --build api
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs --tail=100 api
curl -i http://127.0.0.1:8080/healthz
```

### Web Only

[VPS]

```bash
cd /opt/affluena/Affluena-WEB
git status --short
git pull --ff-only origin master
npm ci
npm run build
rsync -a --delete dist/ /srv/affluena/web/
sudo nginx -t
sudo systemctl reload nginx
curl -I http://127.0.0.1
```

### API dan Web

Deploy API dulu, pastikan `/healthz` HTTP `200`, lalu deploy web.

## 14. Backup dan Rollback Manual

Backup database sebelum deploy besar:

```bash
cd /opt/affluena/deploy
docker compose -f docker-compose.prod.yml exec postgres \
  pg_dump -U affluena_api affluena_api > /opt/affluena/backup-$(date +%F-%H%M%S).sql
```

Rollback API ke commit sebelumnya:

```bash
cd /opt/affluena/Affluena-API
git log --oneline -5
git checkout PREVIOUS_GOOD_API_COMMIT

cd /opt/affluena/deploy
docker compose -f docker-compose.prod.yml up -d --build api
curl -i http://127.0.0.1:8080/healthz
```

Rollback Web:

```bash
cd /opt/affluena/Affluena-WEB
git log --oneline -5
git checkout PREVIOUS_GOOD_WEB_COMMIT
npm ci
npm run build
rsync -a --delete dist/ /srv/affluena/web/
sudo nginx -t
sudo systemctl reload nginx
```

Setelah emergency rollback, balik lagi ke branch:

```bash
git checkout master
```

## 15. Setup CI/CD GitHub Actions

Workflow sudah ada:

```text
Affluena-API/.github/workflows/deploy.yml
Affluena-WEB/.github/workflows/deploy.yml
```

Jadi kamu tidak perlu membuat workflow baru dari GitHub UI. Yang perlu dilakukan adalah memastikan file workflow sudah ada di repo, lalu menyiapkan secrets, variable, SSH key, dan permission VPS.

Alur CI/CD:

1. Push ke `master`.
2. API workflow menjalankan test, vet, build, lalu deploy API ke VPS.
3. Web workflow menjalankan test dan build di GitHub Actions, lalu upload `dist` ke VPS.
4. Telegram mengirim notifikasi jika secret Telegram diisi.

Checklist CI/CD:

| Item | Tempat | Wajib? | Keterangan |
| --- | --- | --- | --- |
| `.github/workflows/deploy.yml` API | Repo `Affluena-API` | Wajib | Sudah ada di repo. Menjalankan test, build, deploy API, dan notif Telegram. |
| `.github/workflows/deploy.yml` Web | Repo `Affluena-WEB` | Wajib | Sudah ada di repo. Menjalankan test, build, upload `dist`, reload Nginx, dan notif Telegram. |
| SSH deploy key | Local + VPS + GitHub Secrets | Wajib | Public key masuk `authorized_keys`, private key masuk secret `VPS_SSH_KEY`. |
| Repository secrets API | GitHub repo `Affluena-API` | Wajib | `VPS_HOST`, `VPS_USER`, `VPS_SSH_KEY`, `VPS_PORT`, `VPS_DEPLOY_ROOT`. |
| Repository secrets Web | GitHub repo `Affluena-WEB` | Wajib | Sama seperti API. |
| Repository variable Web | GitHub repo `Affluena-WEB` | Wajib | `VITE_API_BASE_URL`. |
| Docker permission | VPS | Wajib | User deploy harus bisa menjalankan `docker ps`. |
| Nginx reload permission | VPS | Wajib untuk Web | User deploy harus bisa menjalankan `sudo -n nginx -t` dan `sudo -n systemctl reload nginx`. |
| Telegram secrets | Kedua repo | Opsional | `TELEGRAM_BOT_TOKEN` dan `TELEGRAM_CHAT_ID`. |

Urutan setup CI/CD yang aman:

1. Pastikan manual deploy API dan Web sudah bisa diakses lewat IP atau domain.
2. Buat SSH deploy key khusus CI/CD.
3. Tambahkan public key ke VPS.
4. Tes SSH dari local memakai key baru.
5. Buat GitHub repository secrets satu per satu di `Affluena-API`.
6. Buat GitHub repository secrets satu per satu di `Affluena-WEB`.
7. Buat GitHub repository variable `VITE_API_BASE_URL` di `Affluena-WEB`.
8. Prepare permission Docker dan Nginx di VPS.
9. Run workflow API dulu.
10. Jika API sukses, run workflow Web.

## 16. Buat SSH Deploy Key Khusus CI/CD

Jangan pakai key Termius/keychain kalau sulit diexport. Buat key baru khusus GitHub Actions.

[LOCAL]

```bash
ssh-keygen -t ed25519 -C "affluena-github-actions" -f ~/.ssh/affluena_github_actions -N ""
cat ~/.ssh/affluena_github_actions.pub
```

Copy output `.pub`.

[VPS]

```bash
mkdir -p ~/.ssh
chmod 700 ~/.ssh
nano ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
```

Paste public key sebagai satu baris baru.

[LOCAL]

```bash
ssh -i ~/.ssh/affluena_github_actions -p 22 ubuntu@VPS_PUBLIC_IP
```

Jika berhasil masuk, private key ini yang dipakai untuk GitHub secret:

```bash
cat ~/.ssh/affluena_github_actions
```

## 17. GitHub Repository Secrets

[GITHUB]

Buka:

```text
Repository -> Settings -> Secrets and variables -> Actions -> Secrets -> New repository secret
```

Secret dibuat satu per satu. Nama secret tidak bebas. Buat semua nama berikut di `Affluena-API` dan `Affluena-WEB`.

| Name | Secret value contoh |
| --- | --- |
| `VPS_HOST` | `VPS_PUBLIC_IP` |
| `VPS_USER` | `ubuntu` |
| `VPS_SSH_KEY` | Isi private key dari `cat ~/.ssh/affluena_github_actions` |
| `VPS_PORT` | `22` |
| `VPS_DEPLOY_ROOT` | `/opt/affluena` |
| `TELEGRAM_BOT_TOKEN` | Token dari `@BotFather` |
| `TELEGRAM_CHAT_ID` | Chat id Telegram, misalnya `-1001234567890` |

Contoh form pertama:

```text
Name: VPS_HOST
Secret: 43.133.147.101
```

Klik `Add secret`, lalu buat secret berikutnya.

Khusus `Affluena-WEB`, tambahkan repository variable:

```text
Repository -> Settings -> Secrets and variables -> Actions -> Variables -> New repository variable
```

| Name | Value contoh |
| --- | --- |
| `VITE_API_BASE_URL` | `http://VPS_PUBLIC_IP` |

Jika sudah pakai domain:

```text
VITE_API_BASE_URL=https://your-domain.com
```

## 18. Telegram Notification

Buat bot:

1. Chat `@BotFather`.
2. Kirim `/newbot`.
3. Simpan token ke secret `TELEGRAM_BOT_TOKEN`.

Paling mudah pakai group:

1. Buat group `Affluena Deploy`.
2. Masukkan bot ke group.
3. Kirim pesan `test deploy`.
4. Buka:

```text
https://api.telegram.org/bot<TELEGRAM_BOT_TOKEN>/getUpdates
```

Cari:

```json
"chat": {
  "id": -1001234567890
}
```

Isi `TELEGRAM_CHAT_ID` dengan angka itu.

Jika `getUpdates` kosong:

```text
https://api.telegram.org/bot<TELEGRAM_BOT_TOKEN>/deleteWebhook
```

Lalu kirim pesan lagi dan buka `getUpdates` lagi.

## 19. Prepare VPS Untuk CI/CD

[VPS]

```bash
sudo usermod -aG docker ubuntu
sudo mkdir -p /srv/affluena/web
sudo chown -R ubuntu:ubuntu /srv/affluena
```

Logout-login lagi, lalu cek:

```bash
docker ps
```

Passwordless Nginx reload:

```bash
sudo visudo -f /etc/sudoers.d/affluena-deploy
```

Isi:

```text
ubuntu ALL=(root) NOPASSWD: /usr/sbin/nginx -t, /usr/bin/systemctl reload nginx
```

Validasi:

```bash
sudo visudo -cf /etc/sudoers.d/affluena-deploy
sudo -n nginx -t
sudo -n systemctl reload nginx
```

## 20. Jalankan CI/CD Pertama Kali

[GITHUB]

Run API dulu:

```text
Affluena-API -> Actions -> API CI/CD -> Run workflow
```

Cek:

```bash
curl -i http://VPS_PUBLIC_IP/healthz
```

Lalu run Web:

```text
Affluena-WEB -> Actions -> Web CI/CD -> Run workflow
```

Cek:

```bash
curl -I http://VPS_PUBLIC_IP
curl -i http://VPS_PUBLIC_IP/healthz
```

## 21. Troubleshooting Umum

### `Permission denied` saat buat `/opt/affluena`

Pakai `sudo` dan ubah owner:

```bash
sudo mkdir -p /opt/affluena
sudo chown -R $USER:$USER /opt/affluena
```

### `/etc/fstab: Permission denied`

Gunakan `sudo tee`:

```bash
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

### API restart terus dan log `invalid port` pada `DATABASE_URL`

Password database kemungkinan memakai karakter base64 seperti `/` atau `+`. Ganti dengan:

```bash
openssl rand -hex 24
```

Update `/opt/affluena/deploy/.env`, lalu:

```bash
cd /opt/affluena/deploy
docker compose -f docker-compose.prod.yml up -d --build api
```

### `npm ci` download dari registry aneh atau timeout

Cek registry:

```bash
npm config get registry
```

Set ke public npm:

```bash
npm config set registry https://registry.npmjs.org/
```

### Vite error Node.js 18

Install Node.js 22 sesuai step 4.

### GitHub Actions `Missing VPS_HOST secret`

Secret belum dibuat di repo yang benar, atau dibuat sebagai variable. Buat di:

```text
Settings -> Secrets and variables -> Actions -> Secrets -> Repository secrets
```

### `ssh-keyscan getaddrinfo`

`VPS_HOST` harus IP saja:

```text
43.133.147.101
```

Jangan:

```text
http://43.133.147.101/
ubuntu@43.133.147.101
```

### SSH `Permission denied (publickey)`

Public key belum ada di `/home/ubuntu/.ssh/authorized_keys`, user salah, atau private key di GitHub tidak cocok. Tes dari laptop:

```bash
ssh -i ~/.ssh/affluena_github_actions -p 22 ubuntu@VPS_PUBLIC_IP
```

### Deploy berhenti karena tracked local changes

Di VPS:

```bash
cd /opt/affluena/Affluena-API
git status --short

cd /opt/affluena/Affluena-WEB
git status --short
```

Jika ada file tracked berubah, cek dulu dengan:

```bash
git diff --stat
```

Jangan sembarang reset kalau tidak yakin.

### `.env.production` muncul di `git status`

Pull commit terbaru yang sudah meng-ignore `.env.production`:

```bash
git pull --ff-only origin master
git status --short
```

### Web deploy gagal saat reload Nginx

Cek sudoers:

```bash
sudo visudo -cf /etc/sudoers.d/affluena-deploy
sudo -n nginx -t
sudo -n systemctl reload nginx
```

### Cek hasil deploy

[VPS]

```bash
cd /opt/affluena/deploy
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs --tail=100 api
curl -i http://127.0.0.1:8080/healthz
curl -I http://127.0.0.1
```

[LOCAL]

```bash
curl -I http://VPS_PUBLIC_IP
curl -i http://VPS_PUBLIC_IP/healthz
```
