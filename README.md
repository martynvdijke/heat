# Heat - Pedal to the Metal Companion

<p align="center">
  <img src="https://img.shields.io/badge/Board_Game-Heat%3A_Pedal_to_the_Metal-red?style=for-the-badge" alt="Heat Board Game">
  <img src="https://img.shields.io/badge/Status-Inspired_Companion-orange?style=for-the-badge" alt="Inspired Companion">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go" alt="Go Version">
</p>

> This is a fan-made, digital "side-along" companion app for the **Heat: Pedal to the Metal** board game. It is designed to enhance your tabletop experience by providing real-time race tracking, championship management, and interactive map visualization.

## 🏁 Board Game Companion Features

- 🏎️ **Racer Tracking** - Keep track of all players' positions and status on the board.
- 🗺️ **Interactive Maps** - Use official-style GeoJSON tracks or upload a photo of your own game board to use as a background.
- 🤖 **AI Board Extraction** - (Alpha) Snap a photo of your board and let the AI attempt to trace the track spaces for digital tracking.
- 📊 **Championship Management** - Automated points calculation and historical record-keeping across multiple race seasons.
- ⚡ **Digital Dashboard** - A live, synchronized view for all players to see gaps, rankings, and fastest laps.

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

- **Local**: Database stored as `heat.db` in the working directory; images in `static/images`
- **Docker**: Database stored in `heat-db` volume at `/db/heat.db`; images in `heat-data` volume at `/app/images`

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