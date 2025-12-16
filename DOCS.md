# Korus Music Streaming Server

## Goal
Self-hosted multi-user music streaming server with clean, readable code. Features: authentication, library browsing, search, streaming, playlists, favorites, listening history, detailed stats, and admin tools.

## Tech Stack

### Backend
- **Go** - Simple syntax, single binary deployment, excellent for streaming
- **Echo** - Web framework with built-in validation and middleware
- **SQLite** (modernc.org/sqlite) - Zero-config, pure Go, built-in FTS5
- **golang-jwt/jwt/v5** - JWT authentication
- **golang.org/x/crypto/bcrypt** - Password hashing
- **Audio metadata** - dhowden/tag or taglib via CGO
- **Optional**: MusicBrainz integration for metadata enrichment and ListenBrainz for play submissions

### Frontend (Not a priority yet)
- **Svelte** - Reactive UI components
- **Vite** - Fast dev server and builds
- Served as static files from Go backend

## Architecture

```
/cmd/server/
  main.go

/internal/
  /api/
    router.go
    middleware.go
    /handlers/
      auth.go
      library.go
      player.go
      playlists.go
      favorites.go
      history.go
      admin.go
      health.go
  /models/
    user.go
    music.go
    playlist.go
  /services/
    auth.go
    scanner.go
    transcoder.go
    search.go
  /db/
    schema.sql
    queries.sql
    generated/

/web/
  /src/
    App.svelte
    /routes/
    /components/
    /lib/
      api.js
      auth.js
      player.js
    /stores/
```

## Database Schema

```sql
users (id, username, password_hash, email, role, created_at)
sessions (token, user_id, expires_at)

artists (id, name, bio, image_path, mbid, created_at)
albums (id, artist_id, title, year, cover_path, mbid, created_at)
songs (id, album_id, title, track_number, duration, file_path, lyrics, lyrics_synced, mbid)

playlists (id, user_id, name, description, public, created_at)
playlist_songs (playlist_id, song_id, position)

favorites_songs (user_id, song_id, created_at)
favorites_albums (user_id, album_id, created_at)
follows_artists (user_id, artist_id, created_at)

play_history (id, user_id, song_id, played_at, duration_listened, completion_rate, source)

scan_status (id, status, progress, started_at, completed_at)

-- FTS5 for search
songs_fts (song_id, title, artist_name, album_title)

-- Index
CREATE INDEX idx_play_history_user_time ON play_history(user_id, played_at);
```

## API Design

All routes under `/api/*`, everything else serves Svelte SPA.

### Auth
```
POST /api/auth/register {username, email, password} → {user, token}
POST /api/auth/login {username, password} → {user, token}
POST /api/auth/logout {} → {success}
GET /api/auth/me → {id, username, email, role, created_at}
```

### Library
```
GET /api/library → {artists: [...], albums: [...], songs: [...]}
GET /api/artists/:id → {id, name, bio, image_path, mbid, albums: [...], songs: [...]}
GET /api/albums/:id → {id, title, year, artist: {...}, cover_path, mbid, songs: [...]}
GET /api/songs/:id → {id, title, duration, track_number, file_path, lyrics, lyrics_synced, album: {...}, artist: {...}, mbid}
```

### Search
```
GET /api/search?q=query → {songs: [...], albums: [...], artists: [...]}
```

### Streaming
```
GET /api/stream/:id?format=mp3&bitrate=320  (Range support)
GET /api/stream/:id?format=aac&bitrate=256  (Range support)
GET /api/stream/:id?format=opus&bitrate=128 (Range support)

GET /api/artwork/:id → image/jpeg
GET /api/lyrics/:id → {lyrics, synced: [{time, text}], source}
```

### Playlists
```
GET /api/playlists → [{id, name, description, owner: {...}, song_count, public, created_at}]
POST /api/playlists {name, description, public} → {id, name, ...}
GET /api/playlists/:id → {id, name, description, owner, public, songs: [...], created_at}
PUT /api/playlists/:id {name, description, public} → {id, name, ...}
DELETE /api/playlists/:id → {success}
POST /api/playlists/:id/songs {song_id, position} → {success}
DELETE /api/playlists/:id/songs/:song_id → {success}
PUT /api/playlists/:id/reorder {song_ids: [3,1,2]} → {success}
```

### Favorites
```
POST /api/favorites/songs/:id → {success}
DELETE /api/favorites/songs/:id → {success}
POST /api/favorites/albums/:id → {success}
DELETE /api/favorites/albums/:id → {success}
POST /api/follows/artists/:id → {success}
DELETE /api/follows/artists/:id → {success}
GET /api/favorites → {songs: [...], albums: [...], artists: [...]}
```

### History & Stats
```
POST /api/history {song_id, duration_listened, timestamp} → {success}
GET /api/history?limit=50 → [{song: {...}, played_at, duration_listened}]

GET /api/stats?period=hour|today|week|month|year|all_time → {
  period: {start, end},
  overview: {total_plays, total_time, unique_songs, unique_artists, unique_albums, avg_completion_rate},
  top_songs: [{song, play_count, total_time, avg_completion}],
  top_artists: [{artist, play_count, total_time, unique_songs}],
  top_albums: [{album, play_count, total_time, completion_rate}],
  top_genres: [{genre, play_count, percentage}],
  listening_patterns: {by_hour: [...], by_day: [...], by_month: [...]},
  discovery: {new_artists, new_songs, exploration_rate}
}

GET /api/stats/wrapped?period=2024-12|2024|all_time → {
  period: {type, label},
  summary: {total_plays, total_time, days_listened, avg_plays_per_day, top_season},
  top_songs: [...top 10],
  top_artists: [...top 10],
  top_albums: [...top 10],
  top_genres: [...top 5],
  milestones: [{type, date, value}],
  listening_personality: {type, repeat_rate, discovery_rate, variety_score, binge_listener, favorite_time, obscurity_score},
  journey: [{period, plays, top_song}],
  loyalty: [{artist, consistency_score, listened_every_week}],
  discovery_timeline: [{date, new_artists, new_songs}]
}

GET /api/stats/insights → {
  current_streak: {days, type},
  longest_streak: {days, period},
  trends: [{type, change, comparison}],
  predictions: {likely_month_end_top_artist, pace_to_reach},
  fun_facts: [...]
}

GET /api/stats/social?period=month → {
  your_rank,
  total_users,
  leaderboard: [{user, play_count, total_time}],
  taste_match: [{user, similarity, shared_artists, shared_top_songs}]
}

GET /api/home → {recent_plays, recommended, new_additions, stats_summary}
```

### Library Scanning
```
POST /api/scan → {scan_id, status}
GET /api/scan/status → {scan_id, status, progress, total, current_file}
```

### Admin
```
GET /api/admin/system → {library: {...}, storage: {...}, users: {...}, uptime}
DELETE /api/admin/sessions/cleanup {older_than_days} → {deleted}
```

### Health
```
GET /api/health → {status, timestamp}
```

### MusicBrainz (Optional)
```
POST /api/musicbrainz/submit-listen {song_id} → {success, submitted_to_listenbrainz}
GET /api/musicbrainz/recommendations → {songs: [...]}
POST /api/admin/musicbrainz/enrich {artist_id|album_id|song_id} → {success, mbid}
```

## Key Implementation Notes

- JWT tokens in `Authorization: Bearer <token>` header
- First user is Admin user
- Streaming uses HTTP Range headers for seeking
- Stats computed on-demand with SQL aggregations (fast enough for personal use)
- Single `play_history` table, no pre-computed aggregates
- Optional caching for expensive wrapped queries (yearly/all-time)
- MusicBrainz IDs (mbid) stored as nullable strings
- ListenBrainz submissions async (fire and forget)
- Error responses: `{error: "message", code: "ERROR_CODE"}`

## Streaming details
**Supported Formats:**

| Format | Valid Bitrates (kbps) | Content-Type |
|--------|----------------------|--------------|
| `mp3` | 128, 192, 256, 320 | audio/mpeg |
| `aac` | 128, 192, 256 | audio/mp4 |
| `opus` | 64, 96, 128, 192 | audio/ogg |

**Notes:**
- No params = serve original file (no transcoding)
- `format` is required if transcoding, `bitrate` is optional (uses highest for format)
- Range requests not supported for transcoded streams
- Requires FFmpeg installed on server

**Error Responses:**
- `400` — Invalid format or bitrate
- `503` — FFmpeg not available

## Stats Calculation Logic

```
repeat_rate = plays of previously heard songs / total plays
discovery_rate = plays of new songs / total plays
variety_score = unique_songs / total_plays (normalized)
binge_listener = avg session length > 60 mins
obscurity_score = avg(1 / global_play_count_per_song)
consistency_score = listened in X% of weeks/months in period
```