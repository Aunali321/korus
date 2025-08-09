# Korus Music Server

A high-performance, self-hosted music streaming server built with Go, PostgreSQL, and modern web technologies.

## Features

- **RESTful API** with JWT authentication
- **Real-time file scanning** with automatic metadata extraction
- **Comprehensive lyrics support** with embedded, LRC, and TXT file extraction
- **Custom LRC parser** for synchronized lyrics with precise timing
- **Intelligent language detection** using lingua-go for automatic language identification
- **Comprehensive cover art extraction** from embedded metadata and external files
- **WebP, JPEG, PNG, GIF format support** for cover images
- **Full-text search** powered by Bleve
- **Audio streaming** with range request support
- **PostgreSQL-based job queue** for background processing
- **Docker support** for easy deployment
- **User management** with role-based access control

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Music files in a directory on your host system

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd korus
```

2. Copy the environment file and configure it:
```bash
cp .env.example .env
# Edit .env with your configuration
```

3. Set required environment variables:
```bash
# Generate secure passwords/secrets
export POSTGRES_PASSWORD=$(openssl rand -base64 32)
export JWT_SECRET=$(openssl rand -base64 32)

# Set your music library path
export MUSIC_PATH=/path/to/your/music/library
```

4. Start the services:
```bash
docker-compose up -d
```

5. Check the logs for the admin credentials:
```bash
docker-compose logs korus
```

The server will be available at `http://localhost:3000`.

## API Documentation

### Authentication

- `POST /api/auth/login` - Authenticate user
- `POST /api/auth/register` - Register new user (if enabled)
- `POST /api/auth/refresh` - Refresh access token
- `POST /api/auth/logout` - Logout user
- `GET /api/me` - Get current user info

### Library

- `GET /api/library/stats` - Get library statistics
- `GET /api/artists` - List artists
- `GET /api/artists/{id}` - Get artist details
- `GET /api/albums` - List albums
- `GET /api/albums/{id}` - Get album details
- `GET /api/albums/{id}/songs` - Get album songs
- `GET /api/songs` - Batch fetch songs
- `GET /api/songs/{id}` - Get song details
- `GET /api/search` - Search library

### Streaming

- `GET /api/songs/{id}/stream` - Stream audio file

### Cover Art

Cover art is automatically extracted and included in API responses as `cover_path` URLs. Images are served as static files:
- Cover URLs: `/covers/{hash}.{ext}`
- Supported formats: JPEG, PNG, WebP, GIF
- Sources: Song-specific files, embedded metadata, album folder covers

### Health

- `GET /api/ping` - Basic health check
- `GET /api/health` - Detailed health status

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | Required |
| `JWT_SECRET` | JWT signing secret | Required |
| `ADMIN_USERNAME` | Initial admin username | `admin` |
| `ADMIN_PASSWORD` | Initial admin password | Generated |
| `MUSIC_DIR` | Music library directory | `./music` |
| `CACHE_DIR` | Cache directory | `./cache` |
| `SERVER_PORT` | HTTP server port | `3000` |
| `ENVIRONMENT` | Environment mode | `development` |

### Supported Formats

**Audio Files:**
- MP3 (`.mp3`)
- FLAC (`.flac`) 
- M4A/AAC (`.m4a`, `.aac`)
- OGG Vorbis (`.ogg`)
- WAV (`.wav`)

**Cover Art:**
- JPEG (`.jpg`, `.jpeg`)
- PNG (`.png`)
- WebP (`.webp`)
- GIF (`.gif`)

## Development

### Building from Source

1. Install Go 1.24 or later
2. Clone the repository
3. Install dependencies:
   ```bash
   go mod download
   ```
4. Build the application:
   ```bash
   go build -o output/korus ./cmd/korus
   ```

### Running Tests

```bash
go test ./...
```

### Database Migrations

Migrations are automatically applied on startup. To check migration status:

```bash
./korus migrate status
```

## Architecture

Korus is built with a modular architecture:

- **API Layer**: Gin-based REST API with middleware
- **Service Layer**: Business logic and data processing
- **Data Layer**: PostgreSQL with pgx driver
- **Job System**: PostgreSQL-based queue with worker pools
- **Search**: Bleve full-text search index
- **Streaming**: HTTP range request support for audio
- **Cover Art**: Automatic extraction and static file serving

## License

This project is licensed under the MIT License - see the LICENSE file for details.
