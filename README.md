# EyesOn SIM Management System

A full-stack SIM card management dashboard built with Go (Fiber) and React (TypeScript).

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![React Version](https://img.shields.io/badge/React-18-61DAFB?style=flat&logo=react)
![License](https://img.shields.io/badge/License-MIT-green.svg)

## 🚀 Quick Start

```bash
# Start the server
cd eyeson-go-server
.\server.exe

# Open in browser
http://127.0.0.1:5000

# Login credentials
Username: admin
Password: admin123
```

## ✨ Features

- **SIM Management** - View, filter, sort, and search SIM cards
- **Bulk Operations** - Activate/suspend multiple SIMs at once
- **Job Tracking** - Monitor provisioning job history
- **User Management** - Create, edit, delete users (Admin)
- **Role-Based Access** - Administrator, Moderator, Viewer roles
- **VS Code Themes** - Dark+ and Light+ color schemes
- **API Documentation** - Swagger UI at `/docs`
- **Localization** - English and Russian support

## 🏗️ Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│    Browser   │────▶│  Go Server   │────▶│  Pelephone   │
│  React SPA   │◀────│  Fiber :5000 │◀────│  EyesOnT API │
└──────────────┘     └──────────────┘     └──────────────┘
```

## 📁 Project Structure

```
eyeson-go/
├── eyeson-go-server/      # Go Fiber backend
│   ├── cmd/server/        # Entry point
│   ├── internal/          # Handlers, models, routes
│   └── static/            # React build + Swagger
├── eyeson-gui/            # React/TypeScript frontend
│   └── frontend/
├── ARCHITECTURE.md        # System architecture
├── AGENT_SKILLS.md        # AI Agent knowledge base
└── PROJECT_STRUCTURE.md   # Detailed structure
```

## 🔧 Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.21+, Fiber v2.52.10, GORM, SQLite |
| Frontend | React 18, TypeScript, Vite, Bootstrap 5 |
| Auth | JWT (24h), bcrypt, RBAC |
| Docs | OpenAPI 3.0, Swagger UI |

## 📡 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/auth/login | Authenticate |
| GET | /api/v1/sims | List SIMs |
| POST | /api/v1/sims/bulk-status | Bulk status change |
| GET | /api/v1/jobs | Job history |
| GET | /api/v1/stats | Statistics |
| GET | /docs | Swagger UI |

Full API documentation available at `http://localhost:5000/docs`

## ⚙️ Configuration

Create a `.env` file in `eyeson-go-server/`:

```dotenv
# API Base URL (Mock or Production)
EYESON_API_BASE_URL=http://127.0.0.1:8888

# API Credentials
EYESON_API_USERNAME=your_username
EYESON_API_PASSWORD=your_password

# Performance Tuning
EYESON_API_DELAY_MS=10  # Delay between requests (ms)
```

## 🛠️ Development

### Prerequisites

- Go 1.21+
- Node.js 18+
- npm or yarn

### Build Frontend

```bash
cd eyeson-gui/frontend
npm install
npm run build

# Copy to backend
Copy-Item "dist\*" "..\..\eyeson-go-server\static\" -Recurse -Force
```

### Build Backend

```bash
cd eyeson-go-server
go build -o server.exe ./cmd/server
.\server.exe
```

### Full Rebuild (PowerShell)

```powershell
cd eyeson-gui/frontend
npm run build
Copy-Item "dist\*" "..\..\eyeson-go-server\static\" -Recurse -Force
cd ..\..\eyeson-go-server
go build -o server.exe ./cmd/server
.\server.exe
```

## 🎨 Themes

The application includes VS Code-inspired themes:

- **Dark+** (default) - Dark theme optimized for readability
- **Light+** - Light theme for bright environments

Toggle in Profile settings or use the theme button in navbar.

## 📖 Documentation

| Document | Description |
|----------|-------------|
| [ARCHITECTURE.md](ARCHITECTURE.md) | System architecture & API reference |
| [PROJECT_STRUCTURE.md](PROJECT_STRUCTURE.md) | Detailed file structure |
| [AGENT_SKILLS.md](AGENT_SKILLS.md) | AI Agent knowledge base |
| `/docs` | Swagger API documentation |

## 🔐 Default Credentials

| Username | Password | Role |
|----------|----------|------|
| admin | admin123 | Administrator |

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Open a Pull Request

---

Built with ❤️ using Go and React
