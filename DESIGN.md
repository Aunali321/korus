# Project Design Document: Korus

## 1. System Architecture

### Backend Technology Stack
- **Language**: Go (Golang) for its exceptional performance, native concurrency support, and minimal memory footprint
- **Web Framework**: Gin (or Fiber) for lightweight, high-performance HTTP routing with middleware support
- **Database Driver**: pgx for robust PostgreSQL connectivity with connection pooling
- **Search Engine**: Bleve for embedded full-text search capabilities without external dependencies
- **File Watching**: fsnotify for real-time library change detection
- **Job Queue**: PostgreSQL-based Job Queue for transactional, dependency-free background task processing using `LISTEN/NOTIFY`

### Core Components Diagram
```
┌─────────────────┐    ┌──────────────────┐    ┌───────────────────────────────────┐
│   File System   │    │   Metadata       │    │      PostgreSQL Database          │
│     Scanner     │───▶│    Extractor     │───▶│   (Core Data & Job Queue)       │
└─────────────────┘    └──────────────────┘    └─────────┬────────────▲──────────┘
         │                       │                      │            │
         │                       ▼                      │      ┌─────┴──────┐
         ▼               ┌──────────────────┐           │      │ Job Workers  │
┌─────────────────┐      │   Bleve Search   │           │      │ (LISTEN)   │
│   File Watching │      │     Index        │           │      └────────────┘
│    (fsnotify)   │      └──────────────────┘           │
         │                       │                      ▼
         └───────────────────────┼───────────┌─────────────────┐
                                 │           │   User-Scoped   │
                                 │           │     Logic       │
                                 │           └─────────────────┘
                                 ▼
┌─────────────────────────────────────────────────────────────┐
│                   REST API Layer (Gin)                       │
│  ┌───────────────┐ ┌───────────────┐ ┌───────────────────┐   │
│  │ Authentication│ │ Authorization│ │   Request Router  │   │
│  │   (JWT)       │ │   (RBAC)      │ │                   │   │
│  └───────────────┘ └───────────────┘ └───────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────┐
│                 Audio Streaming Engine                       │
│  ┌───────────────┐ ┌───────────────┐ ┌───────────────────┐   │
│  │ Range Request │ │ Transcoder    │ │   Static Files    │   │
│  │   Handler     │ │   (Future)    │ │     (Covers)      │   │
│  └───────────────┘ └───────────────┘ └───────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────┐
│                  Cover Art Extraction System                │
│  ┌───────────────┐ ┌───────────────┐ ┌───────────────────┐   │
│  │ Embedded Art  │ │ External Files│ │ Song-Specific     │   │
│  │   Extractor   │ │   Scanner     │ │   Cover Scanner   │   │
│  └───────────────┘ └───────────────┘ └───────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### Component Interactions
1. **File Scanner** detects changes → enqueues metadata extraction jobs in the PostgreSQL job queue.
2. **Metadata Extractor** (as a Job Worker) processes files from the queue → updates PostgreSQL data and the Bleve index.
3. **REST API** handles client requests → authenticates via JWT → authorizes via RBAC.
4. **User-Scoped Logic** enforces data isolation → interacts with the database.
5. **Streaming Engine** serves audio/artwork → supports range requests.
6. **Job Queue (PostgreSQL-native)** processes async tasks (scans, transcoding, cleanup) with full transactional integrity. Workers listen for new jobs via `NOTIFY`.
7. **Cover Art System** extracts artwork during metadata processing → stores as static files → includes URLs in API responses.

## 2. The "Korus" API Design (Stateless and RESTful)

### Authentication Flow
```mermaid
sequenceDiagram
    participant Client
    participant Korus
    participant Database

    Client->>Korus: POST /api/auth/login {username, password}
    Korus->>Database: Verify credentials
    Database-->>Korus: User record
    Korus->>Korus: Generate JWT pair (access + refresh)
    Korus->>Database: Store refresh token
    Korus-->>Client: {accessToken, refreshToken, expiresAt}

    Client->>Korus: GET /api/me (with accessToken)
    Korus->>Korus: Validate JWT signature
    Korus-->>Client: User profile

    Client->>Korus: POST /api/auth/refresh {refreshToken}
    Korus->>Database: Verify refresh token
    Database-->>Korus: Token record
    Korus->>Korus: Generate new access token
    Korus-->>Client: {accessToken, expiresAt}
```

### API Endpoints Specification

#### Authentication Endpoints
| Method | Endpoint | Description | Request Body | Response |
|--------|----------|-------------|--------------|----------|
| POST | `/api/auth/register` | Create initial user (if enabled post-setup) | `{username, password, email}` | `201 Created` |
| POST | `/api/auth/login` | Authenticate user | `{username, password}` | `{accessToken, refreshToken, expiresAt}` |
| POST | `/api/auth/refresh` | Refresh access token | `{refreshToken}` | `{accessToken, expiresAt}` |
| POST | `/api/auth/logout` | Invalidate session | `{refreshToken}` | `204 No Content` |

#### Core Library Endpoints
| Method | Endpoint | Description | Parameters | Response |
|--------|----------|-------------|------------|----------|
| GET | `/api/ping` | Health check | None | `{status: "ok"}` |
| POST | `/api/library/scan` | Trigger scan (admin) | None | `{jobId: "uuid"}` |
| GET | `/api/library/stats` | Library statistics | None | `{totalSongs, totalArtists, totalAlbums, totalDuration}` |
| GET | `/api/artists` | List artists | `limit, offset, sort` | `[{id, name, albumCount, songCount}]` |
| GET | `/api/artists/{id}` | Artist details | None | `{id, name, albums, topTracks}` |
| GET | `/api/albums` | List albums | `limit, offset, sort, year` | `[{id, name, artist, year, cover_path}]` |
| GET | `/api/albums/{id}` | Album details | None | `{id, name, artist, songs, cover_path}` |
| GET | `/api/songs/{id}` | Song details | None | `{id, title, album, artist, duration, cover_path, url}` |
| GET | `/api/songs` | Batch fetch songs | `ids` | `[{id, title, album, artist, duration, cover_path}]` |
| GET | `/api/songs/{id}/stream` | Stream audio | Range header | Audio stream |
| GET | `/api/search` | Search library | `q, type, limit, offset` | `{songs, albums, artists}` |

## 3. Feature-Driven API Endpoints (User-Scoped)

### Playlist Management
| Method | Endpoint | Description | Request Body | Response |
|--------|----------|-------------|--------------|----------|
| GET | `/api/playlists` | User's playlists | None | `[{id, name, visibility, songCount}]` |
| POST | `/api/playlists` | Create playlist | `{name, description, visibility}` | `201 Created` |
| GET | `/api/playlists/{id}` | Playlist details | None | `{id, name, songs, owner}` |
| PUT | `/api/playlists/{id}` | Update playlist | `{name, description, visibility}` | `200 OK` |
| DELETE | `/api/playlists/{id}` | Delete playlist | None | `204 No Content` |
| POST | `/api/playlists/{id}/songs` | Add songs | `{songIds, position}` | `201 Created` |
| PUT | `/api/playlists/{id}/songs/reorder` | Bulk reorder all songs in a playlist | `{"songIds": ["id3", "id1", "id2"]}` | `200 OK` |
| DELETE | `/api/playlists/{id}/songs` | Remove songs | `{songIds}` or `{positions}` | `204 No Content` |

### Library Management
| Method | Endpoint | Description | Parameters | Response |
|--------|----------|-------------|------------|----------|
| GET | `/api/me/library/songs` | Liked songs | `limit, offset, sort` | `[{id, title, artist, album}]` |
| GET | `/api/me/library/albums` | Liked albums | None | `[{id, name, artist, year}]` |
| GET | `/api/me/library/artists` | Followed artists | None | `[{id, name, albumCount}]` |
| POST | `/api/songs/like` | Batch like songs | `{songIds}` | `204 No Content` |
| DELETE | `/api/songs/like` | Batch unlike songs | `{songIds}` | `204 No Content` |
| POST | `/api/albums/{id}/like` | Like album | None | `204 No Content` |
| DELETE | `/api/albums/{id}/like` | Unlike album | `204 No Content` |
| POST | `/api/artists/{id}/follow` | Follow artist | None | `204 No Content` |
| DELETE | `/api/artists/{id}/follow` | Unfollow artist | None | `204 No Content` |

### Listening Activity
| Method | Endpoint | Description | Request Body | Response |
|--------|----------|-------------|--------------|----------|
| POST | `/api/me/history/scrobble` | Record play | `{songId, playedAt, duration}` | `201 Created` |
| GET | `/api/me/history/recent` | Recent history | `limit` | `[{song, playedAt, duration}]` |
| GET | `/api/me/stats` | User statistics | None | `{totalPlays, mostPlayedArtist, totalTimeListened}` |

### Discovery
| Method | Endpoint | Description | Parameters | Response |
|--------|----------|-------------|------------|----------|
| GET | `/api/me/home` | Personalized home | `sections, limit` | `{recentlyAdded, recentlyPlayed, mostPlayed, recommendedAlbums}` |
| GET | `/api/discover/new` | New releases | `limit` | `[{id, name, artist, dateAdded}]` |

## 4. Database Schema (PostgreSQL)

### Core Tables
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login TIMESTAMPTZ
);

CREATE TABLE artists (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    sort_name VARCHAR(255),
    musicbrainz_id VARCHAR(255) UNIQUE
);

CREATE TABLE albums (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    artist_id INTEGER REFERENCES artists(id),
    album_artist_id INTEGER REFERENCES artists(id),
    year INTEGER,
    musicbrainz_id VARCHAR(255) UNIQUE,
    cover_path VARCHAR(255),
    date_added TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE songs (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    album_id INTEGER REFERENCES albums(id) ON DELETE CASCADE,
    artist_id INTEGER REFERENCES artists(id),
    track_number INTEGER,
    disc_number INTEGER DEFAULT 1,
    duration INTEGER NOT NULL,
    file_path VARCHAR(1024) NOT NULL UNIQUE,
    file_size BIGINT NOT NULL,
    file_modified TIMESTAMPTZ NOT NULL,
    bitrate INTEGER,
    format VARCHAR(10),
    cover_path VARCHAR(255),
    date_added TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### User-Specific Tables
```sql
CREATE TABLE playlists (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    visibility VARCHAR(10) NOT NULL DEFAULT 'private',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE playlist_songs (
    id SERIAL PRIMARY KEY,
    playlist_id INTEGER NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
    song_id INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE liked_songs (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    song_id INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    liked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, song_id)
);

CREATE TABLE liked_albums (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    album_id INTEGER NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
    liked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, album_id)
);

CREATE TABLE followed_artists (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    artist_id INTEGER NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    followed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, artist_id)
);

CREATE TABLE play_history (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    song_id INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    played_at TIMESTAMPTZ NOT NULL,
    play_duration INTEGER,
    ip_address INET
);
```

### System Tables
```sql
CREATE TABLE scan_history (
    id SERIAL PRIMARY KEY,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    songs_added INTEGER NOT NULL DEFAULT 0,
    songs_updated INTEGER NOT NULL DEFAULT 0,
    songs_removed INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE user_sessions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE job_queue (
    id SERIAL PRIMARY KEY,
    job_type VARCHAR(50) NOT NULL,
    payload JSONB,
    status VARCHAR(20) DEFAULT 'pending', -- pending/processing/failed/complete
    created_at TIMESTAMPTZ DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    attempts INTEGER DEFAULT 0,
    last_error TEXT
);
```

### Critical Indexes
```sql
CREATE INDEX idx_songs_album_id ON songs(album_id);
CREATE INDEX idx_songs_artist_id ON songs(artist_id);
CREATE INDEX idx_songs_file_path ON songs(file_path);
CREATE INDEX idx_liked_songs_user_id ON liked_songs(user_id);
CREATE INDEX idx_liked_songs_liked_at ON liked_songs(liked_at);
CREATE INDEX idx_play_history_user_id ON play_history(user_id);
CREATE INDEX idx_play_history_played_at ON play_history(played_at);
CREATE INDEX idx_playlist_songs_playlist_position ON playlist_songs(playlist_id, position);
CREATE INDEX idx_job_queue_status_created ON job_queue(status, created_at) WHERE status = 'pending';
CREATE INDEX idx_songs_cover_path ON songs(cover_path) WHERE cover_path IS NOT NULL;
```

## 5. Cover Art System

### Architecture Overview

Korus implements a comprehensive cover art extraction and serving system that automatically discovers, processes, and serves album and song artwork without requiring separate API endpoints.

### Cover Art Sources (Priority Order)

1. **Song-Specific Covers**
   - Files matching audio filename: `song.jpg`, `track01.webp`
   - Highest priority for individual songs
   - Allows different artwork per track

2. **Embedded Cover Art**
   - Extracted from audio file metadata using `dhowden/tag`
   - Supports JPEG, PNG, WebP formats
   - Automatically extracted during metadata scanning

3. **Album Folder Covers**
   - Common filenames: `cover.jpg`, `folder.webp`, `albumart.png`
   - Shared across all songs in the album
   - Fallback for songs without specific covers

### File Format Support

- **JPEG** (`.jpg`, `.jpeg`) - Most common format
- **PNG** (`.png`) - High quality, transparency support
- **WebP** (`.webp`) - Modern format, excellent compression
- **GIF** (`.gif`) - Legacy support

### Storage and Serving

```
static/covers/
├── a1b2c3d4e5f6.jpg    # Content-based hash naming
├── f1e2d3c4b5a6.webp   # Prevents duplicates
└── b5c4d3e2f1a6.png    # Multiple formats supported
```

**Static File Serving:**
- **Endpoint**: `/covers/{hash}.{ext}`
- **Caching**: Browser-friendly caching headers
- **Performance**: Direct file serving via Gin static middleware

### Database Schema

**Songs Table:**
```sql
ALTER TABLE songs ADD COLUMN cover_path VARCHAR(255);
CREATE INDEX idx_songs_cover_path ON songs(cover_path) WHERE cover_path IS NOT NULL;
```

**Albums Table:**
```sql
-- Already exists
cover_path VARCHAR(255)
```

### API Integration

Cover URLs are embedded directly in JSON responses:

```json
{
  "id": 1,
  "title": "Come Together",
  "cover_path": "/covers/d4e5f6a1b2c3.jpg",
  "album": {
    "id": 1,
    "name": "Abbey Road",
    "cover_path": "/covers/f1e2d3c4b5a6.jpg"
  }
}
```

### Extraction Process

1. **During File Scan**: Audio files trigger metadata extraction jobs
2. **Cover Detection**: System tries all sources in priority order
3. **Deduplication**: Content hashing prevents duplicate storage
4. **URL Generation**: Relative URLs stored in database
5. **Static Serving**: Files served directly via `/covers/` endpoint

### Performance Optimizations

- **Content-Based Hashing**: Eliminates duplicate cover storage
- **Conditional Indexing**: Only indexes rows with covers
- **Static File Serving**: No database queries for cover requests
- **Browser Caching**: Long-term caching headers for covers

### Fallback Strategy

```
Song Cover Flow:
1. Check for song-specific cover file
2. Extract from embedded metadata
3. Use album folder cover
4. Return null (no cover)
```

## 6. User Experience

### Initial Setup
1.  **Environment Preparation**:
    ```bash
    mkdir -p korus/{data/{postgres,cache},music}
    cd korus
    ```

2.  **docker-compose.yml**:
    ```yaml
    version: '3.8'
    services:
      postgres:
        image: postgres:15-alpine
        environment:
          POSTGRES_DB: korus
          POSTGRES_USER: korus
          POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
        volumes:
          - ./data/postgres:/var/lib/postgresql/data

      korus:
        image: korus:latest
        environment:
          DATABASE_URL: postgres://korus:${POSTGRES_PASSWORD}@postgres/korus
          JWT_SECRET: ${JWT_SECRET}
          ADMIN_USERNAME: admin
          MUSIC_DIR: /music
          CACHE_DIR: /cache
        volumes:
          - ./music:/music:ro
          - ./data/cache:/cache
        ports:
          - "3000:3000"
        depends_on:
          - postgres
    ```

3.  **First Run**:
    On the very first launch, Korus will automatically create an administrator account if no users exist in the database. This eliminates manual setup steps and default password vulnerabilities.

    ```bash
    # Generate required secrets
    export POSTGRES_PASSWORD=$(openssl rand -base64 32)
    export JWT_SECRET=$(openssl rand -base64 32)

    # Start services
    docker-compose up -d
    ```

    After starting, check the container's logs to retrieve the secure, randomly generated password for the admin account.

    ```bash
    docker-compose logs korus
    ```

    The log output will contain a message similar to this:
    ```
    korus_1  | ====================KORUS INITIAL SETUP====================
    korus_1  | ADMIN ACCOUNT CREATED:
    korus_1  | Username: admin
    korus_1  | Password: aB3$fK9!pL2@zQ7x (Securely generated - change immediately)
    korus_1  | ==========================================================
    ```
    You can now log in with these credentials.

### Admin Functions
- **User Management**: Create/manage users via future `/api/users` endpoints
- **Library Scanning**: Trigger via `POST /api/library/scan`
- **System Monitoring**: View scan history and system stats via `/api/admin/status`

### File Watching
- Automatic detection of file changes using `fsnotify`
- Configurable debounce period (default: 5s) to handle rapid changes
- Background processing of new/modified files via the internal job queue

## 6. Operational Features

### Background Jobs
Background jobs are managed by a robust, transactional job queue built directly into PostgreSQL, eliminating external dependencies and ensuring data integrity.

| Job Type | Description | Schedule | Dependencies |
|----------|-------------|----------|--------------|
| Full Scan | Complete library reindexing | Manual/Periodic | File system |
| Incremental Scan | Process detected changes | Real-time | fsnotify |
| Metadata Enrichment | Fetch additional metadata | On-demand | MusicBrainz API |
| Thumbnail Generation | Create artwork variants | On-demand | ImageMagick |
| Session Cleanup | Remove expired sessions | Hourly | Database |
| Stats Aggregation | Update user statistics | Daily | Play history |

### Privacy Controls
- **User Visibility**: Global setting to hide user from public discovery
- **Scrobble Privacy**: Per-user setting to make listening history private
- **Playlist Sharing**: Public playlists accessible via UUID-based URLs
- **Data Export**: User can download all their data (playlists, likes, history)

### Performance Optimizations
- **Connection Pooling**: pgx with 10-20 connections per instance
- **Scan Concurrency**: Configurable worker pool (default: 4)
- **Metadata Cache**: In-memory LRU cache for frequently accessed items
- **HTTP/2 Support**: Multiplexed requests for improved client performance
- **Artwork Caching**: Disk-based caching of resized images

## 7. Post-MVP Roadmap

### Phase 1: Core Enhancements
1. **Advanced Permissions**
   - Role-based access control (RBAC)
   - Granular permissions (read-only, playlist-only, etc.)
   - Shared library access

2. **Smart Playlists**
   - Rule-based auto-updating playlists
   - Dynamic criteria (date added, play count, genre)
   - Playlist templates

### Phase 2: Media Handling
1. **Transcoding**
   - On-the-fly format conversion (FLAC→MP3, etc.)
   - Bitrate adaptation based on client
   - Caching of transcoded files

2. **Radio Stations**
   - Artist-based infinite streams
   - Genre/mood-based stations
   - Personalized radio

### Phase 3: Ecosystem Expansion
1. **Federation**
   - ActivityPub protocol support
   - Cross-instance library sharing
   - Federated playlists

2. **Mobile Sync**
   - Offline playback API
   - Sync queue management
   - Bandwidth optimization

### Phase 4: Intelligence & Integration
1. **Recommendations**
   - Collaborative filtering
   - Content-based recommendations
   - "Discover Weekly" playlists

2. **Import Tools**
   - Spotify library import
   - Last.fm history import
   - Subsonic migration tool

3. **Bundled Web UI**
   - React-based single-page application
   - Progressive Web App (PWA) support
   - Mobile-responsive design
