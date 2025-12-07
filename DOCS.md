# Korus - Self-Hosted Music Server

## Architecture

```
┌──────────────┐   ┌──────────────┐   ┌─────────────────────┐
│  Filesystem  │──▶│   Indexer    │──▶│     PostgreSQL      │
│   Scanner    │   │  (Metadata)  │   │  (Data + Sessions)  │
└──────────────┘   └──────────────┘   └──────────┬──────────┘
                                                 │
       ┌──────────────────┐                      ▼
       │  Bleve Search    │◀────────────────────────────────┐
       └──────────────────┘                                 │
                │                                           │
                ▼                                           │
┌───────────────────────────────────────────────────────────┴───┐
│                      REST API (Gin)                           │
│   JWT Auth │ Rate Limiting │ CORS │ Request Routing           │
└───────────────────────────────────────────────────────────────┘
                │
                ▼
┌───────────────────────────────────────────────────────────────┐
│            Audio Streaming + Static Cover Serving             │
└───────────────────────────────────────────────────────────────┘
```

**Stack:** Go, PostgreSQL, Bleve, Gin, JWT, fsnotify

---

## Quick Start

```bash
# Generate secrets
export POSTGRES_PASSWORD=$(openssl rand -base64 32)
export JWT_SECRET=$(openssl rand -base64 32)
export MUSIC_DIR=/path/to/your/music

# Start
docker-compose up -d

# Get admin credentials
docker-compose logs korus
```

---

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_PASSWORD` | Required | Database password |
| `JWT_SECRET` | Required | Token signing key |
| `MUSIC_DIR` | Required | Music library path |
| `SERVER_PORT` | `3000` | HTTP port |
| `SCAN_WORKERS` | `4` | Parallel scan workers |
| `INGEST_WORKERS` | `4` | Metadata extraction workers |
| `INGEST_BATCH_SIZE` | `300` | Files per batch |
| `EXTRACT_LYRICS` | `true` | Parse lyrics from files |

---

## Supported Formats

| Format | Metadata |
|--------|----------|
| MP3 | Full (ID3v1/v2) |
| FLAC | Full (Vorbis) |
| M4A/AAC | Full (iTunes) |
| OGG | Full (Vorbis) |
| WAV | Limited |
| OPUS | Full |

**Extracted:** Title, Artist, Album, Year, Track, Disc, Duration, Bitrate, Cover Art, Lyrics (embedded + .lrc + .txt)

---

## Database Tables

**Core:** `users`, `artists`, `albums`, `songs`, `lyrics`  
**User:** `playlists`, `playlist_songs`, `liked_songs`, `liked_albums`, `followed_artists`, `play_history`  
**System:** `user_sessions`, `scan_history`, `schema_migrations`

See [migrations/001_initial_schema.sql](./migrations/001_initial_schema.sql)

---

## Project Structure

```
korus/
├── cmd/korus/main.go       # Entry point
├── internal/
│   ├── auth/               # JWT + password
│   ├── config/             # Environment
│   ├── database/           # PostgreSQL pool
│   ├── handlers/           # HTTP handlers
│   ├── indexer/            # Async scan + job tracking
│   ├── middleware/         # Auth, CORS, rate limit
│   ├── models/             # Domain types
│   ├── search/             # Bleve integration
│   ├── services/           # Business logic
│   └── streaming/          # Range requests
├── migrations/             # SQL schemas
├── docker-compose.yml
└── Dockerfile
```

---

## Development

```bash
# Local setup
docker run -d --name korus-db \
  -e POSTGRES_DB=korus \
  -e POSTGRES_USER=korus \
  -e POSTGRES_PASSWORD=dev \
  -p 5432:5432 postgres:15-alpine

export DATABASE_URL="postgres://korus:dev@localhost/korus?sslmode=disable"
export JWT_SECRET="dev-secret"
export MUSIC_DIR="./test-music"

go build -o korus ./cmd/korus && ./korus
```

```bash
# Tests
go test ./...

# Build
go build -o korus ./cmd/korus
docker build -t korus .
```

---

## Dependencies

**Go:** gin, pgx/v5, bleve/v2, jwt/v5, fsnotify, dhowden/tag, bcrypt, google/uuid, lingua-go  
**System:** PostgreSQL 15+, Docker

---

## API Reference

See [API.md](./API.md) for complete endpoint documentation.
