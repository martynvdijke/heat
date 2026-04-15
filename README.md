# Heat - Racing Application

A Go-based racing application with a web interface for managing racers, points, and race information. Built with a SQLite backend and served through a RESTful API.

## Project Overview

Heat is a lightweight racing management system that tracks:
- Racer profiles and statistics
- Race information and results
- Points and rankings
- Fastest lap times

The application consists of:
- **Backend**: Go server with REST API
- **Database**: SQLite for data persistence
- **Frontend**: HTML/CSS web interface
- **Infrastructure**: Docker support for easy deployment

## Prerequisites

### For Local Development
- **Go** 1.21 or higher
- **SQLite3** development libraries
- **GCC** compiler (for CGO)

### For Docker
- **Docker** 20.10+
- **Docker Compose** 1.29+

## Building Locally

### 1. Install Dependencies

Ensure Go dependencies are available:
```bash
go mod download
go mod tidy
```

### 2. Build the Application

Build the executable with SQLite support (requires CGO):
```bash
CGO_ENABLED=1 go build -o heat-server .
```

On different platforms:
- **Linux/macOS**: The command above works directly
- **Windows**: Requires MinGW or similar C compiler setup

The build will create a `heat-server` binary in the current directory.

## Running Locally

### 1. Run the Server

```bash
./heat-server
```

The server will:
- Initialize or connect to `heat.db` (SQLite database)
- Start listening on `http://localhost:8080`
- Serve static files (HTML/CSS) from the `static/` directory

### 2. Access the Application

Open your browser and navigate to:
```
http://localhost:8080
```

### 3. API Endpoints

The application exposes REST endpoints:
- `GET /api/racers` - Get all racers
- `POST /api/racers` - Update a racer
- `DELETE /api/racers` - Delete a racer
- `GET /` - Serve the web interface

## Building with Docker

Build the Docker image:
```bash
docker build -t heat-server .
```

The Docker build uses a multi-stage approach:
1. **Builder stage**: Includes Go compiler and build dependencies
2. **Runtime stage**: Minimal Alpine image with only runtime requirements

## Running with Docker Compose

### 1. Start the Application

```bash
docker-compose up -d
```

This will:
- Build the image (if not already built)
- Start the container named `heat_pedal_to_the_metal`
- Expose the application on `http://localhost:8080`
- Create a persistent volume `heat-data` for database storage

### 2. Access the Application

Open your browser:
```
http://localhost:8080
```

### 3. Stop the Application

```bash
docker-compose down
```

To remove the persistent data volume as well:
```bash
docker-compose down -v
```

### 4. View Logs

```bash
docker-compose logs -f heat-server
```

## Project Structure

```
.
├── main.go               # Application entry point and API handlers
├── go.mod               # Go module definition
├── go.sum              # Go dependencies checksums
├── heat.db             # SQLite database (created at runtime)
├── Dockerfile          # Multi-stage Docker build configuration
├── docker-compose.yml  # Docker Compose orchestration
├── static/
│   ├── index.html      # Main web interface
│   ├── admin.html      # Admin dashboard
│   └── style.css       # Styling
└── README.md           # This file
```

## Database

The application uses SQLite with the database file `heat.db` created automatically on first run. The database is initialized by the `initDB()` function in `main.go`.

### Data Persistence

- **Local runs**: Database is stored as `heat.db` in the working directory
- **Docker runs**: Database is stored in the `heat-data` volume for persistence across container restarts

## Troubleshooting

### Build Issues

**Error: "no C compiler found"**
- Install build essentials: `sudo apt-get install build-essential` (Linux) or XCode Command Line Tools (macOS)

**Error: "sqlite3.h not found"**
- Install SQLite dev packages: `sudo apt-get install libsqlite3-dev` (Linux)

### Runtime Issues

**Port 8080 already in use**
- Change the port in `docker-compose.yml` or use a different port in local runs

**Database locked errors**
- Ensure only one instance of the application is running
- Check for orphaned processes: `ps aux | grep heat-server`

## Development

To rebuild and test changes:

```bash
# Stop current container (if running)
docker-compose down

# Rebuild the image
docker-compose build

# Start the application
docker-compose up -d
```

## License

See [LICENSE](LICENSE) for details.