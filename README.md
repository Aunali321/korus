## Korus Backend

Go + Echo self-hosted music server (backend only).

### Requirements
- Go 1.24
- SQLite (modernc.org/sqlite)
- ffmpeg and ffprobe in PATH (or set `FFMPEG_PATH` / `FFPROBE_PATH`)

### Config (env)
- `ADDR` (default `:8080`)
- `DB_PATH` (default `./korus.db`)
- `MEDIA_ROOT` (default `./media`)
- `JWT_SECRET` (required)
- `TOKEN_TTL`, `REFRESH_TTL`
- `FFMPEG_PATH`, `FFPROBE_PATH`
- `RATE_LIMIT_AUTH_COUNT`, `RATE_LIMIT_AUTH_WINDOW`
- Scanner: `SCAN_WATCH`, `SCAN_EXCLUDE_PATTERN`, `SCAN_EMBEDDED_COVER`
- Integrations: `ENABLE_MUSICBRAINZ`, `MUSICBRAINZ_AGENT`, `ENABLE_LISTENBRAINZ`, `LISTENBRAINZ_TOKEN`, `LISTENBRAINZ_USER`

### Run
```bash
go run ./cmd/server
```

### Tests
```bash
go test ./...
```

### Key Endpoints
- Auth: `/api/auth/register`, `/api/auth/login`, `/api/auth/refresh`, `/api/auth/logout`, `/api/auth/me`
- Library/Search: `/api/library`, `/api/search`, `/api/artists/:id`, `/api/albums/:id`, `/api/songs/:id`
- Streaming: `/api/stream/:id` (optional `format`, `bitrate`), `/api/artwork/:id`, `/api/lyrics/:id`
- Playlists/Favorites/History/Stats: `/api/playlists*`, `/api/favorites*`, `/api/history`, `/api/stats`, `/api/home`
- Admin: `/api/admin/scan`, `/api/admin/scan/status`, `/api/admin/system`, `/api/admin/sessions/cleanup`, `/api/admin/musicbrainz/enrich`

