# VetFlow Backend — Veterinary Management System API

A production-ready REST API for managing veterinary clinics, built with **Go** and
**MongoDB**, following SOLID principles and an MVC-style architecture.

> This is the backend of **VetFlow**. The umbrella repository
> [`vetflow`](https://github.com/IannCarvalho/vetflow) is a **map of four
> independent repositories** (backend, frontend, agent, MCP) — not a monorepo.
> For the web client, see
> [`vetflow-frontend`](https://github.com/IannCarvalho/vetflow-frontend).

> **Note — no LiteLLM here.** This backend is written in **Go** and never imports
> `litellm`; the LLM lives entirely in the
> [`vetflow-agent`](https://github.com/IannCarvalho/vetflow-agent) (Python) service. The agent's
> import-ordering rule — import `litellm` right after loading the env, and at
> module top-level, because LiteLLM reads environment variables at **import
> time** — is documented in the agent's README and does **not** apply to this
> repo. Cross-referenced here only so the detail isn't looked for in the wrong
> place.

---

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Setup](#setup)
- [Environment Variables](#environment-variables)
- [Running](#running)
- [Testing](#testing)
- [Build](#build)
- [API Docs](#api-docs)
- [Package Documentation](#package-documentation)
- [Git Hooks](#git-hooks--merge-conflict-marker-gate)
- [License](#license)

---

## Features

- **Authentication** — email/password with JWT, email verification, social login (OAuth) and OAuth callback flow
- **User Management** — profile and profile-picture updates, password change/set (protected routes)
- **Client Management** — register and manage clinic clients
- **Pet Management** — track pets (species, breed, age, weight), cascading deletes
- **Consultation Records** — anamnesis, diet and physiological data
- **Vaccine Tracking** — batch info and next-dose scheduling
- **Notifications** — generated for upcoming consultations/vaccines, with an SMTP email scheduler
- **MongoDB Integration** — flexible NoSQL storage with explicit index creation
- **Demo mode** (`APP_ENV=demo`) — a public, throwaway deployment with a one-click
  visitor login, fake data seeded on boot, auto-reset on boot (re-seeds when the
  data is older than `DEMO_RESET_INTERVAL_HOURS`) and a protected
  `POST /admin/reset-demo` endpoint (`X-Admin-Token`) for a scheduled wipe/reseed
- **Production access control** — new-account creation is gated in production via
  an invite code (`REGISTER_INVITE_CODE`) and an account cap (`MAX_PRODUCTION_USERS`),
  so free-tier infra stays controlled; open signup remains in demo/development
- **Docker Support** — `docker-compose` for one-command local runs
- **SOLID / Interface-based design** — repository pattern + dependency injection for testability

---

## Architecture

**Pattern:** MVC adapted for REST, with dependency injection, the repository pattern
for data-access abstraction, and interface-based design for testability.

The codebase is split into focused packages. Each package has its own README with
layer-specific detail — see [Package Documentation](#package-documentation) below.

```
cmd/           → Entry point & server wiring
config/        → Environment configuration loading
db/            → MongoDB connection, indexes, seed data
models/        → Domain structs & validation
repositories/  → Data-access layer (interfaces + Mongo implementations)
services/      → Business logic (notification generation & email scheduler)
controllers/   → HTTP handlers
routes/        → Router, middleware, rate limiter
helpers/       → Shared utilities (JWT, images, passwords, email, validation)
```

Request flow: `routes` → `controllers` → `repositories`/`services` → `models` → MongoDB (`db`).

```mermaid
flowchart TD
    client(["HTTP client<br/>(frontend · vetflow-mcp)"])

    subgraph boot["Bootstrap — cmd/"]
        wiring["main.go<br/>DI wiring · server start"]
    end

    subgraph http["HTTP layer"]
        router["routes/<br/>chi router"]
        mw["middleware<br/>JWT auth · rate limiter"]
        ctrl["controllers/<br/>HTTP handlers · bind + validate"]
    end

    subgraph domain["Domain layer"]
        repo["repositories/<br/>interfaces + Mongo impl"]
        svc["services/<br/>business logic"]
        models["models/<br/>structs · validation"]
    end

    subgraph infra["Infra"]
        db["db/<br/>connection · indexes · seed"]
        mongo[("MongoDB")]
        smtp["SMTP email scheduler<br/>(services/ · async)"]
    end

    helpers["helpers/<br/>JWT · images · passwords · email · validation"]

    client -->|"request"| router
    router --> mw
    mw --> ctrl
    ctrl --> repo
    ctrl --> svc
    repo --> models
    svc --> models
    svc -.-> repo
    repo --> db
    db --> mongo
    svc -. "daily reminders" .-> smtp
    smtp -. "sends email" .-> client

    wiring -. "injects deps into" .-> ctrl
    wiring -. "starts" .-> svc
    mw -. "uses" .-> helpers
    ctrl -. "uses" .-> helpers

    classDef go fill:#2f855a,stroke:#1c4532,color:#fff;
    classDef store fill:#2b6cb0,stroke:#1a365d,color:#fff;
    classDef ext fill:#b7791f,stroke:#744210,color:#fff;
    class wiring,router,mw,ctrl,repo,svc,models,db,helpers go;
    class mongo store;
    class client,smtp ext;
```

Each arrow crosses one layer boundary only: controllers never touch Mongo
directly (they go through repositories), and the domain layer never parses HTTP.
Dependencies are wired in `cmd/` and injected downward, so each layer depends on
the **interface** of the one below it — the core of the testable, SOLID design.

---

## Prerequisites

- Go 1.22 or higher
- Docker and Docker Compose (optional, for containerized runs)
- MongoDB 7.0+ (if running without Docker)

---

## Setup

Run all commands from this directory (`vetflow-backend/`).

### 1. Install dependencies

```bash
go mod download
```

### 2. Create your `.env`

```powershell
# Windows (PowerShell)
copy .env.example .env
```

```bash
# macOS / Linux
cp .env.example .env
```

Then edit `.env` — see [Environment Variables](#environment-variables).

---

## Environment Variables

Source of truth is [`.env.example`](.env.example). Key variables:

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `MONGO_URI` | yes | — | MongoDB connection string (local or Atlas) |
| `MONGO_DB` | yes | `vetflow` | Database name. A non-production `APP_ENV` must **not** point this at the production database `vetflow` (the app refuses to start) — use e.g. `vetflow_demo` |
| `PORT` | no | `8080` | HTTP listen port |
| `APP_ENV` | no | `development` | `development`, `demo` or `production` |
| `JWT_SECRET` | **yes (all envs)** | — | JWT signing secret. The app **refuses to start** without it — no dev fallback. Generate with `openssl rand -hex 32` |
| `CORS_ALLOWED_ORIGINS` | no | `*` | Comma-separated allowed origins; empty allows all |
| `AUTH_RATE_LIMIT_PER_MIN` | no | `10` | Per-IP/minute limit on `/auth/login` & `/auth/register`; `0` disables |
| `SEED_FORCE` | no | `false` | When `true`, forces a clean wipe + re-seed on startup (non-production only) |
| `ADMIN_RESET_TOKEN` | no | — | Enables `POST /admin/reset-demo`; callers must present this value in the `X-Admin-Token` header. Honored only outside production; leave unset in prod |
| `DEMO_RESET_INTERVAL_HOURS` | no | `6` | Demo only: on boot, re-seed if the last seed is older than this many hours (`0` = reset every boot). Keeps the public demo self-cleaning with no external scheduler |
| `REGISTER_INVITE_CODE` | no | — | **Production only**: invite code required by `POST /auth/register`. If unset in production, classic registration is **closed**. Ignored outside production |
| `MAX_PRODUCTION_USERS` | no | `0` | **Production only**: cap on total accounts (classic + social + OAuth); `0`/unset = no cap. Ignored outside production |
| `EMAIL_NOTIFICATIONS_ENABLED` | no | `true` | Enables the daily email reminder scheduler |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASS` / `SMTP_FROM` / `SMTP_FROM_NAME` | no | — | SMTP settings; if `SMTP_HOST` is empty the mailer runs in log-only mode |

---

## Running

### With Docker Compose (recommended)

[`docker-compose.yml`](docker-compose.yml) builds the image, injects your `.env`, and
requires `JWT_SECRET` to be set (the container fails fast otherwise).

```bash
docker-compose up -d
# API available at http://localhost:8080
```

Set `SEED_FORCE=true` in your shell to force a clean re-seed on startup.

### Locally (Go toolchain)

Start MongoDB (e.g. via Docker):

```powershell
docker run -d -p 27017:27017 `
  -e MONGO_INITDB_ROOT_USERNAME=admin `
  -e MONGO_INITDB_ROOT_PASSWORD=admin `
  --name vetflow-mongo mongo:7.0
```

Point `.env` at it (`MONGO_URI=mongodb://admin:admin@localhost:27017`), then:

```bash
go run ./cmd/main.go
# API available at http://localhost:8080
```

---

## Testing

```bash
# Run the full test suite
go test ./...

# Vet / format
go vet ./...
go fmt ./...
```

---

## Build

```powershell
# Windows
go build -o vetflow.exe ./cmd/main.go
```

```bash
# macOS / Linux
go build -o vetflow ./cmd/main.go
```

```bash
# Docker image
docker build -t vetflow:latest .
```

---

## API Docs

Interactive Swagger docs are served at `GET /swagger` (redirects to
`/swagger/index.html`). The spec is generated into [`docs/`](docs/).

Endpoint and route-level detail lives in the [`routes/`](routes/README.md) and
[`controllers/`](controllers/README.md) package READMEs.

---

## Package Documentation

Each package documents only its own layer. Start at the layer you are working in:

| Package | Responsibility | README |
| --- | --- | --- |
| `cmd/` | Application entry point and server wiring | [cmd/README.md](cmd/README.md) |
| `config/` | Environment configuration loading & validation | [config/README.md](config/README.md) |
| `db/` | MongoDB connection, index creation, data seeding | [db/README.md](db/README.md) |
| `models/` | Domain structs, validation, error types | [models/README.md](models/README.md) |
| `repositories/` | Data-access layer (interfaces + Mongo implementations) | [repositories/README.md](repositories/README.md) |
| `services/` | Business logic: notification generation & email scheduler | [services/README.md](services/README.md) |
| `controllers/` | HTTP request handlers | [controllers/README.md](controllers/README.md) |
| `routes/` | Router, auth middleware, rate limiter | [routes/README.md](routes/README.md) |
| `helpers/` | Shared utilities: JWT, images, passwords, email, validation | [helpers/README.md](helpers/README.md) |

---

## Git Hooks — Merge Conflict Marker Gate

This repo ships a tracked `pre-commit` hook that refuses any commit containing
unresolved git merge conflict markers (`<<<<<<<`, `=======`, `>>>>>>>`). The same
check runs in CI as the `no-conflict-markers` job in
[`.github/workflows/ci.yml`](.github/workflows/ci.yml), so a commit that passes
locally also passes CI.

### Activate the hook (once per clone)

The hook lives in the tracked `.githooks/` directory (git does not share
`.git/hooks`). Point git at it with the provided script:

```bash
# macOS / Linux / Git Bash
bash scripts/install-hooks.sh
```

```powershell
# Windows PowerShell
powershell -ExecutionPolicy Bypass -File scripts\install-hooks.ps1
```

Both run `git config core.hooksPath .githooks`. Git for Windows runs the bash hook
through its bundled `sh`, so no extra setup is required.

### Behaviour

- On `git commit`, the hook scans the **staged** content of every added/modified
  file. If any still contains a conflict marker, the commit is aborted and the
  offending `file:line` locations are printed.
- Emergency bypass (discouraged): `git commit --no-verify`.
- The gate is enforced regardless of the hook in CI, so a bypassed local commit
  is still rejected on push / pull request.

---

## License

MIT License

---

**Built with Go, MongoDB, and ❤️ for veterinary professionals.**
