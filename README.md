<div align="center">

# 🏕️ CampMenu

**Plan group-trip menus and shopping lists, together — and ditch the shared spreadsheet.**

Collaborative web app to organize meals, drinks, cocktails, shopping lists and accommodation
for trips with friends (ski, camping, festivals, road trips…). Real‑time, multi‑user, self‑hostable,
and installable as a mobile app (PWA).

[Features](#-features) · [Quick start](#-quick-start-docker-compose) · [Kubernetes](#-deploy-on-kubernetes)
· [Configuration](#-configuration) · [Mobile app](#-mobile-app-pwa)

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
