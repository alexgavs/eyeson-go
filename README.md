# EyesOn SIM Management System

> **Copyright (c) 2026 Alexander G. (Samsonix)**  
> **License: MIT**

A full-stack SIM card management dashboard built with Go (Fiber) and React (TypeScript).

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![React Version](https://img.shields.io/badge/React-18-61DAFB?style=flat&logo=react)
![License](https://img.shields.io/badge/License-MIT-green.svg)

---

## 🚀 Quick Start

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
| admin | admin123 | Administrator |

---

## ✨ Features

- **SIM Management** - View, filter, sort, and search SIM cards
- **Bulk Operations** - Activate/suspend multiple SIMs at once  
- **Queue System** - Background task processing with retry logic
- **Auto-Sync** - Data synchronized from API after task completion
- **Live Countdown** - Real-time countdown to next scheduled task
- **Job Tracking** - Monitor provisioning job history
- **User Management** - Create, edit, delete users (Admin)
- **Role-Based Access** - Administrator, Moderator, Viewer roles
- **VS Code Themes** - Dark+ and Light+ color schemes
- **API Documentation** - Swagger UI at `/docs`

---

## 🏗️ Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Browser    │────▶│  Go Server   │────▶│  Pelephone   │
│  React SPA   │◀────│ Fiber :5000  │◀────│  API :8888   │
└──────────────┘     └──────┬───────┘     └──────────────┘
                           │
                    ┌──────▼──────┐
                    │   SQLite    │
                    │  (eyeson.db)│
                    └─────────────┘
```

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed system design.

---

## 📁 Project Structure

```
eyeson-go/
├── eyeson-go-server/      # Go Fiber backend
│   ├── cmd/server/        # Entry point
│   ├── internal/          # Handlers, models, jobs, syncer
│   └── static/            # React build + Swagger
├── eyeson-gui/            # React/TypeScript frontend
│   └── frontend/          # Vite project
├── pelephone-simulator/   # Standalone API simulator
│   └── web/               # Admin panel
├── ARCHITECTURE.md        # System architecture
├── README.md              # This file
├── build_and_run.bat      # Build and start server
├── rebuild_ui.bat         # Rebuild frontend only
└── run_simulator.bat      # Start simulator
```

---

## 🔧 Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.21+, Fiber v2, GORM, SQLite |
| Frontend | React 18, TypeScript, Vite, Bootstrap 5 |
| Auth | JWT (24h), bcrypt, RBAC |
| Docs | OpenAPI 3.0, Swagger UI |

---

## 📡 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/auth/login | Authenticate |
| GET | /api/v1/sims | List SIMs |
| POST | /api/v1/sims/bulk-status | Bulk status change |
| GET | /api/v1/jobs/queue | Task queue |
| GET | /api/v1/sims/:msisdn/history | SIM history |
| GET | /api/v1/stats | Statistics |
| GET | /docs | Swagger UI |

---

## ⚙️ Configuration

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

## 🛠️ Development

### Build Frontend
```bash
cd eyeson-gui/frontend
npm install
npm run build
```

### Build Backend
```bash
cd eyeson-go-server
go build -o eyeson-server.exe cmd/server/main.go
```

### Quick Scripts
- `build_and_run.bat` - Build server and start
- `rebuild_ui.bat` - Rebuild frontend and copy to server
- `run_simulator.bat` - Start Pelephone API simulator

---

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

## 👤 Author

**Alexander G. (Samsonix)**

- GitHub: [@alexgavs](https://github.com/alexgavs)

---

Built with ❤️ using Go and React
