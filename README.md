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
- 📈 **Statistics & Analytics** - Advanced stats including head-to-head comparisons, points progression charts, streaks, ELO ratings, and CSV export.

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

Open [http://localhost:6270](http://localhost:6270) in your browser.

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
├── main.go               # Application entry point & route setup
├── main_test.go          # Integration tests
├── paths_test.go         # Path configuration tests
├── go.mod / go.sum       # Go module definition
├── heat.db               # SQLite database (created at runtime)
├── Dockerfile            # Multi-stage Docker build
├── docker-compose.yml    # Docker Compose orchestration
├── Taskfile.yml          # Task runner
├── app/                  # Application state (DB, sessions, globals)
│   └── app.go
├── db/                   # Database initialization & migrations
│   └── db.go
├── handlers/             # HTTP handlers by domain
│   ├── auth.go           # Login, logout, setup
│   ├── racers.go         # Racer CRUD
│   ├── race.go           # Race info & history
│   ├── tracks.go         # Track CRUD & AI extraction
│   ├── upload.go         # File upload & management
│   ├── quotes.go         # Quote management
│   ├── settings.go       # AI, email, notification, Umami settings
│   ├── email.go          # Email sending & racer emails
│   └── stats.go          # Statistics & analytics
├── middleware/            # HTTP middleware
│   └── middleware.go      # Auth, rate limiting, security headers, Umami
├── models/               # Data structures
│   └── models.go
├── ws/                   # WebSocket broadcasting
│   └── ws.go
├── static/               # Frontend assets
│   ├── index.html        # Main web interface
│   ├── admin.html        # Admin dashboard
│   ├── controller.html   # Race controller view
│   ├── stats.html        # Statistics view
│   ├── chat.html         # Commentary chat
│   ├── style.css         # Styling
│   └── js/               # Compiled TypeScript
├── ts/                   # TypeScript source
└── tests/                # Playwright e2e tests
```

## 🔌 API Endpoints

### Race Management

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/api/racers` | No | List all racers |
| `POST` | `/api/racers` | Yes | Create/update a racer |
| `DELETE` | `/api/racers` | Yes | Delete a racer |
| `GET` | `/api/race-info` | No | Get current race info |
| `POST` | `/api/race-info` | Yes | Update race info |
| `GET` | `/api/race-history` | No | Get race history (season or oneoff) |
| `POST` | `/api/race-history` | Yes | Save a race result to history |
| `DELETE` | `/api/race-history` | Yes | Delete a race history entry |
| `GET` | `/api/oneoff-races` | No | Get one-off (non-season) races |
| `DELETE` | `/api/oneoff-races` | Yes | Delete a one-off race |

### Track Management

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/api/tracks` | No | List all tracks |
| `POST` | `/api/tracks` | Yes | Create/update a track |
| `DELETE` | `/api/tracks` | Yes | Delete a track |
| `POST` | `/api/tracks/ai-extract` | Yes | AI track extraction from image |

### Statistics & Analytics

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/api/racer-stats` | No | Get racer career statistics |
| `POST` | `/api/racer-stats` | Yes | Update racer statistics manually |
| `GET` | `/api/track-stats` | No | Get per-track statistics (wins, races, fastest laps) |
| `GET` | `/api/stats/head-to-head` | No | Head-to-head comparison between two racers |
| `GET` | `/api/stats/points-progression` | No | Cumulative points progression over a season |
| `GET` | `/api/stats/streaks` | No | Win, podium, and DNF streaks for all racers |
| `GET` | `/api/stats/elo` | No | ELO-style ratings based on race results |
| `GET` | `/api/stats/export` | No | Export all racer stats as CSV |
| `GET` | `/api/stats/track-performance` | No | Per-track performance breakdown by racer |

### Settings & Admin

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/api/login` | No* | Login or create admin account |
| `POST` | `/api/logout` | No | Logout and clear session |
| `GET` | `/api/check-setup` | No | Check if admin is configured |
| `GET` | `/api/ai-settings` | Yes | Get AI extraction settings |
| `POST` | `/api/ai-settings` | Yes | Update AI extraction settings |
| `GET` | `/api/notification-settings` | Yes | Get Gotify notification settings |
| `POST` | `/api/notification-settings` | Yes | Update notification settings |
| `POST` | `/api/test-notification` | Yes | Send a test notification |
| `GET` | `/api/email-settings` | Yes | Get SMTP email settings |
| `POST` | `/api/email-settings` | Yes | Update SMTP email settings |
| `GET` | `/api/racer-emails` | Yes | Get racer email addresses |
| `POST` | `/api/racer-emails` | Yes | Save a racer's email address |
| `POST` | `/api/send-race-email` | Yes | Manually send race results email |
| `GET` | `/api/umami-settings` | Yes | Get Umami analytics settings |
| `POST` | `/api/umami-settings` | Yes | Update Umami settings |

### Media & Content

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/api/uploads` | No | List recent file uploads |
| `POST` | `/api/upload` | Yes | Upload an image file |
| `GET` | `/api/quotes` | No | List all quotes |
| `POST` | `/api/quotes` | Yes | Add a new quote |
| `PUT` | `/api/quotes` | Yes | Update a quote |
| `DELETE` | `/api/quotes` | Yes | Delete a quote |
| `GET` | `/api/quote/random` | No | Get a random quote |

### System

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/api/version` | No | Get application version |
| `GET` | `/ws` | No | WebSocket endpoint for live race updates |
| `GET` | `/docs` | No | Swagger UI documentation |
| `GET` | `/api-docs` | No | OpenAPI spec (JSON) |

> \* Login endpoint requires setup flag for initial admin creation. After setup, standard credentials required.

## 📈 Statistics & Analytics

### Head-to-Head Comparison
Compare two racers across all season races they've both participated in:
```
GET /api/stats/head-to-head?racer1=1&racer2=2
```
Returns total races together, wins per racer, and average finishing positions.

### Points Progression
Track cumulative points growth for any racer across a season:
```
GET /api/stats/points-progression?racer_id=1
```
Returns an ordered array of cumulative points after each race, suitable for line charts.

### Streaks
See which racers have the longest active and all-time streaks:
```
GET /api/stats/streaks
```
Tracks three streak types per racer:
- **wins** - Consecutive 1st place finishes
- **podiums** - Consecutive top-3 finishes
- **dnf** - Consecutive Did Not Finish results

### ELO Ratings
ELO-style skill ratings computed from all pairwise race results:
```
GET /api/stats/elo
```
Uses K=32 with 1500 starting rating. Each race creates pairwise comparisons between every two racers. Ratings update after each race and are sorted descending.

### Export to CSV
Download all racer statistics as a CSV file:
```
GET /api/stats/export
```
Returns a CSV file with columns: ID, Name, Car, Points, Rank, Races, Wins, Podiums, Fastest Laps, DNF.

### Track Performance Breakdown
Analyze performance per track:
```
GET /api/stats/track-performance           # Overview of all tracks
GET /api/stats/track-performance?racer_id=1 # Per-racer track breakdown
```
Without racer_id: returns track summaries with unique driver counts.
With racer_id: returns wins, podiums, races, and average finishing position per track for that racer.

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

The application will be available at [http://localhost:6270](http://localhost:6270)

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
| Port 6270 in use | Change port in `docker-compose.yml` |
| Database locked | Ensure only one instance is running |

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.