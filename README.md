# Korus

Self-hosted music streaming server with a web interface.

## Features

- **Library management** - Automatic scanning with file watch support, artist/album/song organization
- **Streaming** - Direct playback for browser-supported formats, on-the-fly transcoding for others
- **Lossless support** - WAV transcoding with seeking for ALAC/FLAC files that browsers can't play natively
- **Playlists** - Create and manage custom playlists
- **Favorites** - Mark songs, albums, and artists as favorites
- **Search** - Full-text search across your library
- **Listening history** - Track what you've played
- **Stats** - Listening statistics with time period filters
- **Wrapped** - Year-in-review style listening summary
- **Lyrics** - Display lyrics when available
- **Queue management** - Reorder, add, remove tracks
- **MusicBrainz integration** - Enrich metadata from MusicBrainz
- **ListenBrainz scrobbling** - Submit listens to ListenBrainz
- **Multi-user** - User accounts with JWT authentication

## Screenshots

| Home | Library |
|------|---------|
| ![Home](docs/screenshots/home.png) | ![Library](docs/screenshots/library.png) |

| Album Details | Search |
|---------------|--------|
| ![Album Details](docs/screenshots/album_details.png) | ![Search](docs/screenshots/search.png) |

| Queue | Lyrics |
|-------|--------|
| ![Queue](docs/screenshots/queue.png) | ![Lyrics](docs/screenshots/lyrics.png) |

| Stats | Settings |
|-------|----------|
| ![Stats](docs/screenshots/stats.png) | ![Settings](docs/screenshots/settings.png) |

## Tech Stack

**Backend**
- Go with Echo framework
- SQLite database (modernc.org/sqlite, no CGO)
- FFmpeg/FFprobe for audio processing and transcoding

**Frontend**
- SvelteKit with Svelte 5
- Tailwind CSS 4
- TypeScript

## Requirements

- Go 1.24+
- Node.js / Bun (for frontend)
- FFmpeg and FFprobe in PATH (or set `FFMPEG_PATH` / `FFPROBE_PATH`)

## Setup

### Backend

```bash
# Set required environment variables
export JWT_SECRET="your-secret-key"
export MEDIA_ROOT="/path/to/your/music"

# Run the server
go run ./cmd/server

# Or build and run
go build -o korus ./cmd/server
./korus
```

### Frontend

```bash
cd web
bun install
bun run dev      # Development
bun run build    # Production build
```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `ADDR` | `:8080` | Server address |
| `DB_PATH` | `./korus.db` | SQLite database path |
| `MEDIA_ROOT` | `./media` | Music library path |
| `JWT_SECRET` | - | Required. Secret for JWT tokens |
| `TOKEN_TTL` | `15m` | Access token lifetime |
| `REFRESH_TTL` | `7d` | Refresh token lifetime |
| `FFMPEG_PATH` | `ffmpeg` | Path to ffmpeg binary |
| `FFPROBE_PATH` | `ffprobe` | Path to ffprobe binary |

### Scanner

| Variable | Default | Description |
|----------|---------|-------------|
| `SCAN_WATCH` | `false` | Watch for file changes |
| `SCAN_EXCLUDE_PATTERN` | - | Regex pattern to exclude files |
| `SCAN_EMBEDDED_COVER` | `true` | Extract embedded cover art |

### Integrations

| Variable | Default | Description |
|----------|---------|-------------|
| `ENABLE_MUSICBRAINZ` | `false` | Enable MusicBrainz metadata enrichment |
| `MUSICBRAINZ_AGENT` | - | User agent for MusicBrainz API |
| `ENABLE_LISTENBRAINZ` | `false` | Enable ListenBrainz scrobbling |
| `LISTENBRAINZ_TOKEN` | - | ListenBrainz API token |
| `LISTENBRAINZ_USER` | - | ListenBrainz username |

### Rate Limiting

| Variable | Default | Description |
|----------|---------|-------------|
| `RATE_LIMIT_AUTH_COUNT` | `5` | Auth attempts allowed |
| `RATE_LIMIT_AUTH_WINDOW` | `1m` | Time window for auth rate limit |

## API

### Auth
- `POST /api/auth/register` - Create account
- `POST /api/auth/login` - Login
- `POST /api/auth/refresh` - Refresh token
- `POST /api/auth/logout` - Logout
- `GET /api/auth/me` - Current user

### Library
- `GET /api/library` - Library overview
- `GET /api/artists/:id` - Artist details
- `GET /api/albums/:id` - Album details
- `GET /api/songs/:id` - Song details
- `GET /api/search?q=` - Search

### Streaming
- `GET /api/stream/:id` - Stream audio (optional `?format=&bitrate=`)
- `GET /api/artwork/:id` - Album/song artwork
- `GET /api/lyrics/:id` - Song lyrics

### Playlists
- `GET /api/playlists` - List playlists
- `POST /api/playlists` - Create playlist
- `GET /api/playlists/:id` - Playlist details
- `PUT /api/playlists/:id` - Update playlist
- `DELETE /api/playlists/:id` - Delete playlist

### User Data
- `GET /api/favorites` - List favorites
- `POST /api/favorites/:type/:id` - Add favorite
- `DELETE /api/favorites/:type/:id` - Remove favorite
- `GET /api/history` - Listening history
- `POST /api/history` - Record listen
- `GET /api/stats` - Listening statistics
- `GET /api/home` - Home page data

### Library Scanning
- `POST /api/scan` - Trigger library scan
- `GET /api/scan/status` - Scan status

### Admin
- `GET /api/admin/system` - System info
- `POST /api/admin/sessions/cleanup` - Clean expired sessions
- `POST /api/admin/musicbrainz/enrich` - Enrich metadata

## Tests

```bash
# Backend
go test ./...

# Frontend
cd web && bun run check
```
