-- Migration: 001_initial_schema.sql
-- Description: Create initial database schema for Korus music server
-- Date: 2025-08-01

BEGIN;

-- Users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login TIMESTAMPTZ
);

-- Artists table
CREATE TABLE artists (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    sort_name VARCHAR(255),
    musicbrainz_id VARCHAR(255) UNIQUE
);

-- Albums table
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

-- Songs table
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

-- Playlists table
CREATE TABLE playlists (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    visibility VARCHAR(10) NOT NULL DEFAULT 'private',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Playlist songs table
CREATE TABLE playlist_songs (
    id SERIAL PRIMARY KEY,
    playlist_id INTEGER NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
    song_id INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Liked songs table
CREATE TABLE liked_songs (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    song_id INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    liked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, song_id)
);

-- Liked albums table
CREATE TABLE liked_albums (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    album_id INTEGER NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
    liked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, album_id)
);

-- Followed artists table
CREATE TABLE followed_artists (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    artist_id INTEGER NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    followed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, artist_id)
);

-- Play history table
CREATE TABLE play_history (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    song_id INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    played_at TIMESTAMPTZ NOT NULL,
    play_duration INTEGER,
    ip_address INET
);

-- Scan history table
CREATE TABLE scan_history (
    id SERIAL PRIMARY KEY,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    songs_added INTEGER NOT NULL DEFAULT 0,
    songs_updated INTEGER NOT NULL DEFAULT 0,
    songs_removed INTEGER NOT NULL DEFAULT 0
);

-- User sessions table
CREATE TABLE user_sessions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Job queue table
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

-- Lyrics table
CREATE TABLE lyrics (
    id SERIAL PRIMARY KEY,
    song_id INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'unsynced', -- 'synced' or 'unsynced'
    source VARCHAR(50) NOT NULL, -- 'embedded', 'external_lrc', 'external_txt'
    language VARCHAR(3) DEFAULT 'eng', -- ISO 639-2 language codes
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(song_id, language, type)
);

-- Critical indexes for performance
CREATE INDEX idx_songs_album_id ON songs(album_id);
CREATE INDEX idx_songs_artist_id ON songs(artist_id);
CREATE INDEX idx_songs_file_path ON songs(file_path);
CREATE INDEX idx_songs_cover_path ON songs(cover_path) WHERE cover_path IS NOT NULL;
CREATE INDEX idx_liked_songs_user_id ON liked_songs(user_id);
CREATE INDEX idx_liked_songs_liked_at ON liked_songs(liked_at);
CREATE INDEX idx_play_history_user_id ON play_history(user_id);
CREATE INDEX idx_play_history_played_at ON play_history(played_at);
CREATE INDEX idx_playlist_songs_playlist_position ON playlist_songs(playlist_id, position);
CREATE INDEX idx_job_queue_status_created ON job_queue(status, created_at) WHERE status = 'pending';
CREATE INDEX idx_lyrics_song_id ON lyrics(song_id);
CREATE INDEX idx_lyrics_type ON lyrics(type);
CREATE INDEX idx_lyrics_source ON lyrics(source);

-- Add unique constraint for artists by name
CREATE UNIQUE INDEX idx_artists_name_unique ON artists(LOWER(name));

-- Add unique constraint for albums by name and artist
CREATE UNIQUE INDEX idx_albums_name_artist_unique ON albums(LOWER(name), artist_id);

COMMIT;