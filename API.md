# Korus Music Server API Documentation

## Overview

The Korus API is a RESTful web service that provides access to a self-hosted music streaming server. It supports JWT authentication, library management, search functionality, and audio streaming with range request support.

**Base URL**: `http://localhost:3000/api`

### Contextual Wrapper Pattern

The API follows a consistent "Contextual Wrapper" pattern for primary resource endpoints:

- `GET /albums/{id}` returns album details **with** nested songs array
- `GET /artists/{id}` returns artist details **with** nested albums and top tracks
- `GET /playlists/{id}` returns playlist details **with** nested songs array

This design eliminates the need for multiple API calls when building complete views (Album Page, Artist Page, Playlist View), resulting in better performance and simpler client code.

## Authentication

Korus uses JWT (JSON Web Tokens) for authentication with access and refresh token pairs.

### Authentication Flow

1. **Login** - Exchange credentials for tokens
2. **Access** - Use access token for API requests
3. **Refresh** - Use refresh token to get new access token
4. **Logout** - Invalidate refresh token

### Headers

All protected endpoints require the following header:
```
Authorization: Bearer <access_token>
```

## API Endpoints

### 🔐 Authentication

#### POST /auth/login
Authenticate user and receive tokens.

**Request Body:**
```json
{
  "username": "admin",
  "password": "your_password"
}
```

**Success Response (200):**
```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refreshToken": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4...",
  "expiresAt": "2025-08-01T16:00:00Z",
  "user": {
    "id": 1,
    "username": "admin",
    "email": null,
    "role": "admin"
  }
}
```

**Error Response (401):**
```json
{
  "error": "authentication_failed",
  "message": "Invalid username or password"
}
```

#### POST /auth/refresh
Refresh access token using refresh token.

**Request Body:**
```json
{
  "refreshToken": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4..."
}
```

**Success Response (200):**
```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresAt": "2025-08-01T16:15:00Z"
}
```

#### POST /auth/logout
Invalidate refresh token.

**Request Body:**
```json
{
  "refreshToken": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4..."
}
```

**Success Response (204):** No content

#### GET /me
Get current user information.

**Headers:** `Authorization: Bearer <token>`

**Success Response (200):**
```json
{
  "id": 1,
  "username": "admin",
  "email": null,
  "role": "admin",
  "createdAt": "2025-08-01T10:00:00Z",
  "lastLogin": "2025-08-01T15:30:00Z"
}
```

### 📊 Library Statistics

#### GET /library/stats
Get library statistics.

**Headers:** `Authorization: Bearer <token>`

**Success Response (200):**
```json
{
  "totalSongs": 1250,
  "totalArtists": 89,
  "totalAlbums": 156,
  "totalDuration": 18750
}
```

### 🎤 Artists

#### GET /artists
List artists with pagination and sorting.

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:**
- `limit` (default: 50) - Number of results per page
- `offset` (default: 0) - Number of results to skip
- `sort` - Sorting method: `name`, `name_desc`, `albums`, `songs`

**Example:** `GET /artists?limit=10&offset=0&sort=name`

**Success Response (200):**
```json
[
  {
    "id": 1,
    "name": "The Beatles",
    "sort_name": "Beatles, The",
    "musicbrainz_id": "b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d",
    "album_count": 12,
    "song_count": 147
  },
  {
    "id": 2,
    "name": "Led Zeppelin",
    "sort_name": "Led Zeppelin",
    "musicbrainz_id": null,
    "album_count": 8,
    "song_count": 94
  }
]
```

**Empty Library Response (200):**
```json
[]
```

#### GET /artists/{id}
Get artist details by ID, including all albums and top tracks.

**Headers:** `Authorization: Bearer <token>`

**Success Response (200):**
```json
{
  "id": 1,
  "name": "The Beatles",
  "sort_name": "Beatles, The",
  "musicbrainz_id": "b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d",
  "album_count": 12,
  "song_count": 147,
  "albums": [
    {
      "id": 1,
      "name": "Abbey Road",
      "year": 1969,
      "cover_path": "/covers/abbey_road.jpg",
      "song_count": 17,
      "duration": 2854
    },
    {
      "id": 2,
      "name": "Let It Be",
      "year": 1970,
      "cover_path": "/covers/let_it_be.jpg",
      "song_count": 12,
      "duration": 2156
    },
    {
      "id": 3,
      "name": "Sgt. Pepper's Lonely Hearts Club Band",
      "year": 1967,
      "cover_path": "/covers/sgt_peppers.jpg",
      "song_count": 13,
      "duration": 2387
    }
  ],
  "topTracks": [
    {
      "id": 1,
      "title": "Come Together",
      "duration": 259,
      "album": {
        "id": 1,
        "name": "Abbey Road"
      }
    },
    {
      "id": 15,
      "title": "Hey Jude",
      "duration": 431,
      "album": null
    },
    {
      "id": 8,
      "title": "Let It Be",
      "duration": 243,
      "album": {
        "id": 2,
        "name": "Let It Be"
      }
    },
    {
      "id": 22,
      "title": "Yesterday",
      "duration": 125,
      "album": {
        "id": 4,
        "name": "Help!"
      }
    }
  ]
}
```

### 💿 Albums

#### GET /albums
List albums with pagination, sorting, and filtering.

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:**
- `limit` (default: 50) - Number of results per page
- `offset` (default: 0) - Number of results to skip
- `sort` - Sorting: `name`, `name_desc`, `year`, `year_desc`, `artist`, `date_added`
- `year` - Filter by release year

**Example:** `GET /albums?year=1969&sort=name&limit=20`

**Success Response (200):**
```json
[
  {
    "id": 1,
    "name": "Abbey Road",
    "artist_id": 1,
    "album_artist_id": 1,
    "year": 1969,
    "musicbrainz_id": "7add7441-8f2c-4fbb-828d-0db9c0c2d43b",
    "cover_path": "/covers/abbey_road.jpg",
    "date_added": "2025-08-01T10:00:00Z",
    "artist": {
      "id": 1,
      "name": "The Beatles"
    },
    "song_count": 17,
    "duration": 2854
  }
]
```

**Empty Library Response (200):**
```json
[]
```

#### GET /albums/{id}
Get album details by ID, including all songs in the album.

**Headers:** `Authorization: Bearer <token>`

**Success Response (200):**
```json
{
  "id": 1,
  "name": "Abbey Road",
  "artist_id": 1,
  "album_artist_id": 1,  
  "year": 1969,
  "musicbrainz_id": "7add7441-8f2c-4fbb-828d-0db9c0c2d43b",
  "cover_path": "/covers/abbey_road.jpg",
  "date_added": "2025-08-01T10:00:00Z",
  "artist": {
    "id": 1,
    "name": "The Beatles"
  },
  "album_artist": {
    "id": 1,
    "name": "The Beatles"
  },
  "song_count": 17,
  "duration": 2854,
  "songs": [
    {
      "id": 1,
      "title": "Come Together",
      "album_id": 1,
      "artist_id": 1,
      "track_number": 1,
      "disc_number": 1,
      "duration": 259,
      "file_path": "/music/The Beatles/Abbey Road/01 Come Together.mp3",
      "file_size": 6234567,
      "file_modified": "2025-07-15T14:30:00Z",
      "bitrate": 320,
      "format": "mp3",
      "date_added": "2025-08-01T10:00:00Z",
      "artist": {
        "id": 1,
        "name": "The Beatles"
      }
    },
    {
      "id": 2,
      "title": "Something",
      "album_id": 1,
      "artist_id": 1,
      "track_number": 2,
      "disc_number": 1,
      "duration": 182,
      "file_path": "/music/The Beatles/Abbey Road/02 Something.mp3",
      "file_size": 4567890,
      "file_modified": "2025-07-15T14:30:00Z",
      "bitrate": 320,
      "format": "mp3",
      "date_added": "2025-08-01T10:00:00Z",
      "artist": {
        "id": 1,
        "name": "The Beatles"
      }
    }
  ]
}
```


### 🎵 Songs

#### GET /songs
List all songs with pagination and sorting, or batch fetch songs by IDs.

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:**
- `ids` (optional) - Comma-separated list of song IDs for batch fetch
- `limit` (default: 50) - Number of results per page (when not using ids)
- `offset` (default: 0) - Number of results to skip (when not using ids)
- `sort` - Sorting method: `title`, `title_desc`, `artist`, `album`, `duration`, `duration_desc`, `date_added`

**Examples:** 
- `GET /songs` - List all songs with default pagination
- `GET /songs?limit=20&sort=artist` - List songs sorted by artist
- `GET /songs?ids=1,2,3,4,5` - Batch fetch specific songs

**Success Response (200):**
```json
[
  {
    "id": 1,
    "title": "Come Together",
    "album_id": 1,
    "artist_id": 1,
    "track_number": 1,
    "disc_number": 1,
    "duration": 259,
    "file_path": "/music/The Beatles/Abbey Road/01 Come Together.mp3",
    "file_size": 6234567,
    "file_modified": "2025-07-15T14:30:00Z",
    "bitrate": 320,
    "format": "mp3",
    "date_added": "2025-08-01T10:00:00Z",
    "artist": {
      "id": 1,
      "name": "The Beatles"
    },
    "album": {
      "id": 1,
      "name": "Abbey Road"
    }
  }
]
```

**Error Response (404):**
```json
{
  "error": "not_found",
  "message": "No songs found for the requested IDs"
}
```

#### GET /songs/{id}
Get song details by ID.

**Headers:** `Authorization: Bearer <token>`

**Success Response (200):** Same as individual song object above.

### 🔍 Search

#### GET /search
Search across songs, albums, and artists.

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:**
- `q` (required) - Search query
- `type` (optional) - Filter by type: `song`, `album`, `artist`
- `limit` (default: 20) - Number of results per type
- `offset` (default: 0) - Number of results to skip

**Example:** `GET /search?q=come%20together&limit=5`

**Success Response (200):**
```json
{
  "songs": [
    {
      "id": 1,
      "title": "Come Together",
      "album_id": 1,
      "artist_id": 1,
      "track_number": 1,
      "disc_number": 1,
      "duration": 259,
      "file_path": "/music/The Beatles/Abbey Road/01 Come Together.mp3",
      "artist": {
        "id": 1,
        "name": "The Beatles"
      },
      "album": {
        "id": 1,
        "name": "Abbey Road"
      }
    }
  ],
  "albums": [],
  "artists": [
    {
      "id": 1,
      "name": "The Beatles",
      "sort_name": "Beatles, The"
    }
  ]
}
```

### 🎧 Streaming

#### GET /songs/{id}/stream
Stream audio file with range request support.

**Headers:** 
- `Authorization: Bearer <token>`
- `Range: bytes=0-1023` (optional, for partial content)

**Success Response (200 or 206):**
- Returns audio file with appropriate `Content-Type`
- Supports HTTP range requests for seeking
- Sets proper caching headers

**Response Headers:**
```
Content-Type: audio/mpeg
Accept-Ranges: bytes
Content-Length: 6234567
Last-Modified: Mon, 15 Jul 2025 14:30:00 GMT
Cache-Control: public, max-age=31536000
```

**Partial Content (206) with Range:**
```
Content-Range: bytes 0-1023/6234567
Content-Length: 1024
```

#### GET /albums/{id}/cover
Get album artwork.

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:**
- `size` (optional) - Thumbnail size (future feature)

**Success Response (200):**
- Returns image file (JPEG, PNG, etc.)
- Sets appropriate `Content-Type` header
- Includes caching headers

### ❤️ Health & Monitoring

#### GET /ping
Basic health check (no authentication required).

**Success Response (200):**
```json
{
  "status": "ok",
  "message": "Korus server is running"
}
```

#### GET /health
Detailed health status (no authentication required).

**Success Response (200):**
```json
{
  "status": "healthy",
  "checks": {
    "database": {
      "status": "healthy",
      "stats": {
        "total_connections": 20,
        "idle_connections": 15,
        "acquired_connections": 5,
        "constructing_connections": 0
      }
    }
  }
}
```

**Unhealthy Response (503):**
```json
{
  "status": "unhealthy",
  "checks": {
    "database": {
      "status": "unhealthy",
      "error": "connection timeout"
    }
  }
}
```

## Error Responses

All endpoints may return these common error responses:

### 400 Bad Request
```json
{
  "error": "invalid_request",
  "message": "Invalid request parameters"
}
```

### 401 Unauthorized
```json
{
  "error": "unauthorized",
  "message": "authentication required"
}
```

### 403 Forbidden
```json
{
  "error": "forbidden",
  "message": "insufficient permissions"
}
```

### 404 Not Found
```json
{
  "error": "not_found",
  "message": "resource not found"
}
```

### 429 Too Many Requests
```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests, please try again later"
}
```

### 500 Internal Server Error
```json
{
  "error": "internal_error",
  "message": "internal server error"
}
```

## Rate Limiting

Different endpoints have different rate limits:

- **Authentication endpoints**: 10 requests per minute per IP
- **Search endpoints**: 100 requests per minute per user
- **General API endpoints**: 1000 requests per hour per user

Rate limit headers are included in responses:
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1640995200
```

## Supported Audio Formats

Korus supports the following audio formats:

- **MP3** (`.mp3`) - MPEG-1 Audio Layer III
- **FLAC** (`.flac`) - Free Lossless Audio Codec
- **M4A/AAC** (`.m4a`, `.aac`) - Advanced Audio Coding
- **OGG Vorbis** (`.ogg`) - Open source audio format
- **WAV** (`.wav`) - Waveform Audio File Format

## Libraries Used

- **Web Framework**: Gin (Go)
- **Database**: PostgreSQL with pgx driver
- **Authentication**: JWT tokens
- **Search**: Bleve full-text search
- **File Watching**: fsnotify
- **Audio Metadata**: dhowden/tag library

## Client Examples

### JavaScript/Fetch
```javascript
// Login
const loginResponse = await fetch('/api/auth/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ username: 'admin', password: 'password' })
});
const { accessToken } = await loginResponse.json();

// Search
const searchResponse = await fetch('/api/search?q=beatles', {
  headers: { 'Authorization': `Bearer ${accessToken}` }
});
const results = await searchResponse.json();

// Stream audio
const audioUrl = `/api/songs/1/stream`;
const audio = new Audio(audioUrl);
audio.addEventListener('loadstart', () => {
  // Set auth header for audio requests
  audio.crossOrigin = 'use-credentials';
});
```

### cURL
```bash
# Login
curl -X POST http://localhost:3000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}'

# Get library stats
curl -H "Authorization: Bearer <token>" \
  http://localhost:3000/api/library/stats

# Search
curl -H "Authorization: Bearer <token>" \
  "http://localhost:3000/api/search?q=beatles&limit=5"

# Stream with range request
curl -H "Authorization: Bearer <token>" \
     -H "Range: bytes=0-1023" \
     http://localhost:3000/api/songs/1/stream
```

### 📋 Playlists

#### GET /playlists
List user's playlists.

**Headers:** `Authorization: Bearer <token>`

**Success Response (200):**
```json
[
  {
    "id": 1,
    "name": "My Favorites",
    "description": "Songs I love",
    "user_id": 1,
    "visibility": "public",
    "created_at": "2025-08-01T10:00:00Z",
    "updated_at": "2025-08-01T15:30:00Z",
    "song_count": 25,
    "duration": 5400
  }
]
```

#### POST /playlists
Create a new playlist.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "name": "My New Playlist",
  "description": "Optional description",
  "visibility": "private"
}
```

**Success Response (201):**
```json
{
  "id": 2,
  "name": "My New Playlist",
  "description": "Optional description",
  "user_id": 1,
  "visibility": "private",
  "created_at": "2025-08-01T16:00:00Z",
  "updated_at": "2025-08-01T16:00:00Z",
  "song_count": 0,
  "duration": 0
}
```

#### GET /playlists/{id}
Get playlist details with full, ordered tracklist.

**Headers:** `Authorization: Bearer <token>`

**Success Response (200):**
```json
{
  "id": 1,
  "name": "My Favorites",
  "description": "Songs I love",
  "user_id": 1,
  "visibility": "public",
  "created_at": "2025-08-01T10:00:00Z",
  "updated_at": "2025-08-01T15:30:00Z",
  "duration": 5400,
  "owner": {
    "id": 1,
    "username": "admin"
  },
  "songs": [
    {
      "playlistSongId": 101,
      "position": 0,
      "song": {
        "id": 1,
        "title": "Come Together",
        "duration": 259,
        "artist": {
          "id": 1,
          "name": "The Beatles"
        },
        "album": {
          "id": 1,
          "name": "Abbey Road"
        }
      }
    },
    {
      "playlistSongId": 102,
      "position": 1,
      "song": {
        "id": 8,
        "title": "Let It Be",
        "duration": 243,
        "artist": {
          "id": 1,
          "name": "The Beatles"
        },
        "album": {
          "id": 2,
          "name": "Let It Be"
        }
      }
    }
  ]
}
```

#### PUT /playlists/{id}
Update playlist details.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "name": "Updated Playlist Name",
  "description": "New description",
  "visibility": "public"
}
```

**Success Response (200):** Same as GET /playlists/{id}

#### DELETE /playlists/{id}
Delete a playlist.

**Headers:** `Authorization: Bearer <token>`

**Success Response (204):** No content

#### POST /playlists/{id}/songs
Add songs to playlist.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "songIds": [1, 2, 3]
}
```

**Success Response (204):** No content

#### DELETE /playlists/{id}/songs
Remove songs from playlist.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "playlistSongIds": [101, 102]
}
```

**Success Response (204):** No content

#### PUT /playlists/{id}/reorder
Reorder songs in playlist.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "playlistSongId": 101,
  "newPosition": 3
}
```

**Success Response (204):** No content

### ❤️ User Library

#### GET /me/library/liked/songs
Get user's liked songs.

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:**
- `limit` (default: 50) - Number of results per page
- `offset` (default: 0) - Number of results to skip

**Success Response (200):**
```json
[
  {
    "id": 1,
    "title": "Come Together",
    "album_id": 1,
    "artist_id": 1,
    "duration": 259,
    "artist": {
      "id": 1,
      "name": "The Beatles"
    },
    "album": {
      "id": 1,
      "name": "Abbey Road"
    },
    "liked_at": "2025-08-01T15:30:00Z"
  }
]
```

#### GET /me/library/liked/albums
Get user's liked albums.

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:** Same as liked songs

**Success Response (200):** Array of album objects with `liked_at` timestamp

#### GET /me/library/followed/artists
Get user's followed artists.

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:** Same as liked songs

**Success Response (200):** Array of artist objects with `followed_at` timestamp

#### POST /me/library/like/songs
Like songs.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "songIds": [1, 2, 3]
}
```

**Success Response (204):** No content

#### DELETE /me/library/unlike/songs
Unlike songs.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "songIds": [1, 2]
}
```

**Success Response (204):** No content

#### POST /me/library/like/albums/{id}
Like an album.

**Headers:** `Authorization: Bearer <token>`

**Success Response (204):** No content

#### DELETE /me/library/unlike/albums/{id}
Unlike an album.

**Headers:** `Authorization: Bearer <token>`

**Success Response (204):** No content

#### POST /me/library/follow/artists/{id}
Follow an artist.

**Headers:** `Authorization: Bearer <token>`

**Success Response (204):** No content

#### DELETE /me/library/unfollow/artists/{id}
Unfollow an artist.

**Headers:** `Authorization: Bearer <token>`

**Success Response (204):** No content

### 📊 History & Statistics

#### POST /me/history/play
Record a play event.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "song_id": 1,
  "played_at": "2025-08-01T16:00:00Z",
  "duration_played": 180,
  "completed": true
}
```

**Success Response (204):** No content

#### GET /me/history/recent
Get recent listening history.

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:**
- `limit` (default: 50) - Number of results
- `offset` (default: 0) - Number of results to skip

**Success Response (200):**
```json
[
  {
    "id": 1,
    "song_id": 1,
    "played_at": "2025-08-01T16:00:00Z",
    "duration_played": 180,
    "completed": true,
    "song": {
      "id": 1,
      "title": "Come Together",
      "artist": {
        "id": 1,
        "name": "The Beatles"
      },
      "album": {
        "id": 1,
        "name": "Abbey Road"
      }
    }
  }
]
```

#### GET /me/stats
Get user listening statistics.

**Headers:** `Authorization: Bearer <token>`

**Success Response (200):**
```json
{
  "total_plays": 1234,
  "total_listening_time": 98765,
  "unique_songs": 456,
  "unique_artists": 89,
  "unique_albums": 123,
  "top_artists": [
    {
      "id": 1,
      "name": "The Beatles",
      "play_count": 87
    }
  ],
  "top_albums": [
    {
      "id": 1,
      "name": "Abbey Road",
      "artist_name": "The Beatles",
      "play_count": 34
    }
  ],
  "top_songs": [
    {
      "id": 1,
      "title": "Come Together",
      "artist_name": "The Beatles",
      "album_name": "Abbey Road",
      "play_count": 12
    }
  ]
}
```

#### GET /me/home
Get personalized home data.

**Headers:** `Authorization: Bearer <token>`

**Success Response (200):**
```json
{
  "recent_plays": [],
  "recommended_songs": [],
  "recommended_albums": [],
  "recently_added": []
}
```

### 🔧 Admin

All admin endpoints require admin role.

#### POST /admin/library/scan
Trigger library scan.

**Headers:** `Authorization: Bearer <token>` (admin required)

**Success Response (202):**
```json
{
  "message": "Library scan started",
  "job_id": "scan_20250801_160000"
}
```

#### GET /admin/system/status
Get system status.

**Headers:** `Authorization: Bearer <token>` (admin required)

**Success Response (200):**
```json
{
  "system": {
    "uptime": 86400,
    "go_version": "go1.21.5",
    "goroutines": 25
  },
  "database": {
    "status": "healthy",
    "total_connections": 20,
    "idle_connections": 15
  },
  "job_queue": {
    "pending_jobs": 0,
    "active_workers": 3
  },
  "library": {
    "total_songs": 1250,
    "total_artists": 89,
    "total_albums": 156,
    "last_scan": "2025-08-01T10:00:00Z"
  }
}
```

#### GET /admin/scans/history
Get recent scan history.

**Headers:** `Authorization: Bearer <token>` (admin required)

**Query Parameters:**
- `limit` (default: 10) - Number of results

**Success Response (200):**
```json
[
  {
    "id": "scan_20250801_100000",
    "started_at": "2025-08-01T10:00:00Z",
    "completed_at": "2025-08-01T10:05:30Z",
    "status": "completed",
    "files_processed": 1250,
    "files_added": 25,
    "files_updated": 5,
    "files_removed": 2,
    "errors": 0
  }
]
```

#### POST /admin/cleanup/jobs
Clean up old jobs.

**Headers:** `Authorization: Bearer <token>` (admin required)

**Success Response (200):**
```json
{
  "message": "Cleanup completed",
  "jobs_removed": 150
}
```

#### POST /admin/cleanup/sessions
Clean up old sessions.

**Headers:** `Authorization: Bearer <token>` (admin required)

**Success Response (200):**
```json
{
  "message": "Cleanup completed",
  "sessions_removed": 25
}
```

---

For more information, see the [project repository](https://github.com/your-org/korus) and [DESIGN.md](./DESIGN.md) for architectural details.