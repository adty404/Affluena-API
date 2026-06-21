# CI/CD Setup

This project uses GitHub Actions for deployment to the VPS.

For the complete beginner-friendly manual deploy and CI/CD guide, read [`DEPLOYMENT_VPS_STEP_BY_STEP.md`](DEPLOYMENT_VPS_STEP_BY_STEP.md).

Important: GitHub secrets are created one by one. Do not create one big secret that contains all values.

## 1. Where To Add Secrets

Open each repository on GitHub:

```text
Repository -> Settings -> Secrets and variables -> Actions -> Secrets -> New repository secret
```

Use `Repository secrets` for the first setup. Do not use `Environment secrets` unless the workflow is changed deliberately.

## 2. Affluena-API Repository Secrets

Create these secrets one by one in the `Affluena-API` repository.

| Name | Secret value example | Notes |
| --- | --- | --- |
| `VPS_HOST` | `203.0.113.10` | VPS public IP only. Do not include `http://` or `https://`. |
| `VPS_USER` | `ubuntu` | SSH user on the VPS. |
| `VPS_SSH_KEY` | `-----BEGIN OPENSSH PRIVATE KEY----- ...` | Full private deploy key, including begin/end lines. |
| `VPS_PORT` | `22` | SSH port. Optional, but recommended to set. |
| `VPS_DEPLOY_ROOT` | `/opt/affluena` | App root on the VPS. Optional, but recommended to set. |
| `TELEGRAM_BOT_TOKEN` | `<telegram bot token>` | Optional. Bot token from `@BotFather`. |
| `TELEGRAM_CHAT_ID` | `<telegram chat id>` | Optional. Telegram group/channel/user chat id. |

Example form input:

```text
Name: VPS_HOST
Secret: 203.0.113.10
```

Then click `Add secret`, and create the next one:

```text
Name: VPS_USER
Secret: ubuntu
```

Repeat until all required secrets exist.

## 3. Affluena-WEB Repository Secrets

Create these secrets one by one in the `Affluena-WEB` repository too.

| Name | Secret value example | Notes |
| --- | --- | --- |
| `VPS_HOST` | `203.0.113.10` | VPS public IP only. Do not include `http://` or `https://`. |
| `VPS_USER` | `ubuntu` | SSH user on the VPS. |
| `VPS_SSH_KEY` | `-----BEGIN OPENSSH PRIVATE KEY----- ...` | Full private deploy key, including begin/end lines. |
| `VPS_PORT` | `22` | SSH port. Optional, but recommended to set. |
| `VPS_DEPLOY_ROOT` | `/opt/affluena` | App root on the VPS. Optional, but recommended to set. |
| `TELEGRAM_BOT_TOKEN` | `<telegram bot token>` | Optional. Bot token from `@BotFather`. |
| `TELEGRAM_CHAT_ID` | `<telegram chat id>` | Optional. Telegram group/channel/user chat id. |

## 4. Affluena-WEB Repository Variable

For the web app, add this as a repository variable, not a secret:

```text
Repository -> Settings -> Secrets and variables -> Actions -> Variables -> New repository variable
```

| Name | Value example | Notes |
| --- | --- | --- |
| `VITE_API_BASE_URL` | `http://203.0.113.10` | Use the VPS IP first. Change to `https://your-domain.com` after domain and HTTPS are ready. |

Example form input:

```text
Name: VITE_API_BASE_URL
Value: http://203.0.113.10
```

## 5. Telegram Chat ID

For the simplest setup, use a Telegram group:

1. Create a group, for example `Affluena Deploy`.
2. Add the bot to the group.
3. Send a message in the group, for example `test deploy`.
4. Open this URL:

```text
https://api.telegram.org/bot<TELEGRAM_BOT_TOKEN>/getUpdates
```

Find:

```json
"chat": {
  "id": -1001234567890
}
```

Use that number as:

```text
Name: TELEGRAM_CHAT_ID
Secret: -1001234567890
```

If `getUpdates` returns an empty result, open this URL first:

```text
https://api.telegram.org/bot<TELEGRAM_BOT_TOKEN>/deleteWebhook
```

Then send another message to the group and run `getUpdates` again.

## 6. VPS Preparation

The VPS must already have these folders:

```text
/opt/affluena/Affluena-API
/opt/affluena/Affluena-WEB
/opt/affluena/deploy
```

Pull the latest workflow-related changes on the VPS:

```bash
cd /opt/affluena/Affluena-API
git pull --ff-only origin master

cd /opt/affluena/Affluena-WEB
git pull --ff-only origin master
```

Check the working trees:

```bash
cd /opt/affluena/Affluena-API
git status --short

cd /opt/affluena/Affluena-WEB
git status --short
```

The deploy workflow stops if there are tracked local changes on the VPS. Local `.env.production` files are ignored.

Allow the deploy user to run Docker:

```bash
sudo usermod -aG docker ubuntu
```

Log out from SSH and log in again, then check:

```bash
docker ps
```

Create the web publish directory:

```bash
sudo mkdir -p /srv/affluena/web
sudo chown -R ubuntu:ubuntu /srv/affluena
```

Allow passwordless Nginx reload:

```bash
sudo visudo -f /etc/sudoers.d/affluena-deploy
```

Add:

```text
ubuntu ALL=(root) NOPASSWD: /usr/sbin/nginx -t, /usr/bin/systemctl reload nginx
```

Validate:

```bash
sudo visudo -cf /etc/sudoers.d/affluena-deploy
sudo -n nginx -t
```

## 7. Run The Workflows

Run API first:

```text
Affluena-API -> Actions -> API CI/CD -> Run workflow
```

After it succeeds:

```bash
curl -i http://VPS_PUBLIC_IP/healthz
```

Then run Web:

```text
Affluena-WEB -> Actions -> Web CI/CD -> Run workflow
```

After it succeeds:

```bash
curl -I http://VPS_PUBLIC_IP
```

## 8. Common Mistakes

- Secret names are not free text. They must match the names in the tables exactly.
- `VPS_HOST` is only the IP address, without `http://`.
- `VPS_SSH_KEY` is the private key, not the public `.pub` key.
- Secrets must be created in both repositories because API and Web are separate GitHub repositories.
- `VITE_API_BASE_URL` is a web repository variable, not an API secret.
- If a workflow still sees empty secrets, run a new workflow after saving the secrets.
