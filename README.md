<div align="center">

# 🏕️ CampMenu

**Plan group-trip menus and shopping lists, together — and ditch the shared spreadsheet.**

Collaborative web app to organize meals, drinks, cocktails, shopping lists and accommodation
for trips with friends (ski, camping, festivals, road trips…). Real‑time, multi‑user, self‑hostable,
and installable as a mobile app (PWA).

[Features](#-features) · [Quick start](#-quick-start-docker-compose) · [Ubuntu/Debian](#-run-on-ubuntu--debian)
· [Kubernetes](#-deploy-on-kubernetes) · [Configuration](#-configuration) · [Mobile app](#-mobile-app-pwa)

</div>

---

## ✨ Features

- **Events** — create a trip (dates, participants), auto‑generated day × meal grid, modular tabs.
- **Menu planning** — drag & drop recipes onto a *day × Breakfast/Lunch/Dinner/Aperitif/Dessert* grid,
  1–3 recipes per slot with per‑recipe head counts, plus free‑form ingredient lines.
- **Recipes & cocktails** — shared library with photos, step‑by‑step instructions, fuzzy ingredient
  matching (avoids duplicates), multi‑tag categories, and a separate **Cocktails** section.
- **Voted lists** — “breakfast / on the slopes” style tabs where each participant picks a per‑day
  quantity (0–3); totals computed automatically × number of days.
- **Organizer lists** — “Apéro / Indispensables” style tabs (sections, no vote): organizers set a
  single total quantity per item for the whole trip. Reusable catalogs across events.
- **Shopping list** — auto‑consolidated from all planned recipes + tabs, deduplicated by ingredient,
  grouped by section, with provisioning source, “brought by”, notes and a got‑it checkbox.
- **Accommodations (Locations)** — participants propose places (beds, price, amenities, photos, map
  link) and **vote** (weighted podium); the winner becomes the event venue info.
- **Roles** — `User`, `Organizer`, `Admin` with granular permissions; invite‑only sign‑up with
  multi‑use / expiring invitations; manual account confirmation; admin impersonation for testing.
- **Real‑time** — changes appear instantly for everyone (Server‑Sent Events).
- **i18n** — French & English. **Themes** — light/dark/auto, 4 palettes, colorblind‑safe mode.
- **Mobile app (PWA)** — installable on iOS & Android; focused “Courses” mode (login → pick event →
  shopping list only).
- **Self‑hosted email** (optional SMTP), structured JSON logs, security headers / CSP / rate limiting.

## 🧱 Tech stack

| Layer | Tech |
|---|---|
| Frontend | React + Vite + TypeScript, Tailwind CSS, react‑i18next, @dnd‑kit |
| Backend | Go (chi router, GORM), REST JSON, JWT (httpOnly cookie), SSE |
| Database | PostgreSQL (CloudNativePG on Kubernetes) |
| Deploy | Docker Compose **or** Kubernetes (official images only — built at pod startup) |

## 📁 Project structure

```
.
├── backend/            # Go API (cmd/server, internal/{api,auth,db,models,seed,settings,sse,…})
├── frontend/           # React SPA (src/{pages,components,context,lib})
├── k8s/                # Kubernetes manifests (namespace, configmap, secret.example, postgres, …)
├── docker-compose.yml  # one‑command local stack
└── .env.example        # documented environment variables
```

---

## 🚀 Quick start (Docker Compose)

Requires Docker + Docker Compose v2.

```bash
git clone https://github.com/morphatrix/campmenu.git
cd campmenu
cp .env.example .env          # edit at least JWT_SECRET and BOOTSTRAP_ADMIN_*
#   for a quick test without email: set EMAIL_CONFIRM_REQUIRED=false
docker compose up --build
```

- Frontend → http://localhost:5173
- API → http://localhost:8080 (health: http://localhost:8080/healthz)

On first start the backend waits for PostgreSQL, runs migrations, creates the **first admin**
(`BOOTSTRAP_ADMIN_EMAIL` / `BOOTSTRAP_ADMIN_PASSWORD`) and seeds the recipe/cocktail library and the
*Apéro* / *Indispensables* catalogs. Log in, then **Admin → Invitations** to add everyone else.

> Without SMTP, confirmation/reset emails are written to the backend logs
> (`docker compose logs backend`). With `EMAIL_CONFIRM_REQUIRED=false`, the step is skipped.

---

## 🐧 Run on Ubuntu / Debian

Two paths on a fresh Ubuntu (20.04+) or Debian (11+) server. **Docker is the recommended way**; a
native build is documented below if you'd rather not use containers.

### Option A — Docker (recommended)

```bash
# 1. Install Docker Engine + Compose plugin (official convenience script)
sudo apt update && sudo apt install -y git curl
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker "$USER" && newgrp docker   # use docker without sudo

# 2. Get the app and configure
git clone https://github.com/morphatrix/campmenu.git
cd campmenu
cp .env.example .env
# Edit .env: set JWT_SECRET (openssl rand -base64 48) and BOOTSTRAP_ADMIN_*.
# For a quick test without an SMTP server, keep EMAIL_CONFIRM_REQUIRED=false.

# 3. Build and run (detached)
docker compose up --build -d
```

- Frontend → http://SERVER_IP:5173 · API → http://SERVER_IP:8080 (health: `/healthz`)
- Logs: `docker compose logs -f` · Stop: `docker compose down` (add `-v` to wipe the database)
- **Update:** `git pull && docker compose up --build -d`

The containers already **run as a service**: `restart: unless-stopped` + Docker enabled on boot
(`sudo systemctl enable --now docker`) means the stack comes back after a reboot. For a public host,
set `APP_URL` / `CORS_ORIGINS` to your domain and put a TLS reverse proxy (nginx/Caddy/Traefik) in front.

### Option B — Native (no Docker)

Install the toolchain (**Go ≥ 1.23**, **Node 20**, **PostgreSQL**):

```bash
sudo apt update && sudo apt install -y git curl postgresql

# Go — use the official tarball (distro packages often lag the 1.23 requirement)
curl -fsSL https://go.dev/dl/go1.23.4.linux-amd64.tar.gz | sudo tar -C /usr/local -xz
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh
export PATH=$PATH:/usr/local/go/bin

# Node 20 (NodeSource)
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs
```

Create the database and a user:

```bash
sudo -u postgres psql -c "CREATE USER campmenu WITH PASSWORD 'campmenu';"
sudo -u postgres psql -c "CREATE DATABASE campmenu OWNER campmenu;"
```

Build and run the **backend** (listens on `:8080`, runs migrations + seeds on first start):

```bash
git clone https://github.com/morphatrix/campmenu.git
cd campmenu/backend
go build -o campmenu ./cmd/server

DATABASE_DSN="host=localhost user=campmenu password=campmenu dbname=campmenu port=5432 sslmode=disable TimeZone=UTC" \
JWT_SECRET="$(openssl rand -base64 48)" \
BOOTSTRAP_ADMIN_EMAIL="you@example.com" \
BOOTSTRAP_ADMIN_PASSWORD="a-strong-password" \
EMAIL_CONFIRM_REQUIRED=false \
APP_URL="http://localhost:5173" CORS_ORIGINS="http://localhost:5173" \
./campmenu
```

Build and serve the **frontend** (in a second shell):

```bash
cd campmenu/frontend
npm install
echo 'VITE_API_URL=http://localhost:8080' > .env   # where the browser reaches the API
npm run build                                       # static files in dist/
npx serve -s dist -l 5173                            # or: npm run dev (hot reload)
```

#### Run as a service (systemd + nginx + HTTPS)

For a real server, run the backend under **systemd** and serve the built SPA behind **nginx** (which
proxies `/api` to the backend, so the browser uses one origin).

```bash
# Put the built app under /opt and run it as a dedicated user
sudo useradd --system --home /opt/campmenu --shell /usr/sbin/nologin campmenu
sudo mkdir -p /opt/campmenu && sudo cp -r campmenu/. /opt/campmenu/
sudo chown -R campmenu:campmenu /opt/campmenu
sudo mkdir -p /etc/campmenu
```

Create the backend's environment file `/etc/campmenu/campmenu.env` (mode `600`, owned by `campmenu`):

```ini
DATABASE_DSN=host=localhost user=campmenu password=campmenu dbname=campmenu port=5432 sslmode=disable TimeZone=UTC
JWT_SECRET=replace-with-openssl-rand-base64-48
APP_URL=https://campmenu.example.com
CORS_ORIGINS=https://campmenu.example.com
EMAIL_CONFIRM_REQUIRED=false
BOOTSTRAP_ADMIN_EMAIL=you@example.com
BOOTSTRAP_ADMIN_PASSWORD=a-strong-password
# Optional: encrypt sensitive settings at rest (openssl rand -base64 32)
SETTINGS_ENC_KEY=
```

Create the service unit `/etc/systemd/system/campmenu.service`:

```ini
[Unit]
Description=CampMenu backend
After=network.target postgresql.service
Wants=postgresql.service

[Service]
User=campmenu
WorkingDirectory=/opt/campmenu/backend
EnvironmentFile=/etc/campmenu/campmenu.env
ExecStart=/opt/campmenu/backend/campmenu
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable it (starts now and on every boot):

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now campmenu
sudo systemctl status campmenu          # check it's running; journalctl -u campmenu -f for logs
```

Build the frontend for **same-origin** (no `VITE_API_URL`, so the browser calls `/api` on your domain
and nginx proxies it):

```bash
cd /opt/campmenu/frontend
sudo -u campmenu npm install
sudo -u campmenu npm run build          # outputs /opt/campmenu/frontend/dist
```

nginx site `/etc/nginx/sites-available/campmenu` (then `ln -s` into `sites-enabled` and `nginx -t && systemctl reload nginx`):

```nginx
server {
    listen 80;
    server_name campmenu.example.com;
    root /opt/campmenu/frontend/dist;
    index index.html;

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;   # required so auth cookies get Secure over HTTPS
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_buffering off;                          # let /api/stream (SSE) flow
        proxy_read_timeout 3600s;
    }
    location / {
        try_files $uri $uri/ /index.html;             # SPA fallback
    }
}
```

Add HTTPS with Let's Encrypt:

```bash
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d campmenu.example.com
```

> After updating the code: `git pull`, rebuild (`go build -o campmenu ./cmd/server` and
> `npm run build`), then `sudo systemctl restart campmenu` and `sudo systemctl reload nginx`.

---

## ☸️ Deploy on Kubernetes

> **No custom images to build or push.** The manifests use only **official images**
> (`alpine`, `golang`, `node`, `nginx`, `postgres`). At pod startup, initContainers **clone this repo
> and build** the Go binary and the SPA into a shared volume, then run/serve them with the matching
> official image. Updates = push to your fork + `rollout restart`.

**Prerequisites:** a cluster with **ingress‑nginx** and the **CloudNativePG** operator installed, and a
StorageClass (default in `k8s/postgres.yml` is `freenas-nfs-csi` — change it to yours).

1. **Fork** this repo (the cluster clones from it). Then adjust in `k8s/`:
   - `configmap.yml` → `REPO_URL` (your fork), `REPO_REF`, `APP_SUBDIR` (empty for this layout),
     `APP_URL` / `CORS_ORIGINS` (your domain), and `GIT_SSL_NO_VERIFY` (`false` for GitHub).
   - `ingress.yml` → your host.
   - `postgres.yml` → your `storageClass`.
   - **Secrets:** `cp k8s/secret.example.yml k8s/secret.yml` and fill it (this file is gitignored).

2. **Apply:**

```bash
kubectl apply -f k8s/namespace.yml
kubectl apply -f k8s/configmap.yml
kubectl apply -f k8s/nginx-configmap.yml
kubectl apply -f k8s/secret.yml          # your filled copy (never committed)

kubectl apply -f k8s/postgres.yml
kubectl -n campmenu wait --for=condition=Ready cluster/campmenu-db --timeout=300s

kubectl apply -f k8s/backend.yml
kubectl apply -f k8s/frontend.yml
kubectl apply -f k8s/ingress.yml

# Follow the first build (clone → compile):
kubectl -n campmenu logs -f deploy/campmenu-backend  -c build
kubectl -n campmenu logs -f deploy/campmenu-frontend -c build
```

> ⚠️ **First start is slow** (clone + `go build` + `npm install/build` on every pod (re)creation) and
> needs internet access for Go/npm modules. Pods become Ready once the build finishes. The provided
> `Dockerfile`s remain if you later prefer pre‑built images.

> **Same host = simpler cookies & CORS.** The ingress routes `/api` to the backend and `/` to the
> frontend on one host → keep `API_URL: ""` (same‑origin). For a separate backend domain, set the
> origin in `API_URL` and add it to `CORS_ORIGINS`.

---

## 🔧 Configuration

Most settings are **editable live** in **Admin → Site settings** (name/logo, default theme & palette,
public URL, allowed origins, email confirmation, **SMTP**). They are stored in the database and
override the env defaults. Only `DATABASE_DSN`, `JWT_SECRET` (+ `BCRYPT_COST`, `PORT`) stay env‑only.

### Backend environment

| Variable | Default | Description |
|---|---|---|
| `APP_URL` | `http://localhost:5173` | Public frontend URL (invite/confirmation links) |
| `DATABASE_DSN` | local DSN | PostgreSQL DSN (`key=value` **or** `postgres://…`); on k8s, the CNPG `uri` secret key |
| `JWT_SECRET` | `change-me…` | JWT signing secret (**change it**, ≥ 32 chars) |
| `BCRYPT_COST` | `12` | bcrypt cost |
| `EMAIL_CONFIRM_REQUIRED` | `true` | `false` auto‑confirms new accounts |
| `CORS_ORIGINS` | `http://localhost:5173` | Allowed origins (comma‑separated) |
| `BOOTSTRAP_ADMIN_EMAIL` / `_PASSWORD` | — | First admin, created on startup if no admin exists |
| `SMTP_HOST` / `_PORT` / `_USER` / `_PASS` / `_FROM` | — | SMTP; empty `SMTP_HOST` logs emails instead of sending |
| `SITE_NAME` / `LOGO_URL` / `DEFAULT_THEME` / `DEFAULT_PALETTE` | CampMenu / … | Branding defaults |

### Kubernetes‑specific (ConfigMap)

`REPO_URL`, `REPO_REF`, `APP_SUBDIR` (build source), `GIT_SSL_NO_VERIFY`, and `API_URL`
(browser‑facing API origin, empty for same‑origin).

---

## 📱 Mobile app (PWA)

CampMenu installs on a phone like a real app — **no app store needed**.

- Open the site and go to **/install** (or “Install mobile app” on the login screen) for a QR code
  and step‑by‑step instructions.
- **iPhone:** open in **Safari** → Share → *Add to Home Screen* (third‑party iOS browsers can’t install PWAs).
- **Android:** open in Chrome/Brave → menu → *Install app*.

The installed app opens straight into the focused **Courses** mode: sign in → pick your event → shopping
list only, with filters (hide bought, by section, by “brought by”) and live updates.

---

## 👥 Roles & access

| | User | Organizer | Admin |
|---|---|---|---|
| Participate (vote, check items, propose locations, create recipes) | ✅ | ✅ | ✅ |
| Manage events, tabs, menus, lists, recipes, invitations | — | ✅ | ✅ |
| See all users (read) + promote a user to Organizer | — | ✅ | ✅ |
| Edit/reset/delete accounts, promote to Admin, **Site settings** | — | — | ✅ |

Sign‑up is **invite‑only**: an admin/organizer creates an invitation (single‑use or multi‑use,
optional expiry) and shares the link.

---

## 🔒 Security notes

- Passwords hashed with bcrypt; JWT in an **httpOnly** cookie (`Secure` under HTTPS, `SameSite=Lax`).
- CSRF defense (Origin/Host check) on writes, CORS allow‑list, login/global rate limiting.
- Security headers + a strict **Content‑Security‑Policy** on the SPA; `server_tokens off`.
- For production: serve over **HTTPS**, set a strong random `JWT_SECRET`, and change the bootstrap
  admin password after first login.

---

## 🤝 Contributing

Issues and pull requests welcome. The seed data (`backend/internal/seed/data.go`) ships a starter
recipe/cocktail library and reusable lists — adapt them to your group.

## 📄 License

MIT — see [`LICENSE`](LICENSE).
