# Heat - Racing Application

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/SQLite-3.x-003B57?style=for-the-badge&logo=sqlite" alt="SQLite">
  <img src="https://img.shields.io/badge/Docker-20.10+-2496ED?style=for-the-badge&logo=docker" alt="Docker">
  <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="License">
</p>

> A lightweight Go-based racing management system for tracking racers, points, race results, and fastest lap times for the HEAT boardgame.

## ✨ Features

- 🏎️ **Racer Management** - Track racer profiles and statistics
- 🏁 **Race Tracking** - Record race information and results
- 📊 **Points System** - Automated points and rankings
- ⚡ **Fastest Laps** - Record and compare lap times
- 🌐 **Web Interface** - Clean HTML/CSS frontend
- 🐳 **Docker Ready** - Easy deployment with containers

## 🚀 Quick Start

```bash
# Clone and setup
git clone <repository-url>
cd heat

# Install dependencies
go mod download

# Build
CGO_ENABLED=1 go build -o heat-server .

# Run
./heat-server
```

Open [http://localhost:8080](http://localhost:8080) in your browser.

## 📋 Prerequisites

| Component | Version | Notes |
|-----------|---------|-------|
| **Go** | 1.21+ | Backend runtime |
| **SQLite3** | - | Development libraries required |
| **GCC** | - | For CGO compilation |
| **Docker** | 20.10+ | Optional, for containerized deployment |
| **Docker Compose** | 1.29+ | Optional |

### Linux Installation

```bash
sudo apt-get update
sudo apt-get install -y build-essential libsqlite3-dev
```

### macOS Installation

```bash
# With Homebrew
brew install go sqlite3
xcode-select --install
```

## 🏗️ Project Structure

```
.
├── main.go               # Application entry point & API handlers
├── go.mod                # Go module definition
├── go.sum                # Dependency checksums
├── heat.db               # SQLite database (created at runtime)
├── Dockerfile            # Multi-stage Docker build
├── docker-compose.yml    # Docker Compose orchestration
├── static/
│   ├── index.html        # Main web interface
│   ├── admin.html        # Admin dashboard
│   └── style.css         # Styling
└── README.md             # This file
```

## 🔌 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/racers` | List all racers |
| `POST` | `/api/racers` | Create/update a racer |
| `DELETE` | `/api/racers` | Delete a racer |
| `GET` | `/` | Web interface |

## 🐳 Docker Deployment

### Build Image

```bash
docker build -t heat-server .
```

### Run with Docker Compose

```bash
# Start the application
docker-compose up -d

# View logs
docker-compose logs -f heat-server

# Stop the application
docker-compose down
```

The application will be available at [http://localhost:8080](http://localhost:8080)

### Data Persistence

- **Local**: Database stored as `heat.db` in the working directory
- **Docker**: Database stored in the `heat-data` volume

To remove persistent data:
```bash
docker-compose down -v
```

## 🛠️ Development

```bash
# Stop current container
docker-compose down

# Rebuild and start
docker-compose build && docker-compose up -d
```

## ⚠️ Troubleshooting

### Build Errors

| Error | Solution |
|-------|----------|
| `no C compiler found` | Install build essentials: `sudo apt-get install build-essential` |
| `sqlite3.h not found` | Install dev packages: `sudo apt-get install libsqlite3-dev` |

### Runtime Errors

| Issue | Solution |
|-------|----------|
| Port 8080 in use | Change port in `docker-compose.yml` |
| Database locked | Ensure only one instance is running |

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.