# EyesOn SIM Management System

> **Copyright (c) 2026 Alexander G. (Samsonix)**  
> **License: MIT**

A full-stack SIM card management dashboard built with Go (Fiber) and React (TypeScript), featuring reactive data pipelines (RxGo) and real-time Server-Sent Events.

![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)
![React Version](https://img.shields.io/badge/React-18-61DAFB?style=flat&logo=react)
![License](https://img.shields.io/badge/License-MIT-green.svg)

---

## Quick Start

### Option 1: With Simulator (Recommended for Testing)

```bash
# Terminal 1: Start Simulator
cd pelephone-simulator
.\run.bat
# → http://localhost:8888/web

# Terminal 2: Start Server
.\build_and_run.bat
# → http://localhost:5000
```

### Option 2: With Real Pelephone API

Edit `eyeson-go-server/.env` and set your Pelephone credentials, then run `build_and_run.bat`.

### Login Credentials

| Username | Password | Role |
|----------|----------|------|
| admin | admin | Administrator |

---

## Features

- **SIM Management** — View, filter, sort, and search SIM cards
- **Reactive Search** — Client-side debounced search with field-specific filters (`field:query`)
- **Real-time Events** — Server-Sent Events via fan-out EventBroadcaster
- **Bulk Operations** — Activate/suspend multiple SIMs at once
- **Queue System** — Background task processing with retry logic
- **Auto-Sync** — Data synchronized from upstream API after task completion
- **Live Countdown** — Real-time countdown to next scheduled task
- **Job Tracking** — Monitor provisioning job history
- **User Management** — Create, edit, delete users (Admin)
- **Role-Based Access** — Administrator, Moderator, Viewer roles
- **Google OAuth** — Optional Google account linking
- **Audit Logging** — Full audit trail with CSV export
- **VS Code Themes** — Dark+ and Light+ color schemes
- **API Documentation** — Swagger UI at `/docs`
- **Test Console** — Interactive reactive endpoint tester at `/test-reactive.html`

---

## Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Browser    │────▶│  Go Server   │────▶│  Pelephone   │
│  React SPA   │◀────│ Fiber :5000  │◀────│  API :8888   │
└──────────────┘ SSE └──────┬───────┘     └──────────────┘
                           │
                    ┌──────▼──────┐
                    │   SQLite    │
                    │  (eyeson.db)│
                    └─────────────┘
```

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed system design.  
See [docs/REACTIVE_ARCHITECTURE.md](docs/REACTIVE_ARCHITECTURE.md) for reactive layer documentation.

---

## Project Structure

```
eyeson-go/
├── eyeson-go-server/          # Go Fiber backend
│   ├── cmd/server/            # Entry point
│   ├── internal/
│   │   ├── config/            # App configuration
│   │   ├── database/          # SQLite + GORM
│   │   ├── eyesont/           # Pelephone API client
│   │   ├── handlers/          # HTTP handlers (REST + reactive + SSE)
│   │   ├── jobs/              # Background task worker
│   │   ├── models/            # GORM models + API DTOs
│   │   ├── reactive/          # RxGo streams, EventBroadcaster, SimRepository
│   │   ├── routes/            # Route registration
│   │   ├── services/          # Queue, audit, upstream services
│   │   └── syncer/            # Background data syncer
│   └── static/                # React build + Swagger + test console
├── eyeson-gui/
│   └── frontend/              # React 18 / TypeScript / Vite source
├── pelephone-simulator/       # Standalone API simulator
│   └── web/                   # Simulator admin panel
├── tools/                     # Dev utilities & scripts
│   ├── authtest/              # OAuth test tool
│   ├── extract_pelephone_spec.py
│   └── generate_upstream_spec.py
├── docs/                      # Documentation
│   ├── ARCHITECTURE.md        # System architecture
│   ├── REACTIVE_ARCHITECTURE.md
│   ├── TESTING_REPORT.md
│   ├── DEVELOPMENT_RULES.md
│   └── design/                # Design documents (billing, subscriptions, hierarchy)
├── build_and_run.bat          # Build server and start
├── rebuild_ui.bat             # Rebuild frontend only
├── run_simulator.bat          # Start simulator
└── README.md                  # This file
```

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.24, Fiber v2.52, GORM v1.31, SQLite |
| Reactive | RxGo v2.5.0 (Observable streams, SSE broadcaster) |
| Frontend | React 18, TypeScript 5, Vite 4, Bootstrap 5 |
| Auth | JWT (24h), bcrypt, RBAC, Google OAuth |
| Docs | OpenAPI 3.0, Swagger UI |

---

## API Endpoints

### Core REST API

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/auth/login | Authenticate |
| GET | /api/v1/sims | List SIMs (paginated) |
| POST | /api/v1/sims/bulk-status | Bulk status change |
| GET | /api/v1/sims/:msisdn/history | SIM history |
| GET | /api/v1/jobs/queue | Task queue |
| GET | /api/v1/stats | Statistics |
| GET | /api/v1/audit | Audit logs (Admin) |
| GET | /docs | Swagger UI |

### Reactive API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/reactive/events | SSE event stream (real-time) |
| GET | /api/v1/reactive/sims | SIM list via Observable pipeline |
| GET | /api/v1/reactive/search?q= | Reactive search (supports `field:query`) |
| GET | /api/v1/reactive/stats | Aggregated event statistics |

---

## Configuration

Edit `eyeson-go-server/.env`:

```dotenv
# Server
PORT=5000

# API (Simulator by default)
EYESON_API_BASE_URL=http://127.0.0.1:8888
EYESON_API_USERNAME=admin
EYESON_API_PASSWORD=admin
EYESON_API_DELAY_MS=10

# For Real Pelephone API:
# EYESON_API_BASE_URL=https://eot-portal.pelephone.co.il:8888
# EYESON_API_USERNAME=your_username
# EYESON_API_PASSWORD=your_password
# EYESON_API_DELAY_MS=1000
```

---

## Development

### Build Frontend
```bash
cd eyeson-gui/frontend
npm install
npm run build
```

### Build Backend
```bash
cd eyeson-go-server
go build -o eyeson-go-server.exe ./cmd/server
```

### Quick Scripts
- `build_and_run.bat` — Build server and start
- `rebuild_ui.bat` — Rebuild frontend and copy to server
- `run_simulator.bat` — Start Pelephone API simulator

---

## License

MIT License — see [LICENSE](LICENSE) for details.

## Author

**Alexander G. (Samsonix)**

- GitHub: [@alexgavs](https://github.com/alexgavs)

---

Built with Go, React, and RxGo
