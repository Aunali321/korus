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

#### POST /auth/register
Register a new user account.

**Request Body:**
```json
{
  "username": "newuser",
  "password": "secure_password_123",
  "email": "user@example.com"
}
```

**Success Response (201):**
```json
{
  "message": "User created successfully",
  "user": {
    "id": 2,
    "username": "newuser",
    "email": "user@example.com",
    "role": "user",
    "created_at": "2025-08-01T16:00:00Z"
  }
}
```

**Error Response (400):**
```json
{
  "error": "validation_failed",
  "message": "Username already exists"
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
    "sortName": "Beatles, The",
    "musicbrainzId": "b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d",
    "albumCount": 12,
    "songCount": 147
  },
  {
    "id": 2,
    "name": "Led Zeppelin",
    "sortName": "Led Zeppelin",
    "musicbrainzId": null,
    "albumCount": 8,
    "songCount": 94
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
  "sortName": "Beatles, The",
  "musicbrainzId": "b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d",
  "albumCount": 12,
  "songCount": 147,
  "albums": [
    {
      "id": 1,
      "name": "Abbey Road",
      "year": 1969,
      "coverPath": "/covers/f1e2d3c4b5a6.jpg",
      "songCount": 17,
      "duration": 2854
    },
    {
      "id": 2,
      "name": "Let It Be",
      "year": 1970,
      "coverPath": "/covers/a6b5c4d3e2f1.jpg",
      "songCount": 12,
      "duration": 2156
    },
    {
      "id": 3,
      "name": "Sgt. Pepper's Lonely Hearts Club Band",
      "year": 1967,
      "coverPath": "/covers/b1c2d3e4f5a6.webp",
      "songCount": 13,
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
    "artistId": 1,
    "albumArtistId": 1,
    "year": 1969,
    "musicbrainzId": "7add7441-8f2c-4fbb-828d-0db9c0c2d43b",
    "coverPath": "/covers/f1e2d3c4b5a6.jpg",
    "dateAdded": "2025-08-01T10:00:00Z",
    "artist": {
      "id": 1,
      "name": "The Beatles"
    },
    "songCount": 17,
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
  "artistId": 1,
  "albumArtistId": 1,  
  "year": 1969,
  "musicbrainzId": "7add7441-8f2c-4fbb-828d-0db9c0c2d43b",
  "coverPath": "/covers/f1e2d3c4b5a6.jpg",
  "dateAdded": "2025-08-01T10:00:00Z",
  "artist": {
    "id": 1,
    "name": "The Beatles"
  },
  "albumArtist": {
    "id": 1,
    "name": "The Beatles"
  },
  "songCount": 17,
  "duration": 2854,
  "songs": [
    {
      "id": 1,
      "title": "Come Together",
      "albumId": 1,
      "artistId": 1,
      "trackNumber": 1,
      "discNumber": 1,
      "duration": 259,
      "filePath": "/music/The Beatles/Abbey Road/01 Come Together.mp3",
      "fileSize": 6234567,
      "fileModified": "2025-07-15T14:30:00Z",
      "bitrate": 320,
      "format": "mp3",
      "coverPath": "/covers/d4e5f6a1b2c3.jpg",
      "dateAdded": "2025-08-01T10:00:00Z",
      "artist": {
        "id": 1,
        "name": "The Beatles"
      }
    },
    {
      "id": 2,
      "title": "Something",
      "albumId": 1,
      "artistId": 1,
      "trackNumber": 2,
      "discNumber": 1,
      "duration": 182,
      "filePath": "/music/The Beatles/Abbey Road/02 Something.mp3",
      "fileSize": 4567890,
      "fileModified": "2025-07-15T14:30:00Z",
      "bitrate": 320,
      "format": "mp3",
      "coverPath": null,
      "dateAdded": "2025-08-01T10:00:00Z",
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
    "albumId": 1,
    "artistId": 1,
    "trackNumber": 1,
    "discNumber": 1,
    "duration": 259,
    "filePath": "/music/The Beatles/Abbey Road/01 Come Together.mp3",
    "fileSize": 6234567,
    "fileModified": "2025-07-15T14:30:00Z",
    "bitrate": 320,
    "format": "mp3",
    "coverPath": "/covers/d4e5f6a1b2c3.jpg",
    "dateAdded": "2025-08-01T10:00:00Z",
    "artist": {
      "id": 1,
      "name": "The Beatles"
    },
    "album": {
      "id": 1,
      "name": "Abbey Road"
    },
    "lyrics": [
      {
        "id": 1,
        "songId": 1,
        "content": "Come together right now over me...",
        "type": "unsynced",
        "source": "embedded",
        "language": "eng",
        "createdAt": "2025-08-01T10:00:00Z"
      },
      {
        "id": 2,
        "songId": 1,
        "content": "{\"metadata\":{\"title\":\"Come Together\",\"artist\":\"The Beatles\",\"album\":\"Abbey Road\",\"language\":\"eng\"},\"lines\":[{\"time\":1234,\"timeStr\":\"[00:01.23]\",\"text\":\"Come together\"},{\"time\":5678,\"timeStr\":\"[00:05.67]\",\"text\":\"Right now over me\"}]}",
        "type": "synced",
        "source": "external_lrc",
        "language": "eng",
        "createdAt": "2025-08-01T10:00:00Z"
      },
      {
        "id": 3,
        "songId": 1,
        "content": "Ven juntos ahora sobre mí...",
        "type": "unsynced",
        "source": "external_txt",
        "language": "spa",
        "createdAt": "2025-08-01T10:00:00Z"
      }
    ]
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

**Success Response (200):** Same as individual song object above, which automatically includes all lyrics for all languages in the `lyrics` array.

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
      "albumId": 1,
      "artistId": 1,
      "trackNumber": 1,
      "discNumber": 1,
      "duration": 259,
      "filePath": "/music/The Beatles/Abbey Road/01 Come Together.mp3",
      "coverPath": "/covers/d4e5f6a1b2c3.jpg",
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
      "sortName": "Beatles, The"
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

## 🎵 Lyrics System

Korus automatically extracts and serves lyrics for songs from multiple sources. Lyrics are included directly in song API responses and available through dedicated endpoints.

### Lyrics Sources (in priority order)
1. **Embedded lyrics**: Extracted from audio file ID3 tags (USLT frames)
2. **External .lrc files**: Synchronized lyrics with timestamps (`songname.lrc`)
3. **External .txt files**: Plain text lyrics (`songname.txt`)

### Lyrics Types
- **`unsynced`**: Plain text lyrics without timestamps
- **`synced`**: LRC format lyrics with precise timing information

### Language Detection
- **LRC files**: Uses `[la:language]` metadata tags when available
- **Content analysis**: Uses lingua-go library for automatic language detection from lyrics text
- **Supported languages**: English (`eng`), Arabic (`ara`), Urdu (`urd`), Hindi (`hin`), Spanish (`spa`), French (`fre`), German (`ger`), Japanese (`jpn`), Korean (`kor`), Chinese (`chi`), Portuguese (`por`), Italian (`ita`), Russian (`rus`)
- **Format**: ISO 639-2 language codes

### LRC Format Support
Korus includes a custom LRC parser that supports:
- **Metadata tags**: `[ti:title]`, `[ar:artist]`, `[al:album]`, `[by:creator]`, `[offset:±ms]`, `[length:mm:ss]`, `[la:language]`
- **Timestamp format**: `[mm:ss.xx]lyrics text`
- **JSON storage**: LRC data is converted to structured JSON for precise timing preservation
- **Auto-metadata filling**: Missing metadata (title, artist, album) is automatically populated from song information

## 🖼️ Cover Art System

Korus automatically extracts and serves cover art for both songs and albums. Cover images are included directly in API responses as URLs, eliminating the need for separate cover endpoints.

### Cover Art Sources (in priority order)
1. **Song-specific covers**: `songname.jpg`, `songname.webp`, etc.
2. **Embedded cover art**: Extracted from audio file metadata
3. **Album folder covers**: `cover.jpg`, `folder.webp`, `albumart.png`, etc.

### Supported Formats
- **JPEG** (`.jpg`, `.jpeg`)
- **PNG** (`.png`) 
- **WebP** (`.webp`)
- **GIF** (`.gif`)

### Cover URL Structure
Cover images are served as static files:
```
/covers/{hash}.{ext}
```

Example: `/covers/a1b2c3d4e5f6.webp`

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
        "totalConnections": 20,
        "idleConnections": 15,
        "acquiredConnections": 5,
        "constructingConnections": 0
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
- **Audio Metadata**: dhowden/tag library for tags, FFprobe for accurate duration/bitrate

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
    "userId": 1,
    "visibility": "public",
    "createdAt": "2025-08-01T10:00:00Z",
    "updatedAt": "2025-08-01T15:30:00Z",
    "songCount": 25,
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
  "userId": 1,
  "visibility": "private",
  "createdAt": "2025-08-01T16:00:00Z",
  "updatedAt": "2025-08-01T16:00:00Z",
  "songCount": 0,
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
  "userId": 1,
  "visibility": "public",
  "createdAt": "2025-08-01T10:00:00Z",
  "updatedAt": "2025-08-01T15:30:00Z",
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

#### PUT /playlists/{id}/songs/reorder
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

#### GET /me/library/songs
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
    }
  }
]
```

#### GET /me/library/albums
Get user's liked albums.

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:** Same as liked songs

**Success Response (200):** Array of album objects with `liked_at` timestamp

#### GET /me/library/artists
Get user's followed artists.

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:** Same as liked songs

**Success Response (200):** Array of artist objects with `followed_at` timestamp

#### POST /songs/like
Like songs.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "songIds": [1, 2, 3]
}
```

**Success Response (204):** No content

#### DELETE /songs/like
Unlike songs.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "songIds": [1, 2]
}
```

**Success Response (204):** No content

#### POST /albums/{id}/like
Like an album.

**Headers:** `Authorization: Bearer <token>`

**Success Response (204):** No content

#### DELETE /albums/{id}/like
Unlike an album.

**Headers:** `Authorization: Bearer <token>`

**Success Response (204):** No content

#### POST /artists/{id}/follow
Follow an artist.

**Headers:** `Authorization: Bearer <token>`

**Success Response (204):** No content

#### DELETE /artists/{id}/follow
Unfollow an artist.

**Headers:** `Authorization: Bearer <token>`

**Success Response (204):** No content

### 📊 History & Statistics

#### POST /me/history/scrobble
Record a play event (scrobble).

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "songId": 1,
  "playedAt": "2025-08-01T16:00:00Z",
  "playDuration": 180
}
```

**Success Response (201):**
```json
{
  "message": "Play recorded successfully"
}
```

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
    "songId": 1,
    "playedAt": "2025-08-01T16:00:00Z",
    "durationPlayed": 180,
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
  "totalPlays": 1234,
  "totalListeningTime": 98765,
  "uniqueSongs": 456,
  "uniqueArtists": 89,
  "uniqueAlbums": 123,
  "topArtists": [
    {
      "id": 1,
      "name": "The Beatles",
      "playCount": 87
    }
  ],
  "topAlbums": [
    {
      "id": 1,
      "name": "Abbey Road",
      "artistName": "The Beatles",
      "playCount": 34
    }
  ],
  "topSongs": [
    {
      "id": 1,
      "title": "Come Together",
      "artistName": "The Beatles",
      "albumName": "Abbey Road",
      "playCount": 12
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
  "recentPlays": [],
  "recommendedSongs": [],
  "recommendedAlbums": [],
  "recentlyAdded": []
}
```

### 🔧 Admin

All admin endpoints require admin role.

#### POST /library/scan
Trigger an asynchronous library scan. Returns immediately with a job ID for tracking.

**Headers:** `Authorization: Bearer <token>` (admin required)

**Query Parameters:**
- `force` (default: false) - Force re-scan of all files even if unchanged

**Success Response (202 Accepted):**
```json
{
  "message": "Library scan started",
  "jobId": "550e8400-e29b-41d4-a716-446655440000",
  "force": false
}
```

**Error Response (409 Conflict):**
```json
{
  "error": "scan_in_progress",
  "message": "A library scan is already running"
}
```

#### GET /library/scan/{id}
Get the status of a library scan job by ID.

**Headers:** `Authorization: Bearer <token>` (admin required)

**Success Response (200):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "running",
  "phase": "ingesting",
  "progress": 150,
  "total": 500,
  "force": false,
  "startedAt": "2025-08-01T16:00:00Z"
}
```

**Completed Response (200):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "phase": "complete",
  "progress": 500,
  "total": 500,
  "force": false,
  "startedAt": "2025-08-01T16:00:00Z",
  "completedAt": "2025-08-01T16:05:30Z",
  "result": {
    "startedAt": "2025-08-01T16:00:00Z",
    "completedAt": "2025-08-01T16:05:30Z",
    "duration": "5m30s",
    "filesDiscovered": 1250,
    "filesQueued": 500,
    "filesNew": 25,
    "filesUpdated": 5,
    "filesRemoved": 2,
    "ingested": 30,
    "errors": []
  }
}
```

**Job Status Values:**
- `pending` - Job created but not yet started
- `running` - Scan in progress
- `completed` - Scan finished successfully
- `failed` - Scan failed with error

**Phase Values:**
- `initializing` - Job starting
- `discovering` - Walking file system
- `analyzing` - Comparing with existing songs
- `ingesting` - Processing new/updated files
- `complete` - Scan finished

**Error Response (404):**
```json
{
  "error": "job_not_found",
  "message": "Scan job not found"
}
```

#### GET /admin/status
Get system status.

**Headers:** `Authorization: Bearer <token>` (admin required)

**Success Response (200):**
```json
{
  "library": {
    "songs": 1250,
    "albums": 156,
    "artists": 89,
    "users": 5,
    "playlists": 12,
    "totalDuration": 18750
  },
  "indexer": {
    "running": false,
    "lastRun": {
      "startedAt": "2025-08-01T10:00:00Z",
      "completedAt": "2025-08-01T10:05:30Z",
      "duration": "5m30s",
      "filesDiscovered": 1250,
      "ingested": 30
    },
    "lastError": null
  },
  "recentScans": [...],
  "activity": {
    "plays24h": 150,
    "plays7d": 892,
    "plays30d": 3200,
    "totalPlays": 12500,
    "activeUsers30d": 3
  }
}
```

#### GET /admin/scans
Get recent scan history.

**Headers:** `Authorization: Bearer <token>` (admin required)

**Query Parameters:**
- `limit` (default: 10, max: 50) - Number of results

**Success Response (200):**
```json
[
  {
    "id": 15,
    "startedAt": "2025-08-01T10:00:00Z",
    "completedAt": "2025-08-01T10:05:30Z",
    "songsAdded": 25,
    "songsUpdated": 5,
    "songsRemoved": 2
  }
]
```

#### DELETE /admin/sessions/cleanup
Clean up expired user sessions.

**Headers:** `Authorization: Bearer <token>` (admin required)

**Success Response (200):**
```json
{
  "message": "Sessions cleanup completed",
  "deletedCount": 25
}
```

---

For more information, see the [project repository](https://github.com/your-org/korus) and [DESIGN.md](./DESIGN.md) for architectural details.