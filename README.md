# EyesOn Go Server

SIM Card Management Dashboard built with Go Fiber + React + TypeScript

## Quick Start

```bash
# 1. Start Go server
cd eyeson-go-server
set PORT=3000
go run cmd/server/main.go

# 2. Open browser
http://127.0.0.1:3000
Login: admin / admin
```

## Project Structure

```
eyeson-go/
 eyeson-go-server/      # Go Fiber backend
    cmd/server/        # Entry point
    internal/          # Handlers, models, routes
    static/            # Built React frontend
 eyeson-gui/            # React/TypeScript frontend
    frontend/
 ARCHITECTURE.md        # Architecture documentation
 AGENT_SKILLS.md        # AI Agent skills & methodology
 PROJECT_STRUCTURE.md   # Detailed project structure
```

## Tech Stack

- **Backend**: Go 1.21+, Fiber v2.52, GORM, SQLite
- **Frontend**: React 18, TypeScript, Vite, Bootstrap 5
- **Auth**: JWT (24h), bcrypt passwords, RBAC

## Development

```bash
# Frontend hot-reload
cd eyeson-gui/frontend
npm run dev

# Build frontend for production
npm run build
xcopy dist\* ..\eyeson-go-server\static\ /E /Y
```

## API Documentation

See [ARCHITECTURE.md](ARCHITECTURE.md) for full API reference.
