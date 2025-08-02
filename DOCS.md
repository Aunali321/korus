# Korus Music Server Documentation

Welcome to the Korus documentation! This page provides links to all available documentation for the Korus self-hosted music streaming server.

## 📚 Documentation Index

### 🚀 Getting Started
- **[README.md](./README.md)** - Quick start guide and basic setup instructions
- **[DESIGN.md](./DESIGN.md)** - System architecture and design specifications
- **[PROGRESS.md](./PROGRESS.md)** - Development progress and implementation status

### 🔧 Development
- **[API.md](./API.md)** - Complete REST API documentation with examples
- **[Docker Setup](#docker-setup)** - Container deployment guide
- **[Configuration](#configuration)** - Environment variables and settings

### 📋 Reference
- **[Supported Formats](#supported-audio-formats)** - Audio format compatibility
- **[Database Schema](#database-schema)** - PostgreSQL table structure
- **[Project Structure](#project-structure)** - Codebase organization

---

## 🐳 Docker Setup

### Quick Start
```bash
# Clone repository
git clone <repository-url>
cd korus

# Configure environment
cp .env.example .env
# Edit .env with your settings

# Generate secrets
export POSTGRES_PASSWORD=$(openssl rand -base64 32)
export JWT_SECRET=$(openssl rand -base64 32)

# Set music library path
export MUSIC_PATH=/path/to/your/music

# Start services
docker-compose up -d

# Check logs for admin credentials
docker-compose logs korus
```

### Docker Compose Services
- **PostgreSQL** - Database with persistent storage
- **Korus Server** - Main application with health checks
- **Volumes** - Persistent data and cache storage

### Container Health Checks
Both services include comprehensive health checks:
- **PostgreSQL**: `pg_isready` connection test
- **Korus**: HTTP ping endpoint verification

---

## ⚙️ Configuration

### Required Environment Variables
| Variable | Description | Example |
|----------|-------------|---------|
| `POSTGRES_PASSWORD` | Database password | Generated with `openssl rand -base64 32` |
| `JWT_SECRET` | JWT signing secret | Generated with `openssl rand -base64 32` |
| `MUSIC_PATH` | Host path to music library | `/home/user/Music` |

### Optional Environment Variables
| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3000` | HTTP server port |
| `ENVIRONMENT` | `production` | Runtime environment |
| `ADMIN_USERNAME` | `admin` | Initial admin username |
| `ADMIN_PASSWORD` | Generated | Initial admin password |
| `DB_MAX_CONNS` | `20` | Max database connections |
| `SCAN_WORKERS` | `4` | File scanner worker count |

### Configuration File Location
```
/app/config/          # Container config directory
./config/             # Host config directory (optional)
```

---

## 🎵 Supported Audio Formats

Korus supports the following audio formats with metadata extraction:

| Format | Extension | Codec | Metadata Support |
|--------|-----------|-------|------------------|
| **MP3** | `.mp3` | MPEG-1 Audio Layer III | ✅ Full (ID3v1/ID3v2) |
| **FLAC** | `.flac` | Free Lossless Audio Codec | ✅ Full (Vorbis Comments) |
| **M4A** | `.m4a` | Advanced Audio Coding | ✅ Full (iTunes-style) |
| **AAC** | `.aac` | Advanced Audio Coding | ✅ Basic |
| **OGG** | `.ogg` | Ogg Vorbis | ✅ Full (Vorbis Comments) |
| **WAV** | `.wav` | Waveform Audio | ⚠️  Limited metadata |

### Metadata Fields Extracted
- **Basic**: Title, Artist, Album, Year, Track Number
- **Extended**: Album Artist, Disc Number, Duration, Bitrate
- **Advanced**: MusicBrainz IDs, Genre, Cover Art

---

## 🗄️ Database Schema

### Core Tables
- **`users`** - User accounts and authentication
- **`artists`** - Music artists with MusicBrainz integration
- **`albums`** - Album information and artwork paths
- **`songs`** - Track metadata and file locations

### User Data Tables
- **`playlists`** - User-created playlists
- **`playlist_songs`** - Playlist track associations
- **`liked_songs`** - User music preferences
- **`play_history`** - Listening activity tracking

### System Tables
- **`job_queue`** - Background task processing
- **`user_sessions`** - JWT refresh token storage
- **`scan_history`** - Library scan audit log
- **`schema_migrations`** - Database version control

For complete schema definition, see [migrations/001_initial_schema.sql](./migrations/001_initial_schema.sql).

---

## 📁 Project Structure

```
korus/
├── cmd/korus/                   # Application entry point
│   └── main.go                  # Server initialization
├── internal/                    # Private application code
│   ├── auth/                    # JWT authentication system
│   │   ├── jwt.go              # Token generation/validation
│   │   ├── password.go         # Password hashing utilities
│   │   └── service.go          # User management service
│   ├── config/                  # Configuration management
│   │   └── config.go           # Environment variable handling
│   ├── database/                # Database connectivity
│   │   └── database.go         # PostgreSQL connection pool
│   ├── handlers/                # HTTP request handlers
│   │   ├── auth.go             # Authentication endpoints
│   │   ├── health.go           # Health check endpoints
│   │   └── library.go          # Music library endpoints
│   ├── jobs/                    # Background job processing
│   │   ├── queue.go            # PostgreSQL-based job queue
│   │   ├── worker.go           # Job worker implementation
│   │   ├── handlers.go         # Job type handlers
│   │   └── scanner_adapter.go  # Scanner interface adapter
│   ├── middleware/              # HTTP middleware components
│   │   ├── auth.go             # Authentication middleware
│   │   ├── cors.go             # CORS policy handling
│   │   ├── logging.go          # Request logging
│   │   └── rate_limit.go       # Rate limiting
│   ├── models/                  # Data structure definitions
│   │   └── user.go             # Domain models
│   ├── scanner/                 # File system monitoring
│   │   └── scanner.go          # fsnotify-based file watcher
│   ├── search/                  # Full-text search engine
│   │   └── search.go           # Bleve search integration
│   ├── services/                # Business logic layer
│   │   ├── library.go          # Music library service
│   │   └── metadata.go         # Audio metadata extraction
│   └── streaming/               # Audio streaming engine
│       └── stream.go           # HTTP range request handler
├── migrations/                  # Database migrations
│   ├── 001_initial_schema.sql  # Initial database schema
│   └── migration.go            # Migration runner
├── static/                      # Static file serving (optional)
├── docker-compose.yml           # Docker services configuration
├── Dockerfile                   # Container image definition
├── .dockerignore               # Docker build exclusions
├── .env.example                # Environment template
├── go.mod                      # Go module dependencies
├── go.sum                      # Dependency checksums
├── README.md                   # Project overview
├── DESIGN.md                   # Architecture documentation
├── PROGRESS.md                 # Development progress
├── API.md                      # REST API documentation
└── DOCS.md                     # This documentation index
```

### Key Architecture Components

1. **Modular Design** - Clean separation between layers
2. **Dependency Injection** - Services injected through constructors
3. **Interface-Based** - Testable components with clear contracts
4. **Background Processing** - PostgreSQL-based job queue
5. **Real-time Updates** - File system watching with debouncing
6. **Caching Strategy** - Multi-level caching for performance

---

## 🔗 External Dependencies

### Go Modules
- **`gin`** - HTTP web framework
- **`pgx/v5`** - PostgreSQL driver and connection pooling
- **`bleve/v2`** - Full-text search engine
- **`jwt/v5`** - JSON Web Token implementation
- **`fsnotify`** - File system event notifications
- **`tag`** - Audio metadata extraction
- **`bcrypt`** - Password hashing

### System Dependencies
- **PostgreSQL 15+** - Primary database
- **Docker & Docker Compose** - Container orchestration
- **Linux/macOS/Windows** - Cross-platform support

---

## 🛠️ Development Workflow

### Local Development Setup
```bash
# Install Go 1.24+
go version

# Clone repository
git clone <repository-url>
cd korus

# Install dependencies
go mod download

# Set up local database
docker run -d --name korus-postgres \
  -e POSTGRES_DB=korus \
  -e POSTGRES_USER=korus \
  -e POSTGRES_PASSWORD=password \
  -p 5432:5432 postgres:15-alpine

# Configure environment
export DATABASE_URL="postgres://korus:password@localhost/korus?sslmode=disable"
export JWT_SECRET="development-secret-key"
export MUSIC_DIR="/path/to/test/music"

# Build and run
go build -o korus ./cmd/korus
./korus
```

### Testing
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/auth/
```

### Build Process
```bash
# Local build
go build -o korus ./cmd/korus

# Docker build
docker build -t korus .

# Cross-platform build
GOOS=linux GOARCH=amd64 go build -o korus-linux ./cmd/korus
```

---

## 📞 Support & Contributing

### Getting Help
- **Issues**: Report bugs and feature requests
- **Discussions**: Community support and questions
- **Documentation**: Check existing docs first
- **Logs**: Include relevant log output with issues

### Contributing Guidelines
1. **Fork** the repository
2. **Create** feature branch (`git checkout -b feature/amazing-feature`)
3. **Commit** changes (`git commit -m 'Add amazing feature'`)
4. **Push** to branch (`git push origin feature/amazing-feature`)
5. **Open** Pull Request

### Development Standards
- **Go Formatting**: Use `go fmt` and `go vet`
- **Testing**: Include tests for new features
- **Documentation**: Update docs for API changes
- **Commit Messages**: Use conventional commit format

---

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details.

---

*Last updated: August 1, 2025*